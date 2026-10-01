from __future__ import annotations

import pytest

from content_pipeline.similarity import (
    best_match,
    classify_similarity,
    cosine,
    semantic_confidence,
)


def test_cosine_identities() -> None:
    assert cosine([1.0, 0.0], [1.0, 0.0]) == pytest.approx(1.0)
    assert cosine([1.0, 0.0], [0.0, 1.0]) == pytest.approx(0.0)


def test_cosine_clamps_negatives_and_handles_bad_input() -> None:
    assert cosine([1.0, 0.0], [-1.0, 0.0]) == 0.0
    assert cosine([], [1.0]) == 0.0
    assert cosine([1.0, 2.0], [1.0, 2.0, 3.0]) == 0.0
    assert cosine([0.0, 0.0], [1.0, 0.0]) == 0.0


def test_classify_similarity_boundaries() -> None:
    assert classify_similarity(0.799) == "unique"
    assert classify_similarity(0.80) == "related"
    assert classify_similarity(0.919) == "related"
    assert classify_similarity(0.92) == "near_duplicate"


def test_semantic_confidence_clamps_to_unit_interval() -> None:
    assert semantic_confidence(0.10) == 0.0
    assert semantic_confidence(0.55) == 0.0
    assert semantic_confidence(1.0) == pytest.approx(1.0)
    assert semantic_confidence(0.775) == pytest.approx(0.5)


def test_best_match_picks_the_highest_cosine() -> None:
    candidates = {1: [0.0, 1.0], 2: [1.0, 0.0]}
    match = best_match([1.0, 0.0], candidates)
    assert match is not None
    assert match[0] == 2
    assert match[1] == pytest.approx(1.0)
    assert best_match([1.0, 0.0], {}) is None
