"""Approved question-figure vocabulary (§22, §30).

``packages/figures/figures.json`` is the single source of truth for the pictured
objects a question may use. The Go API generates its ``Icons`` allowlist from it
(``apps/api/internal/questions``), the web client renders each name from its
licensed Phosphor set (``apps/web/src/lib/figures.ts``), and this module loads it
so the pipeline proposes only names the other two sides accept.

The pipeline never invents an icon: a proposal that is not in this vocabulary is
a QA issue, not a silently-accepted figure.
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass
from pathlib import Path

VOCAB_RELATIVE = Path("packages/figures/figures.json")

# Mirrors questions.Colors in the Go API (the shared palette).
PALETTE: tuple[str, ...] = (
    "red", "orange", "yellow", "green", "blue", "purple", "pink", "brown", "gray",
)


@dataclass(frozen=True)
class FigureVocab:
    icons: tuple[str, ...]
    labels: dict[str, str]
    default_colors: dict[str, str]

    def has(self, name: str) -> bool:
        return name in self.labels

    def label(self, name: str) -> str:
        """Indonesian noun for alt text; falls back to the raw name."""
        return self.labels.get(name, name)

    def default_color(self, name: str) -> str:
        return self.default_colors.get(name, "gray")


def find_figures_json() -> Path | None:
    """Locate the shared vocabulary from an env override, the package, or the repo root."""
    override = os.environ.get("FIGURES_PATH", "").strip()
    if override:
        candidate = Path(override)
        return candidate if candidate.is_file() else None
    for base in (Path(__file__).resolve(), Path.cwd().resolve()):
        for parent in (base, *base.parents):
            candidate = parent / VOCAB_RELATIVE
            if candidate.is_file():
                return candidate
    return None


def load_vocab(path: Path | None = None) -> FigureVocab:
    resolved = path or find_figures_json()
    if resolved is None:
        raise FileNotFoundError(
            "figures vocabulary not found; set FIGURES_PATH to packages/figures/figures.json"
        )
    raw = json.loads(resolved.read_text(encoding="utf-8"))
    if not isinstance(raw, list) or not raw:
        raise ValueError(f"{resolved} must be a non-empty JSON array")

    icons: list[str] = []
    labels: dict[str, str] = {}
    default_colors: dict[str, str] = {}
    for entry in raw:
        name = str(entry["name"])
        label = str(entry["label"])
        color = str(entry["color"])
        if not name or not label:
            raise ValueError(f"{resolved}: entry missing name/label: {entry!r}")
        if color not in PALETTE:
            raise ValueError(f"{resolved}: figure {name!r} has unknown colour {color!r}")
        icons.append(name)
        labels[name] = label
        default_colors[name] = color
    return FigureVocab(tuple(icons), labels, default_colors)
