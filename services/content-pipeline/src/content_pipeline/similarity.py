"""Pure similarity math for the semantic layer (M6, §18/§29).

Cosine similarity is computed in Python because the dev Postgres has no pgvector;
vectors are stored as ``double precision[]``. Thresholds are module constants so
they can be re-tuned per embedding model without a code change, and are only ever
applied to vectors produced by the same model.
"""

from __future__ import annotations

import math
from collections.abc import Mapping, Sequence

# §29 has no numeric thresholds; these are our tuned defaults for sentence/
# paragraph embeddings (see docs/content-pipeline-roadmap.md, M6).
NEAR_DUPLICATE_THRESHOLD = 0.92
RELATED_THRESHOLD = 0.80

# §18 confidence mapping: an embedding cosine below the floor carries no signal,
# and the span normalises the rest to 0..1.
SEMANTIC_FLOOR = 0.55
SEMANTIC_SPAN = 0.45


def cosine(left: Sequence[float], right: Sequence[float]) -> float:
    """Cosine similarity clamped to 0..1 (length mismatch or a zero vector -> 0)."""
    if not left or not right or len(left) != len(right):
        return 0.0
    dot = sum(a * b for a, b in zip(left, right, strict=True))
    left_norm = math.sqrt(sum(a * a for a in left))
    right_norm = math.sqrt(sum(b * b for b in right))
    if left_norm == 0 or right_norm == 0:
        return 0.0
    return max(0.0, min(1.0, dot / (left_norm * right_norm)))


def semantic_confidence(score: float) -> float:
    """Normalise a raw cosine into the 0..1 confidence the mapping blend expects."""
    return max(0.0, min(1.0, (score - SEMANTIC_FLOOR) / SEMANTIC_SPAN))


def classify_similarity(score: float) -> str:
    """The §29 status for a non-identical pair: near_duplicate | related | unique."""
    if score >= NEAR_DUPLICATE_THRESHOLD:
        return "near_duplicate"
    if score >= RELATED_THRESHOLD:
        return "related"
    return "unique"


def best_match(
    query: Sequence[float],
    candidates: Mapping[int, Sequence[float]],
) -> tuple[int, float] | None:
    """Return (key, cosine) for the closest candidate, or None when empty."""
    best: tuple[int, float] | None = None
    for key, vector in candidates.items():
        score = cosine(query, vector)
        if best is None or score > best[1]:
            best = (key, score)
    return best
