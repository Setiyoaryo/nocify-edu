# Content Intelligence Pipeline — Rekonsiliasi & Roadmap

**Sumber rencana:** [`nocify-edu-kurikulum-merdeka-content-crawler-planning.md`](../nocify-edu-kurikulum-merdeka-content-crawler-planning.md) (54 bagian, milestone M1–M8)
**Status:** Roadmap implementasi — belum ada kode
**Konteks repo:** produk berjalan bernama **EduPlay** di dalam repo `nocify-edu` (Go API + SvelteKit + PostgreSQL sudah live untuk TK–SD).

Dokumen ini tidak mengulang isi rencana. Isinya: (1) apa yang **sudah ada** di repo, (2) **gap & konflik** antara rencana dan kenyataan, (3) rekomendasi **boundary Python ↔ Go/PostgreSQL**, (4) **urutan implementasi konkret**, (5) keputusan yang masih perlu diambil.

---

## 1. Ringkasan keputusan

| # | Keputusan | Rekomendasi |
|---|---|---|
| D1 | Bentuk fitur | **Subsistem aditif**, deployable terpisah (`services/content-pipeline`), tidak menyentuh request path produk. Produk berjalan tidak boleh goyah. |
| D2 | Boundary data | **Satu PostgreSQL**, *schema-per-owner*: Python memiliki `content_staging`, Go tetap memiliki kanonik (`public`). Python **tidak pernah** menulis tabel kanonik. |
| D3 | Handoff | **Publish contract**: kandidat yang lolos review ditulis ke kanonik oleh **Go**, bukan oleh Python. Bukan HTTP sinkron. |
| D4 | Integritas | Kanonik **tidak boleh** FK ke `content_staging` (staging bisa di-prune). Provenance disalin sebagai data immutable ke tabel kanonik. |
| D5 | Keamanan | Role DB terpisah untuk pipeline: hanya `content_staging` (full) + `SELECT` pada taksonomi kanonik. Bug crawler tidak bisa merusak konten produksi. |
| D6 | Kurikulum | Model fase + CP + provenance ditambahkan ke kanonik (migrasi Go), **bukan** diganti. Model `chapters/skills` yang ada tetap dipakai. |
| D7 | Review | Gate akademik yang sudah ada (`question_reviews`, status workflow) **diperluas**, bukan dibuat ulang. |
| D8 | Urutan | M1–M5 dulu sampai stabil; M6–M8 hanya setelah itu (sesuai rencana §54). |

---

## 2. Kondisi repo saat ini (yang sudah ada)

| Area | Kenyataan di repo | Relevansi ke rencana |
|---|---|---|
| Backend | Go modular monolith, satu binary: `serve \| worker \| migrate \| seed \| user \| recompute` (`apps/api/cmd/eduplay/main.go`) | Rencana §6/§32 menyarankan Python untuk pipeline, Go tetap produk — cocok |
| Skema | Satu schema `public`, migrasi di `apps/api/internal/db/migrations/` (0001–0003), auto-run saat API start dengan advisory lock | Rencana §31 minta pemisahan schema — **belum ada** |
| Kurikulum | `curriculum_versions → education_levels → grades → subjects → chapters → skills` (`0001_init.sql:30`) | Mendekati §19, tapi **tanpa `phase` dan `CP`** |
| Konten kurikulum | `content/curriculum/curriculum.json` = `kurmer-2024`, label *"penyelarasan internal EduPlay"* | Bukan CP resmi — §2/§46 butuh sumber resmi + provenance |
| Soal | `questions` + `question_versions` (immutable) + `question_reviews`; status `draft→review→approved→published→retired`; ada kolom `source` (teks bebas) | Gate akademik §24/§41 **sudah ada** |
| Validasi soal | `packages/question-schema/question.schema.json` + validator Go `questions.Validate` | Bisa dipakai ulang sebagai kontrak kandidat soal (§22) |
| Aset & lisensi | `assets` + `asset_licenses`, lifecycle, constraint `assets_no_ai_in_production` (`0001_init.sql:136`) | Pola governance yang bisa dipakai ulang untuk lisensi *sumber* (§25) |
| Storage | S3-compatible (SeaweedFS dev), bucket privat, file di-stream lewat API | Siap untuk **raw archive** §11 |
| Worker | Go worker, job idempotent tiap menit (`docs/operations.md:43`) | Kandidat host antrian, tapi jangan digabung dengan pipeline Python |
| CI | 3 job: api (go test + govulncheck), web, e2e (`ci.yml`) | **Belum ada** job Python |
| Docs | `docs/architecture.md`, `decisions.md`, `operations.md`, `privacy.md` | `docs/data-model.md` dirujuk `0001_init.sql:1` tapi **tidak ada** (gap kecil) |
| Nama | repo `nocify-edu`, modul `github.com/nocify/eduplay/api`, produk **EduPlay** | Rencana memakai "Nocify Edu" — perlu disepakati istilahnya |

