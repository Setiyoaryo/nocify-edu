"""Visual proposals for early-band questions (§22, §30, M10).

TK (Fase Fondasi) and Fase A questions are visual-first: a bare sentence is not a
usable stimulus for a 4–8 year old. This module decides *whether* a question needs
a pictured stimulus and proposes one, without ever inventing an image:

* **Icon path** — a small, grounded match: an Indonesian noun that appears in the
  source text and is also an approved figure (``packages/figures/figures.json``)
  becomes a ``content.figures`` entry. The vocabulary is shared with the Go API
  and the web client, so nothing here can render as a missing glyph.
* **Illustration path** — only when no icon matches *and* a licensed, published,
  non-AI asset carries a matching tag. The pipeline references the asset by its
  ``/api/v1/media/<uuid>`` URL; it never fabricates an id.

Two invariants protect the learner:

* **No answer leak** — a visual must not depict the answer. For a cloze this is the
  whole point of the sentence, so a leaking concept is dropped (and flagged by QA
  if one ever slips through).
* **Grounding** — a visual is only proposed for a concept actually present in the
  source. Ungrounded stimuli are advisory QA, not silent invention.

The proposal is a *suggestion*: it lands in ``question_candidates.payload`` and a
reviewer decides (§24). Publish stays blocked while a referenced asset is not
``published`` (the Go ``linkMedia`` gate).
"""

from __future__ import annotations

import json
import logging
import re
from collections.abc import Sequence
from dataclasses import dataclass
from typing import Any, Protocol

from content_pipeline.db import Database
from content_pipeline.figures import FigureVocab
from content_pipeline.mapping import STOPWORDS
from content_pipeline.qa import QaIssue
from content_pipeline.readability import Readability, analyse
from content_pipeline.validation import QuestionSchema

log = logging.getLogger("content_pipeline.visuals")

# Schema limits (packages/question-schema): a figure's count is 1–12. The pipeline
# keeps the *total* pictured objects modest so a stimulus stays legible.
MAX_FIGURE_COUNT = 12
MAX_TOTAL_COUNT = 24
MAX_FIGURES = 8
MIN_TAG_LEN = 4

# Question types that are meaningless without a picture.
IMAGE_TYPES = frozenset({"image_choice", "tap_count"})

VISUAL_QA_CODES = frozenset(
    {
        "unknown_icon",
        "alt_missing",
        "asset_unpublished",
        "visual_answer_leak",
        "visual_ungrounded",
        "visual_required_missing",
    }
)

_NON_WORD = re.compile(r"[^\w]+", re.UNICODE)


@dataclass(frozen=True)
class Visual:
    """A proposed stimulus: either icons, an illustration, or nothing."""

    figures: tuple[dict[str, Any], ...] = ()
    image: str | None = None
    image_alt: str | None = None

    @property
    def is_empty(self) -> bool:
        return not self.figures and not self.image


@dataclass(frozen=True)
class AssetMatch:
    id: str
    name: str
    tags: tuple[str, ...] = ()


class AssetCatalog(Protocol):
    """Read-only view of the published, non-AI asset library."""

    def match(self, tags: Sequence[str]) -> AssetMatch | None: ...

    def is_published(self, asset_id: str) -> bool | None:
        """``True``/``False`` when known; ``None`` when the catalogue is unreachable."""
        ...


def _normalise(text: str) -> str:
    return _NON_WORD.sub(" ", text.lower()).strip()


def visual_need(phase_code: str, readability: Readability, question_type: str) -> bool:
    """True when this question should carry a pictured stimulus. Pure."""
    if question_type in IMAGE_TYPES:
        return True
    if readability.visual_required:
        return True
    return phase_code in ("fondasi", "a")


def _label_index(vocab: FigureVocab) -> dict[str, str]:
    """Indonesian label → icon name, from the shared vocabulary (no drift)."""
    index: dict[str, str] = {}
    for icon in vocab.icons:
        label = _normalise(vocab.label(icon))
        if label:
            index.setdefault(label, icon)
    return index


