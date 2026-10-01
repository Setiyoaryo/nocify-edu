from __future__ import annotations

from content_pipeline.parsers import parse
from content_pipeline.parsers.html import parse_html

HTML = b"""
<html lang="id-ID"><head><title>Panduan Kurikulum</title>
<style>.x{color:red}</style></head>
<body>
  <nav>menu yang harus dibuang</nav>
  <h1>Bilangan</h1>
  <p>Peserta didik membandingkan bilangan sampai 100.</p>
  <p>Mereka juga mengurutkan bilangan.</p>
  <h2>Latihan</h2>
  <ul><li>Urutkan 3, 1, 2</li><li>Bandingkan 45 dan 54</li></ul>
  <table><tr><th>Angka</th><th>Nama</th></tr><tr><td>10</td><td>Sepuluh</td></tr></table>
  <script>alert('x')</script>
</body></html>
"""


def test_html_sections_follow_headings() -> None:
    document = parse_html(HTML)
    assert document.document_type == "html"
    assert document.title == "Panduan Kurikulum"
    assert document.language == "id"
    assert [s.heading for s in document.sections] == ["Bilangan", "Latihan"]
    assert [s.level for s in document.sections] == [1, 2]


def test_html_drops_script_style_and_nav() -> None:
    document = parse_html(HTML)
    combined = "\n".join(s.text for s in document.sections)
    assert "alert" not in combined
    assert "color:red" not in combined
    assert "menu yang harus dibuang" not in combined


def test_html_keeps_list_items_and_tables() -> None:
    document = parse_html(HTML)
    latihan = document.sections[1].text
    assert "Urutkan 3, 1, 2" in latihan
    assert "Angka | Nama" in latihan


def test_parse_dispatch_uses_content_type() -> None:
    document = parse(HTML, "text/html")
    assert document.document_type == "html"
    assert document.sections
