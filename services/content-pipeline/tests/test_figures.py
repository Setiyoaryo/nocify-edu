import json
from pathlib import Path

import pytest

from content_pipeline.figures import (
    PALETTE,
    VOCAB_RELATIVE,
    FigureVocab,
    find_figures_json,
    load_vocab,
)


def test_find_figures_json_resolves_from_repo() -> None:
    found = find_figures_json()
    assert found is not None
    assert found.name == VOCAB_RELATIVE.name
    assert found.is_file()


def test_load_vocab_reads_shared_file() -> None:
    vocab = load_vocab()
    assert isinstance(vocab, FigureVocab)
    assert len(vocab.icons) >= 80
    # Names the pipeline relies on for TK/Fase A object counting.
    for name in ("orange", "carrot", "cheese", "clock", "triangle"):
        assert vocab.has(name), f"{name} missing from the vocabulary"
    assert vocab.label("orange") == "jeruk"
    assert vocab.default_color("orange") == "orange"
    # Unknown names never crash; they fall back.
    assert vocab.label("nope") == "nope"
    assert vocab.default_color("nope") == "gray"


def test_every_colour_is_in_the_shared_palette() -> None:
    vocab = load_vocab()
    for name in vocab.icons:
        assert vocab.default_color(name) in PALETTE


def test_env_override_wins(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    custom = tmp_path / "figures.json"
    custom.write_text(
        json.dumps([{"name": "widget", "label": "widget", "color": "blue"}]),
        encoding="utf-8",
    )
    monkeypatch.setenv("FIGURES_PATH", str(custom))
    vocab = load_vocab()
    assert vocab.icons == ("widget",)
    assert vocab.label("widget") == "widget"


def test_rejects_unknown_colour(tmp_path: Path) -> None:
    bad = tmp_path / "figures.json"
    bad.write_text(
        json.dumps([{"name": "widget", "label": "widget", "color": "chartreuse"}]),
        encoding="utf-8",
    )
    with pytest.raises(ValueError, match="unknown colour"):
        load_vocab(bad)


def test_missing_file_is_reported(monkeypatch: pytest.MonkeyPatch, tmp_path: Path) -> None:
    monkeypatch.setenv("FIGURES_PATH", str(tmp_path / "nope.json"))
    assert find_figures_json() is None
    with pytest.raises(FileNotFoundError):
        load_vocab()