---

## 3. Gap & konflik (rencana vs kenyataan)

| ID | Rencana | Kenyataan | Dampak | Tindakan |
|---|---|---|---|---|
| G1 | Schema `app/content/content_staging/audit` (§31) | Satu schema `public` | Sedang | Bootstrap `content_staging` (lihat D2/D5) |
| G2 | `phases` (Fondasi/A/B/C) + `curriculum_cp` (§19) | Tidak ada fase/CP; hanya `grades` | Tinggi | Migrasi Go: `phases`, `curriculum_cp`, tautan `skills.cp_id` |
| G3 | PAUD = Fase Fondasi, bukan mapel/bab (§3) | TK dimodelkan **dengan** mapel (`content/curriculum/curriculum.json`) | Tinggi | Tambah elemen Fase Fondasi; pertahankan mapel TK untuk UX |
| G4 | `curriculum_versions` punya `revision/effective_date/source_document_id` (§2) | Hanya `code,name,is_active` | Tinggi | Perluas tabel kanonik + provenance |
| G5 | "Jangan overwrite riwayat kurikulum" (§27) | `seedCurriculum` `ON CONFLICT DO UPDATE` (`seed.go:118`) | Tinggi | Ubah seed → buat versi baru, bukan mutasi |
| G6 | Lineage soal penuh (§20) | `questions.source` = teks bebas | Tinggi | Tabel `question_provenance` di kanonik |
| G7 | Source Registry + governance (§4–5) | Tidak ada | Tinggi | `content_staging.source_registry` + status lifecycle |
| G8 | Raw archive + hash/conditional fetch (§8, §11) | Belum ada | Sedang | `crawl_runs`, `crawl_items`, `source_documents` di staging + S3 |
| G9 | Reviewer CMS untuk kandidat (§36–38) | CMS admin ada, tapi hanya untuk soal kanonik | Sedang | Perluas `admin` (Go) dengan antrian review kandidat |
| G10 | Deteksi perubahan kurikulum (§44) | Belum ada | Sedang | Scheduler + diff di pipeline (M7) |
| G11 | Creator/revenue share (§39) | Hanya `created_by` | Rendah | Tunda ke M8 |
| G12 | Referensi regulasi (§46) | Sitasi di dokumen rencana mengandung artefak (`turn…search…`) dan tanggal 2025–2026 | **Wajib verifikasi** | Verifikasi ke sumber resmi sebelum jadi kanonik (§10) |
| G13 | Struktur `services/content-pipeline` (§33, §49) | Tidak ada `services/` | Rendah | Buat saat M1 |

---

## 4. Boundary Python ↔ Go/PostgreSQL (jawaban "best practice")

Pertanyaannya: bagaimana Python dan Go berbagi data tanpa saling merusak.

**Prinsip: satu database, satu pemilik per schema, handoff lewat kontrak eksplisit.**

```text
                 PostgreSQL (satu instance, satu database)
   ┌───────────────────────────────┬──────────────────────────────────┐
   │  public  (KANONIK)            │  content_staging  (SEMENTARA)    │
   │  PEMILIK: Go                  │  PEMILIK: Python                 │
   │                               │                                  │
   │  curriculum_*  skills         │  source_registry  crawl_runs     │
   │  questions     assets         │  crawl_items      source_documents│
   │  children      progress       │  document_chunks  *_candidates   │
   │  ...produk berjalan...        │  publish_queue    jobs           │
   └───────────────▲───────────────┴──────────────▲───────────────────┘
                   │ INSERT/UPDATE (hanya via Go)  │ INSERT/UPDATE (Python)
                   │                               │ SELECT taksonomi (read-only)
                   │        ┌──────────────────────┘
                   │        │
              ┌────┴────────┴────┐
              │  Go: publish      │  ← membaca publish_queue yang APPROVED,
              │  command/worker   │    validasi ulang, tulis kanonik + provenance
              └───────────────────┘
```

### Aturan konkret

