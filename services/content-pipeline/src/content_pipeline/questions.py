"""Deterministic question candidates (§22, §23, §40).

Turns a mapped chunk into a reviewable question using a deterministic template —
no AI. §23 assigns the structured types (counting, matching, ordering, simple
arithmetic, recognition, pattern) to templates and reserves assisted generation
for scenario/comprehension; those need assets and a model, so they are M6.

The one template implemented here is a **source-grounded cloze**: a sentence from
the chunk loses its most salient term and the child restores it. Because the
answer is taken from the source text, automatic QA can verify it actually occurs
in the source (the "answer mismatch" check of §40). Early bands get a
``single_choice`` with distractors drawn from the same chunk; upper bands get a
``short_answer``. The payload must pass the shared question schema before it is
stored (§23, roadmap D2), and a reviewer remains the final authority (§24).
"""

from __future__ import annotations

import hashlib
import json
import logging
import re
from dataclasses import dataclass, replace
from typing import Any, Protocol

from content_pipeline.db import Database
from content_pipeline.figures import FigureVocab
from content_pipeline.mapping import REVIEW_CONFIDENCE, STOPWORDS
from content_pipeline.materials import build_learning_objective
from content_pipeline.qa import QaIssue, has_blocking
from content_pipeline.readability import analyse, difficulty_hint, sentences
from content_pipeline.taxonomy import SkillRef, Taxonomy, load_taxonomy
from content_pipeline.validation import QuestionSchema, SchemaValidation
from content_pipeline.visuals import (
    VisualPlanner,
    apply_visual,
    grounded_icons,
    visual_need,
    visual_quality_checks,
)

log = logging.getLogger("content_pipeline.questions")

TEMPLATE = "cloze"
BLANK = "_____"
CHOICE_LETTERS = "abcdef"
CHOICE_COUNT = 4
MIN_SENTENCE_WORDS = 6
MAX_SENTENCE_WORDS = 40
MIN_TERM_LEN = 5
PROMPT_MAX_WORDS = 60
FALLBACK_OBJECTIVE = "Peserta didik dapat memahami istilah pada bacaan ini."

# Curriculum phase → question type. Bands are advisory (§23); image_choice and
# tap_count need assets, so a text cloze is the honest deterministic choice.
PHASE_TYPE: dict[str, str] = {
    "fondasi": "single_choice",
    "a": "single_choice",
    "b": "short_answer",
    "c": "short_answer",
}
DEFAULT_TYPE = "single_choice"

# Picture-based templates for TK/Fase A (M10). They reuse the Go types
# tap_count and image_choice unchanged; a draft is only emitted when the source
# actually grounds the number and the object.
COUNT_TEMPLATE = "count_objects"
PICK_TEMPLATE = "pick_object"
PICK_DISTRACTORS = 3

NUMBER_WORDS: dict[str, int] = {
    "satu": 1, "dua": 2, "tiga": 3, "empat": 4, "lima": 5,
    "enam": 6, "tujuh": 7, "delapan": 8, "sembilan": 9, "sepuluh": 10,
}
COUNTING_HINTS = ("hitung", "banyak", "jumlah", "bilangan", "angka", "berhitung")
RECOGNITION_HINTS = ("kenal", "pilih", "tunjuk", "gambar", "bentuk", "warna", "buah", "hewan", "huruf")

_WORD = re.compile(r"[A-Za-z][A-Za-z-]*")
_WHITESPACE = re.compile(r"\s+")
_LOWER_WORD = re.compile(r"[a-z0-9]+")


@dataclass(frozen=True)
class QuestionDraft:
    template: str
    type: str
    difficulty: int
    learning_objective: str
    prompt: str
    payload: dict[str, Any]
    answer_text: str
    content_hash: str
    source_sentence: str
    term: str


def question_hash(prompt: str, answer_text: str) -> str:
    """Fingerprint a question by its prompt + answer for duplicate detection (§29)."""
    key = f"{_WHITESPACE.sub(' ', prompt).strip().lower()}|{answer_text.strip().lower()}"
    return hashlib.sha256(key.encode("utf-8")).hexdigest()


def _terms(sentence: str) -> list[str]:
    return [match.group(0) for match in _WORD.finditer(sentence)]


