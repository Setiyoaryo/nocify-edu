from collections.abc import Sequence
from typing import cast

import pytest

from content_pipeline.db import Database
from content_pipeline.figures import load_vocab
from content_pipeline.readability import analyse
from content_pipeline.validation import QuestionSchema
from content_pipeline.visuals import (
    MAX_FIGURE_COUNT,
    AssetMatch,
    DbAssetCatalog,
    Visual,
    VisualPlanner,
    answer_text,
    apply_visual,
    grounded_icons,
    propose_visual,
    visual_need,
    visual_quality_checks,
)


class FakeCatalog:
    """In-memory AssetCatalog for DB-free tests."""

    def __init__(self, match: AssetMatch | None = None, published: bool | None = True) -> None:
        self._match = match
        self._published = published
        self.seen_tags: list[str] | None = None

    def match(self, tags: Sequence[str]) -> AssetMatch | None:
        self.seen_tags = list(tags)
        return self._match

    def is_published(self, asset_id: str) -> bool | None:
        return self._published


def _payload(
    *,
    question_type: str = "single_choice",
    answer: dict[str, object] | None = None,
    content: dict[str, object] | None = None,
) -> dict[str, object]:
    return {
        "type": question_type,
        "content": content if content is not None else {"prompt": "Lengkapi kalimat."},
        "answer": answer if answer is not None else {"accepted": ["pasar"]},
    }


# --------------------------------------------------------------------- visual_need


def test_visual_need_early_bands_and_picture_types() -> None:
    fondasi = analyse("Ibu membeli jeruk.", phase_code="fondasi")
    fase_c = analyse("Ibu membeli jeruk di pasar tradisional setiap pagi.", phase_code="c")
    assert visual_need("fondasi", fondasi, "single_choice") is True
    assert visual_need("a", fondasi, "single_choice") is True
    assert visual_need("c", fase_c, "single_choice") is False
    # Picture-first types always need a stimulus, whatever the band.
    assert visual_need("c", fase_c, "image_choice") is True
    assert visual_need("c", fase_c, "tap_count") is True


# ---------------------------------------------------------------- grounded_icons


def test_grounded_icons_finds_labels_in_order() -> None:
    vocab = load_vocab()
    assert grounded_icons("Ibu membeli jeruk dan wortel.", vocab) == ["orange", "carrot"]


def test_grounded_icons_prefers_the_longer_label() -> None:
    vocab = load_vocab()
    assert grounded_icons("Baca buku terbuka itu.", vocab) == ["book-open"]
    assert grounded_icons("Baca buku itu.", vocab) == ["book"]


def test_grounded_icons_dedupes() -> None:
    vocab = load_vocab()
    assert grounded_icons("Jeruk, jeruk, dan jeruk lagi.", vocab) == ["orange"]


def test_grounded_icons_ignores_unknown_words() -> None:
    vocab = load_vocab()
    assert grounded_icons("Keluarga pergi ke taman.", vocab) == []


# ------------------------------------------------------------------- answer_text


def test_answer_text_reads_each_shape() -> None:
    assert answer_text({"answer": {"accepted": ["jeruk"]}}) == "jeruk"
    assert answer_text({"answer": {"value": 3}}) == "3"
    option = {
        "answer": {"option_id": "b"},
        "content": {"options": [{"id": "a", "label": "apel"}, {"id": "b", "label": "jeruk"}]},
    }
    assert answer_text(option) == "jeruk"
    assert answer_text({"answer": {}}) == ""


# ------------------------------------------------------------------ icon path


def test_propose_visual_icon_path() -> None:
    vocab = load_vocab()
    source = "Ibu membeli jeruk dan wortel di pasar."
    payload = _payload(answer={"accepted": ["pasar"]})
    visual = propose_visual(
        payload,
        phase_code="a",
        readability=analyse(source, phase_code="a"),
        source_text=source,
        vocab=vocab,
    )
    assert visual is not None
    assert [figure["icon"] for figure in visual.figures] == ["orange", "carrot"]
    assert all(figure["count"] == 1 for figure in visual.figures)
    assert visual.figures[0]["color"] == vocab.default_color("orange")
    assert visual.image is None


def test_propose_visual_drops_the_answer_concept() -> None:
    vocab = load_vocab()
    source = "Ani makan jeruk."
    visual = propose_visual(
        _payload(answer={"accepted": ["jeruk"]}),
        phase_code="a",
        readability=analyse(source, phase_code="a"),
        source_text=source,
        vocab=vocab,
    )
    # The only grounded concept is the answer itself, so nothing is honest to show.
    assert visual is None