1. **Kepemilikan tabel.** Python menulis hanya di `content_staging`. Go menulis `public`. Tidak ada tabel yang ditulis dua service.
2. **Role DB terpisah.**
   - `content_pipeline`: `ALL` pada schema `content_staging`; `SELECT` pada `curriculum_versions, education_levels, grades, subjects, chapters, skills`; **tanpa** hak tulis ke kanonik.
   - `eduplay` (existing): akses penuh ke `public`; `SELECT` pada `content_staging` (untuk CMS review & publish).
3. **Publish = operasi Go.** Kandidat yang di-approve masuk `content_staging.publish_queue`. Perintah baru `eduplay publish-candidates` (atau job di worker) membaca antrian, **memvalidasi ulang** dengan `questions.Validate` + `packages/question-schema`, lalu menulis kanonik dalam satu transaksi per batch, mencatat `question_reviews` + `audit_logs`.
   - Kenapa bukan Python yang menulis kanonik: validasi akhir dan gate produksi harus berada di pemilik data kanonik. Ini juga menegakkan aturan §41/§48 "tidak ada shortcut FETCHED → PUBLISHED".
4. **Validasi ganda (sengaja).** Python memvalidasi saat menyimpan kandidat (umpan balik awal, murah); Go memvalidasi lagi saat publish (gate terakhir, otoritatif). Satu kontrak dipakai dua sisi: `packages/question-schema` untuk payload soal, dan `packages/contracts/` untuk amplop kandidat.
5. **Tanpa FK lintas schema.** Kanonik **tidak** mereferensikan `content_staging`. Provenance (URL, dokumen, halaman, hash) **disalin** ke tabel kanonik (`question_provenance`, `curriculum_source_documents`) saat publish. Alasan: staging bersifat sementara dan bisa dibersihkan/di-rebuild; kanonik harus tetap utuh dan auditable selamanya (§20).
6. **Migrasi per pemilik.** Go tetap menjalankan migrasi kanonik (runner yang ada). Go juga yang mem-bootstrap batasnya sekali: `CREATE SCHEMA content_staging`, role, dan grant — karena Go yang memiliki bootstrap DB dan backup. Setelah itu, tabel di dalam `content_staging` dikelola Python (Alembic, atau SQL + runner kecil). Alternatif "semua migrasi di Go" bisa dipakai kalau tim ingin satu runner; konsekuensinya setiap perubahan tabel staging butuh rilis Go. **Rekomendasi: Alembic milik Python.**
7. **Antrian & scheduler.** Queue di Postgres (`content_staging.jobs`, `SELECT … FOR UPDATE SKIP LOCKED`, sesuai §35). Scheduler milik pipeline (cron/APScheduler di container Python). Jangan menumpangkan crawler ke loop worker Go — beda lifecycle dan beda bahasa.
8. **Deployment terpisah.** Pipeline masuk compose sebagai **profile** (`--profile pipeline`) atau file compose sendiri, bukan di `compose.yaml` default. Produk harus tetap bisa `make up` tanpa Python.
9. **Tidak ada HTTP Go↔Python untuk MVP.** Boundary = DB + kontrak. Hindari coupling sinkron (§32).

**Kapan naik ke database terpisah:** hanya kalau ada kebutuhan nyata (isolasi compliance, volume, atau tim terpisah). Untuk MVP, satu DB lebih baik: satu backup, satu restore, satu transaksi publish, tanpa copy lintas-DB.

---

## 5. Arsitektur target (direkonsiliasi)

```text
   Python: services/content-pipeline          Go: apps/api (produk + publish)
   ────────────────────────────────           ──────────────────────────────
   crawl → extract → normalize                serve (produk berjalan)
        → chunk → map ke CP/skill              admin CMS (review kandidat)
        → kandidat material/soal               publish-candidates ←──┐
        → QA otomatis                          worker (job produk)    │
        → tulis content_staging ───────────────┘                     │
                          │                                          │
                          └── review manusia (CMS) ── approve ───────┘
                                                                     │
                                          tulis kanonik (public) ◄───┘
                                                                     │
   S3: raw/ (arsip dokumen sumber)                                   ▼
                                                        produk anak/orang tua
```

---

## 6. Roadmap M1–M8 (konkret)

Setiap milestone: deliverable, artefak, DoD. Urutan ini **menggantikan** §47/§54 rencana karena sudah disesuaikan ke repo.