def grounded_icons(text: str, vocab: FigureVocab) -> list[str]:
    """Approved icons whose Indonesian label appears in the text, in order, deduped.

    Prefers a two-word label (``buku terbuka``) over its single-word prefix
    (``buku``) so the more specific figure wins.
    """
    tokens = _normalise(text).split()
    index = _label_index(vocab)
    out: list[str] = []
    seen: set[str] = set()
    i = 0
    while i < len(tokens):
        pair = f"{tokens[i]} {tokens[i + 1]}" if i + 1 < len(tokens) else ""
        if pair and pair in index:
            icon: str | None = index[pair]
            i += 2
        else:
            icon = index.get(tokens[i])
            i += 1
        if icon is not None and icon not in seen:
            out.append(icon)
            seen.add(icon)
    return out


def answer_text(payload: dict[str, Any]) -> str:
    """The human-readable answer of a payload, for leak detection. Pure."""
    answer = payload.get("answer") or {}
    accepted = answer.get("accepted") or []
    if accepted:
        return str(accepted[0])
    option_id = answer.get("option_id")
    if option_id:
        for option in (payload.get("content") or {}).get("options") or []:
            if option.get("id") == option_id:
                return str(option.get("label", ""))
    if answer.get("value") is not None:
        return str(answer["value"])
    return ""


def _leaks(label: str, answer: str) -> bool:
    """True when a visual's label would give the answer away."""
    given = _normalise(answer)
    shown = _normalise(label)
    if not given or not shown:
        return False
    return given == shown or given in shown.split() or shown in given.split()


def _tag_concepts(text: str, answer: str) -> list[str]:
    """Content words usable as asset tags, in order, excluding the answer."""
    given = _normalise(answer)
    seen: set[str] = set()
    out: list[str] = []
    for word in _normalise(text).split():
        if len(word) < MIN_TAG_LEN or word == given or word in STOPWORDS or word in seen:
            continue
        seen.add(word)
        out.append(word)
    return out


def _illustration_alt(tags: Sequence[str], match: AssetMatch) -> str:
    matched = [tag for tag in tags if tag in match.tags] or list(tags[:3])
    subject = ", ".join(matched) if matched else match.name
    return f"Ilustrasi: {subject}."


def _match_leaks(match: AssetMatch, answer: str) -> bool:
    if _leaks(match.name, answer):
        return True
    return any(_leaks(tag, answer) for tag in match.tags)


def propose_visual(
    payload: dict[str, Any],
    *,
    phase_code: str,
    readability: Readability,
    source_text: str,
    vocab: FigureVocab,
    catalog: AssetCatalog | None = None,
) -> Visual | None:
    """Propose a stimulus for a question, or ``None`` when nothing is honest. Pure."""
    question_type = str(payload.get("type", ""))
    if not visual_need(phase_code, readability, question_type):
        return None
    answer = answer_text(payload)

    figures: list[dict[str, Any]] = []
    total = 0
    for icon in grounded_icons(source_text, vocab):
        if _leaks(vocab.label(icon), answer) or _leaks(icon, answer):
            continue
        figures.append({"icon": icon, "count": 1, "color": vocab.default_color(icon)})
        total += 1
        if len(figures) >= MAX_FIGURES or total >= MAX_TOTAL_COUNT:
            break
    if figures:
        return Visual(figures=tuple(figures))

    if catalog is not None:
        tags = _tag_concepts(source_text, answer)
        match = catalog.match(tags) if tags else None
        if match is not None and not _match_leaks(match, answer):
            return Visual(image=f"/api/v1/media/{match.id}", image_alt=_illustration_alt(tags, match))
    return None


def visual_quality_checks(
    payload: dict[str, Any],
    *,
    need: bool,
    vocab: FigureVocab,
    source_text: str = "",
) -> list[QaIssue]:
    """Automatic checks for a payload's stimulus (§40). Pure."""
    issues: list[QaIssue] = []
    content = payload.get("content") or {}
    figures = content.get("figures") or []
    image = content.get("image")
    alt = content.get("image_alt")
    # tap_count carries its stimulus as symbol+count rather than a figure list.
    has_stimulus = bool(figures or image or content.get("symbol"))
    answer = answer_text(payload)
    normalised_source = _normalise(source_text)

    for figure in figures:
        icon = str(figure.get("icon", ""))
        if not vocab.has(icon):
            issues.append(QaIssue("unknown_icon", f"{icon!r} is not an approved figure", True))
            continue
        count = int(figure.get("count", 1))
        if not 1 <= count <= MAX_FIGURE_COUNT:
            issues.append(QaIssue("unknown_icon", f"{icon!r} count {count} is out of range", True))
        if _leaks(vocab.label(icon), answer):
            issues.append(QaIssue("visual_answer_leak", f"figure {icon!r} gives away the answer", True))
        if normalised_source and _normalise(vocab.label(icon)) not in normalised_source:
            issues.append(QaIssue("visual_ungrounded", f"figure {icon!r} is not in the source", False))

    if image and not alt:
        issues.append(QaIssue("alt_missing", "image has no image_alt", True))
    if need and not has_stimulus:
        issues.append(QaIssue("visual_required_missing", "this band expects a pictured stimulus", False))
    return issues