def content_words(text: str) -> list[str]:
    """Content words (long enough, not stopwords), deduped case-insensitively, ordered."""
    seen: set[str] = set()
    out: list[str] = []
    for word in _terms(text):
        lower = word.lower()
        if len(word) < MIN_TERM_LEN or lower in STOPWORDS or lower in seen:
            continue
        seen.add(lower)
        out.append(word)
    return out


def vocabulary(skill: SkillRef | None) -> set[str]:
    if skill is None:
        return set()
    return {word.lower() for word in _terms(f"{skill.name} {skill.description} {skill.chapter_name}")}


def pick_term(sentence: str, vocab: set[str]) -> str | None:
    """Pick the term to blank: prefer a curriculum term, else the longest content word."""
    candidates = content_words(sentence)
    if not candidates:
        return None
    preferred = [word for word in candidates if word.lower() in vocab]
    pool = preferred or candidates
    # Longest wins; ties resolved by earliest occurrence (max keeps the first).
    return max(pool, key=lambda word: len(word))


def blank_out(sentence: str, term: str) -> str | None:
    pattern = re.compile(rf"\b{re.escape(term)}\b", re.IGNORECASE)
    if not pattern.search(sentence):
        return None
    return pattern.sub(BLANK, sentence, count=1)


def pick_sentence(text: str, vocab: set[str]) -> tuple[str, str] | None:
    """First sentence that yields a blankable term; prefer one with a curriculum term."""
    best: tuple[str, str] | None = None
    for sentence in sentences(text):
        word_count = len(sentence.split())
        if not MIN_SENTENCE_WORDS <= word_count <= MAX_SENTENCE_WORDS:
            continue
        term = pick_term(sentence, vocab)
        if term is None:
            continue
        if term.lower() in vocab:
            return sentence, term
        if best is None:
            best = (sentence, term)
    return best


def _distractors(text: str, answer: str) -> list[str]:
    answer_lower = answer.lower()
    others = [word for word in content_words(text) if word.lower() != answer_lower]
    others.sort(key=lambda word: (-len(word), word.lower()))
    return others


def _option_order(answer: str, distractors: list[str]) -> tuple[list[str], int]:
    digest = hashlib.sha256(answer.lower().encode("utf-8")).hexdigest()
    index = int(digest[:8], 16) % (len(distractors) + 1)
    ordered = distractors[:index] + [answer] + distractors[index:]
    return ordered, index


def _build_payload(
    *,
    code: str,
    skill_code: str,
    question_type: str,
    difficulty: int,
    objective: str,
    prompt: str,
    answer: dict[str, Any],
    explanation: str,
    options: list[dict[str, str]] | None = None,
) -> dict[str, Any]:
    content: dict[str, Any] = {"prompt": prompt}
    if options is not None:
        content["options"] = options
    return {
        "code": code,
        "skill": skill_code,
        "type": question_type,
        "difficulty": difficulty,
        "learning_objective": objective,
        "content": content,
        "answer": answer,
        "explanation": explanation,
    }


def build_question(
    text: str,
    *,
    source_code: str,
    skill: SkillRef | None,
    phase_code: str = "",
    grade_code: str = "",
) -> QuestionDraft | None:
    """Pure, deterministic question draft. Returns None when the chunk cannot ground one."""
    vocab = vocabulary(skill)
    picked = pick_sentence(text, vocab)
    if picked is None:
        return None
    sentence, term = picked
    blanked = blank_out(sentence, term)
    if blanked is None:
        return None
    prompt = f"Lengkapi kalimat berikut: {blanked.rstrip() + '.'}"
    objective = (
        build_learning_objective(skill.name, skill.cp_code)
        if skill is not None and skill.name
        else FALLBACK_OBJECTIVE
    )
    readability = analyse(text, phase_code=phase_code, grade_code=grade_code)
    difficulty = difficulty_hint(readability)

    distractors = _distractors(text, term)
    question_type = PHASE_TYPE.get(phase_code, DEFAULT_TYPE)
    options: list[dict[str, str]] | None = None
    if question_type == "single_choice" and len(distractors) >= CHOICE_COUNT - 1:
        ordered, index = _option_order(term, distractors[: CHOICE_COUNT - 1])
        options = [
            {"id": CHOICE_LETTERS[i], "label": label} for i, label in enumerate(ordered)
        ]
        answer: dict[str, Any] = {"option_id": CHOICE_LETTERS[index]}
        answer_text = term
    else:
        question_type = "short_answer"
        answer = {"accepted": [term]}
        answer_text = term

    content_hash = question_hash(prompt, answer_text)
    code = f"Q-{source_code.upper().replace('_', '-')}-{content_hash[:8].upper()}"
    explanation = f"Jawaban: {term}. {sentence.rstrip() + '.'}"

    payload = _build_payload(
        code=code,
        skill_code=skill.code if skill is not None else "",
        question_type=question_type,
        difficulty=difficulty,
        objective=objective,
        prompt=prompt,
        answer=answer,
        explanation=explanation,
        options=options,
    )
    return QuestionDraft(
        template=TEMPLATE,
        type=question_type,
        difficulty=difficulty,
        learning_objective=objective,
        prompt=prompt,
        payload=payload,
        answer_text=answer_text,
        content_hash=content_hash,
        source_sentence=sentence,
        term=term,
    )