### M1 — Source Registry + Official Crawler
- **Deliverable:** service Python jalan, bisa crawl allowlist resmi, arsip raw ke S3.
- **Artefak:** `services/content-pipeline/` (pyproject, CLI `crawl`), `content_staging.source_registry`, `crawl_runs`, `crawl_items`; S3 prefix `raw/`.
- **DoD:** registry data-driven (bukan hardcoded §4); conditional fetch (ETag/Last-Modified/SHA-256 §8); run tercatat §9; resume aman; allowlist + proteksi SSRF §42; robots/ToS diperiksa sebelum `active` §43.
- **Guardrail:** default source `paused` sampai di-review.
- **Status:** ✅ **selesai (2026-09-30)** — lihat [`services/content-pipeline/`](../services/content-pipeline/README.md).
  - Terverifikasi: `ruff`/`mypy`/`pytest` (38 tes) hijau; migrasi idempoten; crawl nyata ke `kurikulum.kemendikdasmen.go.id` (robots dihormati, 3 halaman, `crawl_runs`/`crawl_items` terisi).
  - Deviasi kecil: `crawl_items.storage_key` ditambahkan (arsip raw M1, di luar §10); runner migrasi = SQL biasa, bukan Alembic (dapat diganti saat skema bercabang).
  - Belum diverifikasi: arsip S3 (butuh run di dalam jaringan compose) dan resume setelah run terputus (M2).

### M2 — Extraction + Raw Archive
- **Deliverable:** dokumen mentah → teks terstruktur → chunk.
- **Artefak:** `source_documents`, `document_chunks`; modul `parsers/` (HTML, PDF, DOCX, TXT, MD).
- **DoD:** deteksi tipe file; PDF text-layer, OCR sebagai *fallback* §13; chunk mengikuti heading→section→paragraf, fallback semantik §15; setiap chunk menyimpan halaman/urutan/hash §14; raw bisa di-reproses §51.
- **Status:** ✅ **selesai (2026-09-30)**.
  - Terverifikasi: `ruff`/`mypy`/`pytest` (64 tes) hijau; `extract` in-container: 4 dokumen HTML → 12 chunk dengan heading benar; re-run idempoten; `--force` mengganti chunk tanpa duplikasi.
  - Parser: HTML (heading → section, tabel), Markdown/teks, PDF (per halaman, deteksi heading via ukuran font; PDF tanpa text layer → `needs_ocr`), DOCX. Deteksi tipe pakai magic bytes.
  - OCR **belum** diimplementasi (hanya dideteksi & ditandai), sesuai §13 "OCR adalah fallback".
  - Catatan: ekstraksi butuh raw archive (S3) karena membaca dari `crawl_items.storage_key`.

### M3 — Curriculum Taxonomy + CP Mapping
- **Deliverable:** kanonik punya fase + CP + provenance; pipeline bisa memetakan chunk → CP/skill.
- **Artefak (kanonik, migrasi Go):** `phases`, `curriculum_cp`, `skills.cp_id` (nullable), `curriculum_versions` + `revision/effective_date/source_document_id`, `curriculum_source_documents`.
- **Artefak (staging):** `mapping_candidates` + confidence §18.
- **DoD:** Fase Fondasi (PAUD) tidak dipaksa ke mapel/bab §3; Fase A/B/C SD §2; revisi tidak menimpa riwayat §27; seed berhenti `DO UPDATE` buta (G5).
- **Catatan:** ini milestone paling berisiko karena menyentuh kanonik — kerjakan di balik migrasi aditif (kolom nullable), bukan rename.

### M4 — Material & Question Candidates
- **Deliverable:** kandidat materi dan soal dengan lineage lengkap.
- **Artefak:** `material_candidates`, `question_candidates`; amplop kontrak di `packages/contracts/`.
- **DoD:** setiap kandidat punya `source_chunk_id`; payload soal lolos `question-schema`; template deterministik untuk tipe terstruktur, AI hanya untuk kandidat §23–24; QA otomatis §40 (missing source/CP/skill, duplikat, opsi kosong, dsb).

### M5 — Reviewer CMS + Publish Workflow
- **Deliverable:** reviewer melihat sumber + kandidat + mapping dalam satu layar; approve → publish ke kanonik.
- **Artefak:** perluasan modul `admin` (Go): antrian kandidat, layar review §37, provenance §38; `content_staging.publish_queue`; perintah `eduplay publish-candidates`.
- **DoD:** publish hanya lewat Go (D3); provenance tersalin ke kanonik (D4); memakai `question_reviews` + status workflow yang ada; `audit_logs` mencatat publish.
- **Gerbang:** setelah M5 stabil, baru boleh mulai M6.

