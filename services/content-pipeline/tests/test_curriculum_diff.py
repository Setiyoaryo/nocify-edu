from __future__ import annotations

from pathlib import Path

from content_pipeline.curriculum_diff import (
    ADDED,
    CHANGED,
    REMOVED,
    diff_cp_statements,
    extract_cp_statements,
    normalise_statement,
    statement_hash,
)

FIXTURES = Path(__file__).parent / "fixtures"


def _fixture(name: str) -> str:
    return (FIXTURES / name).read_text(encoding="utf-8")


# ------------------------------------------------------------------- normalise


def test_normalise_collapses_whitespace_and_case() -> None:
    assert normalise_statement("  Peserta   didik\ndapat ") == "peserta didik dapat"


def test_normalise_handles_nbsp_and_unicode_forms() -> None:
    # NBSP and a full-width digit both normalise under NFKC.
    assert normalise_statement("a\u00a0b") == "a b"
    assert normalise_statement("bilangan １") == "bilangan 1"


def test_statement_hash_ignores_formatting() -> None:
    assert statement_hash("Sama saja.") == statement_hash("  sama   saja. ")


# -------------------------------------------------------------------- extract


def test_extract_reads_code_separator_statement() -> None:
    text = "CP-A-MAT-BILANGAN: Membilang benda sampai 20.\nCP-A-MAT-PENJUMLAHAN - Menjumlahkan sampai 20."
    assert extract_cp_statements(text) == {
        "CP-A-MAT-BILANGAN": "Membilang benda sampai 20.",
        "CP-A-MAT-PENJUMLAHAN": "Menjumlahkan sampai 20.",
    }


def test_extract_accepts_en_dash_separator() -> None:
    assert extract_cp_statements("CP-A-MAT-BILANGAN \u2013 Membilang benda.") == {
        "CP-A-MAT-BILANGAN": "Membilang benda."
    }


def test_extract_appends_continuation_lines_until_a_blank_line() -> None:
    text = "CP-A-MAT-BILANGAN: Membilang\nbenda sampai 20.\n\nBukan lagi bagian CP."
    assert extract_cp_statements(text) == {"CP-A-MAT-BILANGAN": "Membilang benda sampai 20."}


def test_extract_ignores_bare_codes_and_malformed_lines() -> None:
    text = "CP-A-MAT-BILANGAN:\nHalaman 3\nCP-BUKAN-SEPARATOR tanpa pemisah"
    assert extract_cp_statements(text) == {}


def test_extract_first_occurrence_wins() -> None:
    text = "CP-A-MAT-BILANGAN: Versi pertama.\nCP-A-MAT-BILANGAN: Versi kedua."
    assert extract_cp_statements(text) == {"CP-A-MAT-BILANGAN": "Versi pertama."}


# ----------------------------------------------------------------------- diff


def test_diff_classifies_added_changed_removed_and_ignores_unchanged() -> None:
    previous = {"CP-A": "satu", "CP-B": "dua", "CP-C": "tiga"}
    current = {"CP-A": "satu", "CP-B": "dua diperbarui", "CP-D": "empat"}
    changes = {change.code: change for change in diff_cp_statements(previous, current)}
    assert changes["CP-B"].change_type == CHANGED
    assert changes["CP-C"].change_type == REMOVED
    assert changes["CP-D"].change_type == ADDED
    assert "CP-A" not in changes


def test_diff_is_sorted_by_code() -> None:
    changes = diff_cp_statements({}, {"CP-Z": "z", "CP-A": "a", "CP-M": "m"})
    assert [change.code for change in changes] == ["CP-A", "CP-M", "CP-Z"]


def test_diff_ignores_formatting_only_changes() -> None:
    assert diff_cp_statements({"CP-A": "Sama saja."}, {"CP-A": "  sama   saja. "}) == ()


def test_diff_of_empty_inputs_is_empty() -> None:
    assert diff_cp_statements({}, {}) == ()


def test_diff_fixtures_are_one_added_one_changed_one_removed() -> None:
    previous = extract_cp_statements(_fixture("curriculum_v1.txt"))
    current = extract_cp_statements(_fixture("curriculum_v2.txt"))
    by_type = {change.change_type: change for change in diff_cp_statements(previous, current)}
    assert set(by_type) == {ADDED, CHANGED, REMOVED}
    assert by_type[ADDED].code == "CP-A-MAT-PENGURANGAN"
    assert by_type[CHANGED].code == "CP-A-MAT-PENJUMLAHAN"
    assert by_type[REMOVED].code == "CP-A-IND-MEMBACA"
    # The bilang CP differs only in formatting, so it must not be reported.
    assert all(change.code != "CP-A-MAT-BILANGAN" for change in diff_cp_statements(previous, current))