# ------------------------------------------------------- picture templates (M10)


def _lower_words(text: str) -> list[str]:
    return _LOWER_WORD.findall(text.lower())


def _mentions(hints: tuple[str, ...], *parts: str) -> bool:
    haystack = " ".join(parts).lower()
    return any(hint in haystack for hint in hints)


def _skill_text(skill: SkillRef | None) -> str:
    if skill is None:
        return ""
    return f"{skill.name} {skill.description} {skill.chapter_name}"


def find_count(text: str) -> tuple[int, str] | None:
    """First grounded count as ``(value, source_word)``. Pure."""
    for token in _lower_words(text):
        if token.isdigit():
            value = int(token)
            if 1 <= value <= 10:
                return value, token
        elif token in NUMBER_WORDS:
            return NUMBER_WORDS[token], token
    return None


def _choices_around(value: int) -> list[int]:
    """Two to five distinct positive choices that include the answer."""
    candidates = {max(1, value - 1), value, value + 1}
    if len(candidates) < 2:
        candidates.add(value + 2)
    return sorted(candidates)


def _ranked(items: list[str], *, seed: str) -> list[str]:
    """Deterministic ordering of items by a hash of the seed (no RNG)."""
    digest = hashlib.sha256(seed.encode("utf-8")).digest()
    return sorted(items, key=lambda item: hashlib.sha256(digest + item.encode("utf-8")).hexdigest())


def _objective(skill: SkillRef | None) -> str:
    if skill is not None and skill.name:
        return build_learning_objective(skill.name, skill.cp_code)
    return FALLBACK_OBJECTIVE


def _code(source_code: str, content_hash: str) -> str:
    return f"Q-{source_code.upper().replace('_', '-')}-{content_hash[:8].upper()}"


def build_count_question(
    text: str,
    *,
    source_code: str,
    skill: SkillRef | None,
    phase_code: str = "",
    grade_code: str = "",
    vocab: FigureVocab,
) -> QuestionDraft | None:
    """A ``tap_count`` question grounded in a number and an object in the source. Pure."""
    if not _mentions(COUNTING_HINTS, _skill_text(skill)):
        return None
    counted = find_count(text)
    icons = grounded_icons(text, vocab)
    if counted is None or not icons:
        return None
    value, word = counted
    icon = icons[0]
    label = vocab.label(icon)
    objective = _objective(skill)
    readability = analyse(text, phase_code=phase_code, grade_code=grade_code)
    difficulty = difficulty_hint(readability)
    prompt = f"Berapa banyak {label}? Ketuk angkanya."
    content: dict[str, Any] = {
        "prompt": prompt,
        "audio_text": prompt,
        "symbol": icon,
        "count": value,
        "color": vocab.default_color(icon),
        "choices": _choices_around(value),
    }
    content_hash = question_hash(prompt, word)
    payload: dict[str, Any] = {
        "code": _code(source_code, content_hash),
        "skill": skill.code if skill is not None else "",
        "type": "tap_count",
        "difficulty": difficulty,
        "learning_objective": objective,
        "content": content,
        "answer": {"value": value},
        "explanation": f"Jawaban: {value}. Ada {word} {label}.",
    }
    return QuestionDraft(
        template=COUNT_TEMPLATE,
        type="tap_count",
        difficulty=difficulty,
        learning_objective=objective,
        prompt=prompt,
        payload=payload,
        answer_text=word,
        content_hash=content_hash,
        source_sentence=text,
        term=word,
    )