def test_propose_visual_keeps_non_answer_concepts() -> None:
    vocab = load_vocab()
    source = "Ani makan jeruk dan wortel."
    visual = propose_visual(
        _payload(answer={"accepted": ["jeruk"]}),
        phase_code="a",
        readability=analyse(source, phase_code="a"),
        source_text=source,
        vocab=vocab,
    )
    assert visual is not None
    assert [figure["icon"] for figure in visual.figures] == ["carrot"]


def test_propose_visual_icon_path_skips_the_catalogue() -> None:
    vocab = load_vocab()
    catalog = FakeCatalog(AssetMatch("abc", "scene.svg", ("jeruk",)))
    source = "Ibu membeli jeruk."
    visual = propose_visual(
        _payload(answer={"accepted": ["pasar"]}),
        phase_code="a",
        readability=analyse(source, phase_code="a"),
        source_text=source,
        vocab=vocab,
        catalog=catalog,
    )
    assert visual is not None and visual.figures
    assert catalog.seen_tags is None  # illustrations are the fallback


def test_propose_visual_none_when_not_needed() -> None:
    vocab = load_vocab()
    source = "Ibu membeli jeruk di pasar tradisional setiap pagi."
    visual = propose_visual(
        _payload(),
        phase_code="c",
        readability=analyse(source, phase_code="c"),
        source_text=source,
        vocab=vocab,
    )
    assert visual is None


# ----------------------------------------------------------- illustration path


def test_propose_visual_illustration_path() -> None:
    vocab = load_vocab()
    asset = AssetMatch("11111111-1111-1111-1111-111111111111", "family-park.svg", ("keluarga", "taman"))
    catalog = FakeCatalog(asset)
    source = "Keluarga pergi ke taman."
    visual = propose_visual(
        _payload(answer={"accepted": ["jalan"]}),
        phase_code="a",
        readability=analyse(source, phase_code="a"),
        source_text=source,
        vocab=vocab,
        catalog=catalog,
    )
    assert visual is not None
    assert visual.image == "/api/v1/media/11111111-1111-1111-1111-111111111111"
    assert visual.image_alt is not None and "keluarga" in visual.image_alt
    assert catalog.seen_tags is not None and "keluarga" in catalog.seen_tags


def test_propose_visual_illustration_refuses_to_leak() -> None:
    vocab = load_vocab()
    asset = AssetMatch("abc", "family.svg", ("keluarga",))
    catalog = FakeCatalog(asset)
    source = "Keluarga pergi ke taman."
    visual = propose_visual(
        _payload(answer={"accepted": ["keluarga"]}),
        phase_code="a",
        readability=analyse(source, phase_code="a"),
        source_text=source,
        vocab=vocab,
        catalog=catalog,
    )
    assert visual is None


def test_propose_visual_without_catalogue_is_text_only() -> None:
    vocab = load_vocab()
    source = "Keluarga pergi ke taman."
    visual = propose_visual(
        _payload(),
        phase_code="a",
        readability=analyse(source, phase_code="a"),
        source_text=source,
        vocab=vocab,
        catalog=None,
    )
    assert visual is None


# ------------------------------------------------------------------ QA checks


def test_visual_quality_checks_flags_unknown_icon() -> None:
    vocab = load_vocab()
    payload = _payload(content={"prompt": "p", "figures": [{"icon": "not-real", "count": 1}]})
    issues = visual_quality_checks(payload, need=True, vocab=vocab)
    assert any(issue.code == "unknown_icon" and issue.blocking for issue in issues)


def test_visual_quality_checks_flags_answer_leak() -> None:
    vocab = load_vocab()
    payload = _payload(
        answer={"accepted": ["jeruk"]},
        content={"prompt": "p", "figures": [{"icon": "orange", "count": 1}]},
    )
    issues = visual_quality_checks(payload, need=True, vocab=vocab, source_text="Ada jeruk.")
    assert any(issue.code == "visual_answer_leak" and issue.blocking for issue in issues)


def test_visual_quality_checks_flags_missing_alt() -> None:
    vocab = load_vocab()
    payload = _payload(content={"prompt": "p", "image": "/api/v1/media/abc"})
    issues = visual_quality_checks(payload, need=True, vocab=vocab)
    assert any(issue.code == "alt_missing" and issue.blocking for issue in issues)