def apply_visual(payload: dict[str, Any], visual: Visual | None) -> dict[str, Any]:
    """Return a copy of a payload with the visual merged into ``content``. Pure.

    A payload that already carries a stimulus (a figure list, an illustration, or a
    ``tap_count`` symbol) is left untouched so a template's own picture wins.
    """
    if visual is None or visual.is_empty:
        return payload
    content = payload.get("content") or {}
    if content.get("figures") or content.get("image") or content.get("symbol"):
        return payload
    merged = dict(content)
    if visual.figures:
        merged["figures"] = [dict(figure) for figure in visual.figures]
    if visual.image:
        merged["image"] = visual.image
        merged["image_alt"] = visual.image_alt
    return {**payload, "content": merged}


class VisualPlanner:
    """Binds the shared vocabulary and the asset catalogue to ``propose_visual``."""

    def __init__(self, vocab: FigureVocab, catalog: AssetCatalog | None = None) -> None:
        self._vocab = vocab
        self._catalog = catalog

    @property
    def vocab(self) -> FigureVocab:
        return self._vocab

    def propose(
        self,
        payload: dict[str, Any],
        *,
        phase_code: str,
        readability: Readability,
        source_text: str,
    ) -> Visual | None:
        return propose_visual(
            payload,
            phase_code=phase_code,
            readability=readability,
            source_text=source_text,
            vocab=self._vocab,
            catalog=self._catalog,
        )

    def verify(self, payload: dict[str, Any]) -> list[QaIssue]:
        """Flag a referenced illustration the catalogue positively knows is not published."""
        if self._catalog is None:
            return []
        image = (payload.get("content") or {}).get("image")
        if not image:
            return []
        asset_id = str(image).rsplit("/", 1)[-1]
        if self._catalog.is_published(asset_id) is False:
            return [QaIssue("asset_unpublished", f"asset {asset_id} is not published", True)]
        return []


class DbAssetCatalog:
    """Reads the canonical asset library; fails closed when the grant is missing."""

    def __init__(self, db: Database) -> None:
        self._db = db

    def match(self, tags: Sequence[str]) -> AssetMatch | None:
        if not tags:
            return None
        try:
            with self._db.transaction() as cur:
                cur.execute(
                    """
                    SELECT a.id::text, a.name, a.tags
                    FROM public.assets a
                    WHERE a.status = 'published' AND NOT a.ai_generated
                      AND a.tags && %s::text[]
                    ORDER BY a.updated_at DESC
                    LIMIT 1
                    """,
                    (list(tags),),
                )
                row = cur.fetchone()
        except Exception as exc:  # noqa: BLE001 - fail closed when public.assets is unavailable
            log.warning("asset catalogue unavailable: %s", exc)
            return None
        if row is None:
            return None
        return AssetMatch(id=str(row[0]), name=str(row[1]), tags=tuple(str(tag) for tag in (row[2] or [])))

    def is_published(self, asset_id: str) -> bool | None:
        try:
            with self._db.transaction() as cur:
                cur.execute(
                    "SELECT 1 FROM public.assets WHERE id = %s AND status = 'published' AND NOT ai_generated",
                    (asset_id,),
                )
                return cur.fetchone() is not None
        except Exception as exc:  # noqa: BLE001 - unknown, not "unpublished"
            log.warning("asset catalogue unavailable: %s", exc)
            return None


@dataclass(frozen=True)
class VisualSummary:
    source_code: str
    considered: int
    proposed: int
    figures: int
    images: int
    already: int
    skipped: int
    needs_review: int


@dataclass(frozen=True)
class _Candidate:
    id: int
    source_code: str
    payload: dict[str, Any]
    phase_code: str
    grade_code: str
    text: str
    status: str
    qa_issues: tuple[dict[str, Any], ...]