def build_pick_question(
    text: str,
    *,
    source_code: str,
    skill: SkillRef | None,
    phase_code: str = "",
    grade_code: str = "",
    vocab: FigureVocab,
) -> QuestionDraft | None:
    """An ``image_choice`` recognition question grounded in an object in the source. Pure."""
    if not _mentions(RECOGNITION_HINTS, _skill_text(skill)):
        return None
    icons = grounded_icons(text, vocab)
    if not icons:
        return None
    target = icons[0]
    distractors = _ranked([icon for icon in vocab.icons if icon not in set(icons)], seed=text)[:PICK_DISTRACTORS]
    if not distractors:
        return None
    label = vocab.label(target)
    prompt = f"Pilih gambar {label}."
    objective = _objective(skill)
    readability = analyse(text, phase_code=phase_code, grade_code=grade_code)
    difficulty = difficulty_hint(readability)

    options: list[dict[str, Any]] = []
    answer_id = ""
    for index, icon in enumerate(_ranked([target, *distractors], seed=f"order:{text}")):
        option_id = CHOICE_LETTERS[index]
        # No visible label: the child must recognise the picture, not read the answer.
        options.append(
            {
                "id": option_id,
                "label": "",
                "figures": [{"icon": icon, "count": 1, "color": vocab.default_color(icon)}],
            }
        )
        if icon == target:
            answer_id = option_id

    content_hash = question_hash(prompt, label)
    payload: dict[str, Any] = {
        "code": _code(source_code, content_hash),
        "skill": skill.code if skill is not None else "",
        "type": "image_choice",
        "difficulty": difficulty,
        "learning_objective": objective,
        "content": {"prompt": prompt, "audio_text": prompt, "options": options},
        "answer": {"option_id": answer_id},
        "explanation": f"Jawaban: {label}.",
    }
    return QuestionDraft(
        template=PICK_TEMPLATE,
        type="image_choice",
        difficulty=difficulty,
        learning_objective=objective,
        prompt=prompt,
        payload=payload,
        answer_text=label,
        content_hash=content_hash,
        source_sentence=text,
        term=label,
    )


class _TemplateBuilder(Protocol):
    def __call__(
        self,
        text: str,
        *,
        source_code: str,
        skill: SkillRef | None,
        phase_code: str,
        grade_code: str,
    ) -> QuestionDraft | None: ...


def _count_builder(vocab: FigureVocab) -> _TemplateBuilder:
    def build(
        text: str,
        *,
        source_code: str,
        skill: SkillRef | None,
        phase_code: str,
        grade_code: str,
    ) -> QuestionDraft | None:
        return build_count_question(
            text, source_code=source_code, skill=skill, phase_code=phase_code, grade_code=grade_code, vocab=vocab
        )

    return build


def _pick_builder(vocab: FigureVocab) -> _TemplateBuilder:
    def build(
        text: str,
        *,
        source_code: str,
        skill: SkillRef | None,
        phase_code: str,
        grade_code: str,
    ) -> QuestionDraft | None:
        return build_pick_question(
            text, source_code=source_code, skill=skill, phase_code=phase_code, grade_code=grade_code, vocab=vocab
        )

    return build


def _templates(vocab: FigureVocab | None) -> tuple[tuple[str, _TemplateBuilder], ...]:
    templates: list[tuple[str, _TemplateBuilder]] = [(TEMPLATE, build_question)]
    if vocab is not None:
        templates.append((COUNT_TEMPLATE, _count_builder(vocab)))
        templates.append((PICK_TEMPLATE, _pick_builder(vocab)))
    return tuple(templates)