def test_visual_quality_checks_grounding_is_advisory() -> None:
    vocab = load_vocab()
    payload = _payload(content={"prompt": "p", "figures": [{"icon": "orange", "count": 1}]})
    issues = visual_quality_checks(payload, need=True, vocab=vocab, source_text="Tidak ada buah.")
    ungrounded = [issue for issue in issues if issue.code == "visual_ungrounded"]
    assert ungrounded and not ungrounded[0].blocking


def test_visual_quality_checks_required_missing_is_advisory() -> None:
    vocab = load_vocab()
    issues = visual_quality_checks(_payload(), need=True, vocab=vocab)
    missing = [issue for issue in issues if issue.code == "visual_required_missing"]
    assert missing and not missing[0].blocking


def test_visual_quality_checks_count_range() -> None:
    vocab = load_vocab()
    payload = _payload(content={"prompt": "p", "figures": [{"icon": "orange", "count": MAX_FIGURE_COUNT + 1}]})
    issues = visual_quality_checks(payload, need=True, vocab=vocab)
    assert any(issue.code == "unknown_icon" and issue.blocking for issue in issues)


# ------------------------------------------------------------------ apply_visual


def test_apply_visual_merges_figures_and_image() -> None:
    payload = _payload()
    figures = Visual(figures=({"icon": "orange", "count": 1, "color": "orange"},))
    merged = apply_visual(payload, figures)
    assert merged["content"]["figures"] == [{"icon": "orange", "count": 1, "color": "orange"}]
    image = Visual(image="/api/v1/media/abc", image_alt="Ilustrasi: jeruk.")
    merged = apply_visual(payload, image)
    assert merged["content"]["image"] == "/api/v1/media/abc"
    assert merged["content"]["image_alt"] == "Ilustrasi: jeruk."


def test_apply_visual_none_is_a_no_op() -> None:
    payload = _payload()
    assert apply_visual(payload, None) is payload
    assert apply_visual(payload, Visual()) is payload


def test_proposed_visual_keeps_the_payload_schema_valid() -> None:
    vocab = load_vocab()
    source = "Ibu membeli jeruk dan wortel di pasar."
    payload: dict[str, object] = {
        "code": "Q-TEST-12345678",
        "skill": "SK-1",
        "type": "single_choice",
        "difficulty": 2,
        "learning_objective": "Peserta didik dapat mengenal buah.",
        "content": {
            "prompt": "Lengkapi: Ibu membeli _____ di pasar.",
            "options": [
                {"id": "a", "label": "apel"},
                {"id": "b", "label": "pasar"},
                {"id": "c", "label": "meja"},
                {"id": "d", "label": "buku"},
            ],
        },
        "answer": {"option_id": "b"},
        "explanation": "Jawaban: pasar.",
    }
    visual = propose_visual(
        payload,
        phase_code="a",
        readability=analyse(source, phase_code="a"),
        source_text=source,
        vocab=vocab,
    )
    merged = apply_visual(payload, visual)
    assert QuestionSchema.load().validate(merged).ok


# ------------------------------------------------------------------- planner


def test_planner_verify_reports_unpublished() -> None:
    vocab = load_vocab()
    payload = _payload(content={"prompt": "p", "image": "/api/v1/media/abc", "image_alt": "x"})
    assert VisualPlanner(vocab, FakeCatalog(published=False)).verify(payload)
    assert VisualPlanner(vocab, FakeCatalog(published=True)).verify(payload) == []
    assert VisualPlanner(vocab, FakeCatalog(published=None)).verify(payload) == []


def test_planner_verify_without_catalogue() -> None:
    vocab = load_vocab()
    payload = _payload(content={"prompt": "p", "image": "/api/v1/media/abc", "image_alt": "x"})
    assert VisualPlanner(vocab, None).verify(payload) == []


# ------------------------------------------------------------------- catalogue


class _FailingDb:
    """Stands in for an unreachable public.assets (missing grant/table)."""

    def transaction(self) -> object:
        raise RuntimeError("permission denied for table assets")


def test_db_catalogue_fails_closed() -> None:
    catalog = DbAssetCatalog(cast(Database, _FailingDb()))
    assert catalog.match(["jeruk"]) is None
    assert catalog.is_published("abc") is None


def test_db_catalogue_ignores_empty_tags() -> None:
    catalog = DbAssetCatalog(cast(Database, _FailingDb()))
    assert catalog.match([]) is None


if __name__ == "__main__":  # pragma: no cover
    raise SystemExit(pytest.main([__file__]))