### M6 — Semantic Mapping + Deduplikasi
- Embeddings + klasifikasi + dedup (URL kanonik + SHA-256 + fingerprint + similaritas §29).

### M7 — Curriculum Change Detection
- Watcher sumber resmi → diff dokumen → CP terdampak → skill/soal terdampak → review queue §44.

### M8 — Creator / Revenue Attribution
- `creator_id`, usage/completion/quality, adaptation lineage §39; marketplace §47 Fase 5.

---

## 7. Daftar perubahan skema

### Kanonik (migrasi Go, aditif)
- `curriculum_versions`: `+ revision`, `+ effective_date`, `+ source_document_id`.
- Baru: `curriculum_source_documents` (provenance immutable), `phases`, `curriculum_cp`.
- `skills`: `+ cp_id` (nullable).
- Baru: `question_provenance` (question_id, cp_id, skill_id, source_document_ref, original_url, retrieved_at).
- M8: `questions.creator_id` + tabel atribusi.

### Staging (dikelola Python)
- `source_registry`, `crawl_runs`, `crawl_items`, `source_documents`, `document_chunks`, `mapping_candidates`, `material_candidates`, `question_candidates`, `publish_queue`, `jobs`.
- Semua status/lifecycle sesuai §5, §9, §21, §41.

---

## 8. Guardrail (yang tidak boleh dilakukan)

Mengadopsi §48 rencana, ditambah aturan repo ini:

- Tidak crawl seluruh internet; tidak bypass robots.txt / ToS.
- Tidak publish soal hasil AI tanpa review manusia.
- Tidak menyalin karya berhak cipta tanpa lisensi.
- Tidak menimpa versi kurikulum lama.
- Tidak mencampur staging dan kanonik (tidak ada FK lintas schema, tidak ada tulis silang).
- Tidak menaruh pipeline di request path produk; produk harus tetap `make up` tanpa Python.
- Tidak menaruh secret di git.

---

## 9. Keputusan (disepakati 2026-09-30)

| # | Keputusan | Nilai |
|---|---|---|
| 1 | Istilah produk | **EduPlay** untuk artefak baru ("Nocify Edu" tetap nama repo/perusahaan) |
| 2 | Acuan kurikulum | **Internal dulu**; dokumen resmi jadi referensi, provenance disimpan agar migrasi ke CP resmi nanti tidak bongkar ulang |
| 3 | Review manusia | **Wajib**; role `reviewer` diisi tim kurikulum. Tanpa reviewer, kandidat tetap di status `review` |
| 4 | Tooling Python | `uv` + `ruff` + `mypy` + `pytest` |
| 5 | Hosting | **1 VPS dulu** (container terpisah, profile `pipeline`); siap dipecah nanti |
| 6 | M8 revenue share | **Ditunda** sampai M1–M5 stabil |
| 7 | Publish | **CLI** `eduplay publish-candidates` dulu; endpoint admin menyusul |
| 8 | Allowlist awal | 5 sumber resmi (§45), semua `discovered`/disabled; sumber lain boleh ditambah setelah review |


---

## 10. Referensi regulasi — wajib diverifikasi

Rencana §46 memuat sitasi dengan artefak tidak valid (mis. `turn353788search…`) dan tanggal 2025–2026. Sebelum dipakai sebagai kanonik:

- Verifikasi setiap nomor keputusan/peraturan ke **sumber resmi** (`kurikulum.kemendikdasmen.go.id`, `peraturan.bpk.go.id`).
- Catat `source_document_id`, URL, dan tanggal pengambilan (§20).
- Jangan jadikan teks rencana ini dasar akademik sebelum terverifikasi.

---

## Lampiran — Pemetaan istilah

| Rencana | Repo ini |
|---|---|
| Curriculum framework/version | `curriculum_versions` |
| Education level / Grade / Subject | `education_levels` / `grades` / `subjects` |
| CP | **belum ada** → `curriculum_cp` |
| Phase | **belum ada** → `phases` |
| Topic / Skill | `chapters` / `skills` |
| Material/Question Candidate | **belum ada** → `content_staging.*_candidates` |
| Published Content | `questions` + `question_versions` (status `published`) |
| Human Review | `question_reviews` + status workflow |
| Source Registry | **belum ada** → `content_staging.source_registry` |
| Raw Archive | S3 `raw/` + `content_staging.source_documents` |