def question_quality_checks(
    draft: QuestionDraft,
    validation: SchemaValidation,
    *,
    skill_id: int | None,
    cp_id: int | None,
    confidence: float,
    duplicate_of: int | None,
    source_text: str,
) -> list[QaIssue]:
    """Automatic checks (§40). Blocking issues keep a candidate out of 'draft'."""
    issues: list[QaIssue] = []
    if skill_id is None:
        issues.append(QaIssue("missing_skill", "no curriculum skill mapped", True))
    if cp_id is None:
        issues.append(QaIssue("missing_cp_mapping", "no Capaian Pembelajaran mapped", True))
    if duplicate_of is not None:
        issues.append(QaIssue("duplicate", f"same question as candidate #{duplicate_of}", True))
    if not validation.ok:
        issues.append(QaIssue("schema_invalid", "; ".join(validation.errors), True))

    answer = draft.payload["answer"]
    content = draft.payload["content"]
    if draft.type in ("single_choice", "image_choice"):
        option_id = answer.get("option_id", "")
        options = content.get("options") or []
        ids = [option["id"] for option in options]
        if not option_id:
            issues.append(QaIssue("empty_answer", "no option selected as answer", True))
        if len(options) < 2 or len(set(ids)) != len(ids) or option_id not in ids:
            issues.append(QaIssue("invalid_options", "options are missing or inconsistent", True))
    elif draft.type == "short_answer":
        accepted = answer.get("accepted") or []
        if not accepted:
            issues.append(QaIssue("empty_answer", "no accepted answer", True))
    elif draft.type in ("tap_count", "number_input"):
        if answer.get("value") is None:
            issues.append(QaIssue("empty_answer", "no numeric answer", True))

    if draft.answer_text.lower() not in source_text.lower():
        issues.append(QaIssue("answer_mismatch", f"{draft.answer_text!r} not in the source", True))
    if len(draft.prompt.split()) > PROMPT_MAX_WORDS:
        issues.append(QaIssue("prompt_too_long", f"{len(draft.prompt.split())} words", False))
    if confidence < REVIEW_CONFIDENCE:
        issues.append(QaIssue("low_confidence", f"mapping confidence {confidence:.2f}", False))
    return issues


@dataclass(frozen=True)
class QuestionSummary:
    source_code: str
    generated: int
    drafts: int
    needs_review: int
    duplicates: int
    skipped: int
    invalid: int


@dataclass(frozen=True)
class _Pending:
    chunk_id: int
    document_id: int
    source_id: int
    source_code: str
    skill_id: int | None
    cp_id: int | None
    phase_code: str
    grade_code: str
    confidence: float
    heading: str
    text: str
    existing: frozenset[str]


