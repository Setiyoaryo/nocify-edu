from __future__ import annotations

from content_pipeline.dedup import dedup_issues, relation_status


def test_exact_hash_beats_similarity() -> None:
    assert relation_status("abc", "abc", 0.10) == "exact"


def test_relation_status_uses_similarity_otherwise() -> None:
    assert relation_status("abc", "def", 0.95) == "near_duplicate"
    assert relation_status("abc", "def", 0.85) == "related"
    assert relation_status("abc", "def", 0.50) == "unique"


def test_near_duplicate_blocks_and_related_does_not() -> None:
    near = dedup_issues("near_duplicate", 7)
    assert near[0].code == "near_duplicate"
    assert near[0].blocking is True
    assert "7" in near[0].message

    related = dedup_issues("related", None)
    assert related[0].code == "related"
    assert related[0].blocking is False


def test_unique_and_exact_produce_no_issues() -> None:
    assert dedup_issues("unique", None) == []
    assert dedup_issues("exact", 3) == []