class VisualBuilder:
    """Proposes visuals for existing question candidates (the ``visuals`` command).

    Idempotent: a candidate that already carries a stimulus is left alone unless
    ``force`` is set, so re-runs are cheap and never clobber a reviewer's edits.
    """

    def __init__(self, db: Database, schema: QuestionSchema, planner: VisualPlanner) -> None:
        self._db = db
        self._schema = schema
        self._planner = planner

    def run(
        self,
        *,
        codes: list[str] | None = None,
        limit: int | None = None,
        force: bool = False,
        dry_run: bool = False,
    ) -> list[VisualSummary]:
        counters: dict[str, dict[str, int]] = {}
        for candidate in self._pending(codes, limit):
            counts = counters.setdefault(
                candidate.source_code,
                {"considered": 0, "proposed": 0, "figures": 0, "images": 0,
                 "already": 0, "skipped": 0, "needs_review": 0},
            )
            counts["considered"] += 1
            readability = analyse(candidate.text, phase_code=candidate.phase_code, grade_code=candidate.grade_code)
            if not visual_need(candidate.phase_code, readability, str(candidate.payload.get("type", ""))):
                counts["skipped"] += 1
                continue

            content = candidate.payload.get("content") or {}
            if (content.get("figures") or content.get("image") or content.get("symbol")) and not force:
                counts["already"] += 1
                continue

            visual = self._planner.propose(
                candidate.payload,
                phase_code=candidate.phase_code,
                readability=readability,
                source_text=candidate.text,
            )
            if visual is None or visual.is_empty:
                counts["skipped"] += 1
                continue

            payload = apply_visual(candidate.payload, visual)
            issues = self._merge_issues(candidate, payload)
            status = "needs_review" if any(issue["blocking"] for issue in issues) else candidate.status
            if not dry_run:
                self._store(candidate.id, payload, issues, status)

            counts["proposed"] += 1
            counts["figures" if visual.figures else "images"] += 1
            if status == "needs_review":
                counts["needs_review"] += 1

        return [VisualSummary(source_code=code, **counters[code]) for code in sorted(counters)]

    # ----------------------------------------------------------------- internal

    def _pending(self, codes: list[str] | None, limit: int | None) -> list[_Candidate]:
        conditions = ["q.status <> 'rejected'"]
        params: list[object] = []
        if codes:
            conditions.append("s.code = ANY(%s)")
            params.append(codes)
        limit_sql = ""
        if limit:
            limit_sql = " LIMIT %s"
            params.append(limit)
        sql = f"""
            SELECT q.id, COALESCE(s.code, ''), q.payload, q.phase_code, q.grade_code,
                   c.text, q.status, q.qa_issues
            FROM content_staging.question_candidates q
            JOIN content_staging.document_chunks c ON c.id = q.chunk_id
            LEFT JOIN content_staging.source_registry s ON s.id = q.source_id
            WHERE {' AND '.join(conditions)}
            ORDER BY q.id
            {limit_sql}
        """
        with self._db.transaction() as cur:
            cur.execute(sql, params)
            rows = cur.fetchall()
        return [
            _Candidate(
                id=int(row[0]),
                source_code=str(row[1]),
                payload=dict(row[2]),
                phase_code=str(row[3]),
                grade_code=str(row[4]),
                text=str(row[5]),
                status=str(row[6]),
                qa_issues=tuple(dict(issue) for issue in (row[7] or [])),
            )
            for row in rows
        ]

    def _merge_issues(self, candidate: _Candidate, payload: dict[str, Any]) -> list[dict[str, Any]]:
        kept = [issue for issue in candidate.qa_issues if issue.get("code") not in VISUAL_QA_CODES]
        visual = visual_quality_checks(
            payload, need=True, vocab=self._planner.vocab, source_text=candidate.text
        )
        visual += self._planner.verify(payload)
        return kept + [issue.as_dict() for issue in visual]

    def _store(self, candidate_id: int, payload: dict[str, Any], issues: list[dict[str, Any]], status: str) -> None:
        with self._db.transaction() as cur:
            cur.execute(
                """
                UPDATE content_staging.question_candidates
                SET payload = %s, qa_issues = %s, status = %s, updated_at = now()
                WHERE id = %s
                """,
                (json.dumps(payload), json.dumps(issues), status, candidate_id),
            )