class QuestionBuilder:
    def __init__(
        self,
        db: Database,
        schema: QuestionSchema,
        planner: VisualPlanner | None = None,
        vocab: FigureVocab | None = None,
    ) -> None:
        self._db = db
        self._schema = schema
        self._planner = planner
        self._templates = _templates(vocab)

    def run(
        self,
        *,
        codes: list[str] | None = None,
        limit: int | None = None,
        force: bool = False,
    ) -> list[QuestionSummary]:
        taxonomy = load_taxonomy(self._db)
        skills = {skill.id: skill for skill in taxonomy.skills}

        counters: dict[str, dict[str, int]] = {}
        for pending in self._pending(codes, limit):
            counts = counters.setdefault(
                pending.source_code,
                {"generated": 0, "drafts": 0, "needs_review": 0, "duplicates": 0,
                 "skipped": 0, "invalid": 0},
            )
            skill = skills.get(pending.skill_id) if pending.skill_id is not None else None
            made = 0
            for template, build in self._templates:
                if template in pending.existing and not force:
                    continue
                draft = build(
                    pending.text,
                    source_code=pending.source_code,
                    skill=skill,
                    phase_code=pending.phase_code,
                    grade_code=pending.grade_code,
                )
                if draft is None:
                    continue

                visual_issues: list[QaIssue] = []
                if self._planner is not None:
                    readability = analyse(
                        pending.text, phase_code=pending.phase_code, grade_code=pending.grade_code
                    )
                    need = visual_need(pending.phase_code, readability, draft.type)
                    visual = self._planner.propose(
                        draft.payload,
                        phase_code=pending.phase_code,
                        readability=readability,
                        source_text=pending.text,
                    )
                    draft = replace(draft, payload=apply_visual(draft.payload, visual))
                    visual_issues = visual_quality_checks(
                        draft.payload, need=need, vocab=self._planner.vocab, source_text=pending.text
                    )

                validation = self._schema.validate(draft.payload)
                if not validation.ok:
                    # A payload that fails the shared schema never enters staging.
                    log.warning("question for chunk %s failed the schema: %s",
                                pending.chunk_id, "; ".join(validation.errors))
                    counts["invalid"] += 1
                    continue

                duplicate_of = self._duplicate_of(draft.content_hash, pending.chunk_id)
                issues = question_quality_checks(
                    draft,
                    validation,
                    skill_id=pending.skill_id,
                    cp_id=pending.cp_id,
                    confidence=pending.confidence,
                    duplicate_of=duplicate_of,
                    source_text=pending.text,
                )
                issues += visual_issues
                status = "needs_review" if has_blocking(issues) else "draft"
                self._store(pending, taxonomy, draft, issues, status, duplicate_of)

                counts["generated"] += 1
                counts["needs_review" if status == "needs_review" else "drafts"] += 1
                if duplicate_of is not None:
                    counts["duplicates"] += 1
                made += 1

            if made == 0 and not pending.existing:
                counts["skipped"] += 1

        return [QuestionSummary(source_code=code, **counters[code]) for code in sorted(counters)]

    # ----------------------------------------------------------------- internal

    def _pending(self, codes: list[str] | None, limit: int | None) -> list[_Pending]:
        conditions = ["mc.status = 'candidate'"]
        params: list[object] = []
        if codes:
            conditions.append("s.code = ANY(%s)")
            params.append(codes)
        where = " AND ".join(conditions)
        limit_sql = ""
        if limit:
            limit_sql = " LIMIT %s"
            params.append(limit)

        sql = f"""
            SELECT mc.chunk_id, mc.document_id, mc.source_id, s.code,
                   mc.skill_id, mc.cp_id, mc.phase_code, mc.grade_code, mc.confidence,
                   c.heading, c.text,
                   COALESCE((SELECT array_agg(q.template) FROM content_staging.question_candidates q
                             WHERE q.chunk_id = mc.chunk_id), '{{}}') AS existing
            FROM content_staging.mapping_candidates mc
            JOIN content_staging.document_chunks c ON c.id = mc.chunk_id
            JOIN content_staging.source_documents d ON d.id = mc.document_id
            JOIN content_staging.source_registry s ON s.id = mc.source_id
            WHERE {where}
            ORDER BY mc.id
            {limit_sql}
        """
        with self._db.transaction() as cur:
            cur.execute(sql, params)
            rows = cur.fetchall()
        return [
            _Pending(
                chunk_id=int(row[0]),
                document_id=int(row[1]),
                source_id=int(row[2]),
                source_code=str(row[3]),
                skill_id=int(row[4]) if row[4] is not None else None,
                cp_id=int(row[5]) if row[5] is not None else None,
                phase_code=str(row[6]),
                grade_code=str(row[7]),
                confidence=float(row[8]),
                heading=str(row[9]),
                text=str(row[10]),
                existing=frozenset(str(template) for template in (row[11] or [])),
            )
            for row in rows
        ]

    def _duplicate_of(self, content_hash: str, chunk_id: int) -> int | None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                SELECT id FROM content_staging.question_candidates
                WHERE content_hash = %s AND chunk_id <> %s
                ORDER BY id LIMIT 1
                """,
                (content_hash, chunk_id),
            )
            row = cur.fetchone()
        return int(row[0]) if row is not None else None

    def _store(
        self,
        pending: _Pending,
        taxonomy: Taxonomy,
        draft: QuestionDraft,
        issues: list[QaIssue],
        status: str,
        duplicate_of: int | None,
    ) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                INSERT INTO content_staging.question_candidates
                    (document_id, chunk_id, source_id, curriculum_version_id, skill_id, cp_id,
                     phase_code, grade_code, template, type, difficulty, learning_objective,
                     prompt, payload, content_hash, duplicate_of, confidence, qa_issues, status)
                VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s)
                ON CONFLICT (chunk_id, template) DO UPDATE SET
                    curriculum_version_id = EXCLUDED.curriculum_version_id,
                    skill_id = EXCLUDED.skill_id,
                    cp_id = EXCLUDED.cp_id,
                    phase_code = EXCLUDED.phase_code,
                    grade_code = EXCLUDED.grade_code,
                    type = EXCLUDED.type,
                    difficulty = EXCLUDED.difficulty,
                    learning_objective = EXCLUDED.learning_objective,
                    prompt = EXCLUDED.prompt,
                    payload = EXCLUDED.payload,
                    content_hash = EXCLUDED.content_hash,
                    duplicate_of = EXCLUDED.duplicate_of,
                    confidence = EXCLUDED.confidence,
                    qa_issues = EXCLUDED.qa_issues,
                    status = EXCLUDED.status,
                    updated_at = now()
                """,
                (
                    pending.document_id, pending.chunk_id, pending.source_id,
                    taxonomy.curriculum_version_id or None, pending.skill_id, pending.cp_id,
                    pending.phase_code, pending.grade_code, draft.template, draft.type,
                    draft.difficulty, draft.learning_objective, draft.prompt,
                    json.dumps(draft.payload), draft.content_hash, duplicate_of,
                    pending.confidence, json.dumps([issue.as_dict() for issue in issues]),
                    status,
                ),
            )
