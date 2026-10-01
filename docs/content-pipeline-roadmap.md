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
3. **Publish = operasi Go.** Kandidat yang di-approve masuk antrian publish. Perintah `eduplay publish-candidates` (atau job di worker) membaca antrian, **memvalidasi ulang** dengan `questions.Validate` + `packages/question-schema`, lalu menulis kanonik dalam satu transaksi per kandidat, mencatat `question_reviews` + `audit_logs`.
   - **Realisasi M5:** antrian = tabel kanonik `candidate_publications`, bukan `content_staging.publish_queue` (lihat M5 §6 dan §9 keputusan 9).
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
- **Status:** ✅ **selesai (2026-09-30)**.
  - Kanonik (migrasi Go `0004_curriculum_cp.sql`, aditif): `curriculum_source_documents`, `phases`, `grades.phase_id`, `curriculum_cp`, `skills.cp_id`, `curriculum_versions.revision/effective_date/source_document_id`.
  - Data: `content/curriculum/phases.json` + `cp.json`; seed mengisi 4 fase, 9 CP, dan menautkan 18 skill ke CP. Terverifikasi di DB dev.
  - G3 selesai: CP Fase Fondasi ber-`subject_id` NULL (PAUD tidak dipaksa ke mapel).
  - G5 selesai: seed tidak lagi `DO UPDATE` buta — `DO NOTHING`/no-op update, jadi edit CMS tidak tertimpa.
  - Staging: `mapping_candidates` + mapper deterministik/leksikal (CLI `map`). Verifikasi: 12 chunk → 6 mapped / 6 unmapped, semua band `manual` (benar — teks regulasi tidak cocok dengan skill TK–SD), `--force` idempoten.
  - Catatan: layer embedding/LLM baru di M6; belum ada perubahan API/CMS Go (data fase/CP baru terlihat via SQL sampai M5).

### M4 — Material & Question Candidates
- **Deliverable:** kandidat materi dan soal dengan lineage lengkap.
- **Artefak:** `material_candidates`, `question_candidates`; amplop kontrak di `packages/contracts/`.
- **DoD:** setiap kandidat punya `source_chunk_id`; payload soal lolos `question-schema`; template deterministik untuk tipe terstruktur, AI hanya untuk kandidat §23–24; QA otomatis §40 (missing source/CP/skill, duplikat, opsi kosong, dsb).
- **Status:** ✅ **selesai (2026-09-30).**
  - Staging material: `material_candidates` (`sql/0004_materials.sql`) — satu baris per chunk, idempoten (`UNIQUE (chunk_id)`).
  - Staging soal: `question_candidates` (`sql/0005_question_candidates.sql`) — satu baris per template per chunk (`UNIQUE (chunk_id, template)`), `payload jsonb` berisi amplop soal penuh.
  - `readability.py` (§30): `age_band`, `reading_level`, `complexity`, hitungan kata/kalimat, `visual_required`/`audio_required`; heuristik dari fase/grade + metrik teks.
  - `materials.py` (§21/§29/§40): judul/ringkasan/tujuan deterministik, `difficulty_hint` 1–5, fingerprint SHA-256 + `duplicate_of`, QA otomatis; issue blocking → `needs_review`, selain itu `draft`.
  - `questions.py` (§22/§23/§40): template deterministik **cloze berbasis sumber** (tanpa AI) — kalimat kehilangan istilah paling menonjol (istilah kurikulum bila ada, jika tidak kata konten terpanjang); band awal → `single_choice` dengan distraktor dari chunk yang sama, band atas → `short_answer`. `validation.py` memvalidasi `payload` terhadap `packages/question-schema/question.schema.json` **sebelum** disimpan; payload gagal tidak masuk staging.
  - QA soal §40: `missing_skill`, `missing_cp_mapping`, `duplicate`, `schema_invalid`, `empty_answer`, `invalid_options`, `answer_mismatch` (jawaban harus ada di sumber), `prompt_too_long`, `low_confidence`.
  - Kontrak: `packages/contracts/material-candidate.schema.json`, `packages/contracts/question-candidate.schema.json`.
  - Terverifikasi: `ruff`/`mypy`/`pytest` (133 tes) hijau; end-to-end di dev stack — 11 chunk → 5 mapped → 5 material + 5 soal (semua `draft`, `invalid=0`), payload lolos schema; re-run tanpa `--force` no-op, `--force` mengganti in-place; duplikat teks terdeteksi (`duplicate_of` + `needs_review`) pada simulasi; jalur error schema-missing mengembalikan exit 2.
  - Catatan: layer embedding/LLM, dedup semantik, dan **assisted (AI) generation** §23 (scenario/comprehension) baru di M6. Template deterministik sengaja sempit: hanya menghasilkan soal recognition/vocabulary yang benar-benar ter-ground di sumber; soal skenario menunggu M6 + review manusia §24.

### M5 — Reviewer CMS + Publish Workflow
- **Deliverable:** reviewer melihat sumber + kandidat + mapping dalam satu layar; approve → publish ke kanonik.
- **Artefak:** perluasan modul `admin` (Go): antrian kandidat, layar review §37, provenance §38; **ledger kanonik** `candidate_publications` + `question_provenance`; perintah `eduplay publish-candidates`.
- **DoD:** publish hanya lewat Go (D3); provenance tersalin ke kanonik (D4); memakai `question_reviews` + status workflow yang ada; `audit_logs` mencatat publish.
- **Gerbang:** setelah M5 stabil, baru boleh mulai M6.
- **Status:** ✅ **selesai (2026-09-30).**
  - **Koreksi rencana (penting):** `content_staging.publish_queue` **tidak dipakai**. Antrian = tabel kanonik `candidate_publications` (`apps/api/internal/db/migrations/0005_candidate_publications.sql`), karena `roles_grants.sql` memberi Go hanya `SELECT` pada `content_staging` — Go tidak boleh menulis antrian di schema milik Python (D2/D5). Ledger ini sekaligus menyimpan keputusan review (`approved`/`changes_requested`/`rejected`), `checklist`, status publish, percobaan, dan `last_error`. Keputusan tetap ditulis lewat API Go, bukan oleh pipeline.
  - Migrasi kanonik `0005`: `candidate_publications` (`UNIQUE (kind, candidate_id)`, FK `decided_by → users`, `canonical_question_id → questions`) + `question_provenance` (PK `question_id`, FK `curriculum_cp`/`skills`; `source_document_id`/`source_chunk_id` **tanpa FK** — staging bisa di-prune, D4).
  - API (Go, `internal/candidates`): `GET /admin/candidates` (antrian + `counts`, degradasi `available:false` bila pipeline belum jalan), `GET /admin/candidates/{kind}/{id}` (lineage sumber + mapping + QA + publikasi), `POST /admin/candidates/{kind}/{id}/decision` (reviewer/admin; `approved` butuh checklist lengkap — memakai `admin.ChecklistComplete`, 11 item yang sama; 409 bila sudah `published`/`publishing`).
  - Publish (`internal/candidates/publish.go`): `eduplay publish-candidates` (`--limit`, `--all`, `--dry-run`, `--retry-failed`, `--max-attempts`) mengklaim antrian `queued` dengan `FOR UPDATE SKIP LOCKED` (satu tx per kandidat), memvalidasi ulang lewat `admin.CreateQuestion` (schema soal kanonik), menulis `question_provenance`, `question_reviews`, dan `audit_logs`. Idempoten: kode soal = `code` kandidat, kandidat yang sudah terbit tidak diproses ulang. Ada `reapStale` (publishing macet) + `requeueFailed`.
  - **Materi bersifat review-only untuk sekarang:** belum ada tabel kanonik materi, jadi keputusan material disimpan di ledger tanpa publish (status `not_applicable`), sesuai keputusan sesi.
  - Web (`apps/web`): `/app/admin/review` (antrian + filter kind/status/q) dan `/app/admin/review/[kind]/[id]` (sumber, pratinjau soal via `QuestionRenderer` read-only, mapping, QA blocking/advisory, form keputusan + checklist); helper di `src/lib/candidates.ts`.
  - Terverifikasi: unit + DB-gated publish tests Go hijau (`go test -p 1 ./...` dengan `TEST_DATABASE_URL`); `svelte-check` 0 error, vitest hijau; end-to-end di dev stack (keputusan → publish kanonik → idempoten → provenance tercatat).
  - Catatan: layer embedding/LLM, dedup semantik, dan assisted generation §23 tetap di M6. Endpoint admin untuk menjalankan publish dari UI menyusul (keputusan 7: CLI dulu).

### M6 — Semantic Mapping + Deduplikasi + Assisted Generation
- **Deliverable:** layer embedding/semantik untuk mapping dan dedup (§17–§18, §29), plus generasi soal *scenario*/*comprehension* berbantuan AI yang selalu lewat review manusia (§23–§24).
- **Artefak (staging):** `embeddings`, `duplicate_relations`, `generation_ledger` (`sql/0006_semantic.sql`) + kolom `ai_generated` pada `question_candidates`/`material_candidates`.
- **Artefak (kode):** `provider.py` (abstraksi provider OpenAI-compatible + `mock`), `similarity.py` (cosine + ambang), `embeddings.py` (cache + `Embedder`), `mapping.py` (blend hybrid), `dedup.py`, `assisted.py`; CLI `embed` / `map --semantic` / `dedup` / `generate`; kontrak kandidat + `ai_generated` di Go (`internal/candidates`) dan web (badge review).
- **DoD:** §29 empat status dedup; §18 confidence tidak pernah jadi gate publish; §24 output AI tidak pernah auto-approve/publish; biaya dibatasi (cache ketat + cap + `--dry-run`); lisensi `unknown` dilewati (§25).
- **Status:** ✅ **selesai (2026-10-01)** — dua commit di branch `feat/content-pipeline-m6` (stacked di M5/PR #8).
  - **Migrasi `0006_semantic.sql`** (aditif, idempoten): `embeddings` (polimorfik `chunk`/`skill`, `UNIQUE (owner_kind, owner_id, model)`, `vector double precision[]` — tanpa pgvector, cosine dihitung Python), `duplicate_relations` (`status` ∈ `exact|near_duplicate|related|unique`, `signal` ∈ `semantic|hash`, `UNIQUE (chunk_id)`), `generation_ledger` (`cache_key` UNIK = `sha256(chunk_hash|template|model|prompt_version)`, token, `attempts`, `raw_output jsonb`, `status` ∈ `pending|generated|failed|skipped|rejected`), dan `ai_generated boolean` di kedua tabel kandidat.
  - **Provider** (`provider.py`): satu antarmuka `embed`/`complete` untuk `OpenAICompatibleProvider` (`{base}/embeddings`, `{base}/chat/completions`, httpx) dan `MockProvider` deterministik (offline, tanpa key — meniru `PAYMENT_PROVIDER=mock` Go). Client HTTP injectable → tes tanpa jaringan. Env: `AI_PROVIDER` (`""`/`mock`/`openai`), `AI_BASE_URL`, `AI_API_KEY`, `AI_EMBEDDING_MODEL`, `AI_CHAT_MODEL`, `AI_TIMEOUT_SECONDS`, `AI_MAX_OUTPUT_TOKENS`, `AI_GENERATION_CAP`, `AI_EMBEDDING_BATCH_SIZE`, `AI_PROMPT_VERSION`. Compose dev default `AI_PROVIDER=mock`.
  - **Similaritas & mapping hybrid** (`similarity.py`, `mapping.py`): `semantic_confidence = clamp((cos−0.55)/0.45, 0, 1)`; `confidence = max(lexical, 0.5·(lexical+semantic))` — sinyal semantik **hanya boleh menaikkan**, tidak pernah menurunkan skor leksikal (menjaga band §18 dan hasil lama). Ambang dedup: **≥ 0.92** `near_duplicate` (blocking), **0.80–0.92** `related` (advisory), **< 0.80** `unique`; konstanta modul agar bisa disetel per model.
  - **Dedup** (`dedup.py`): hash identik → `exact` mengalahkan similaritas; hasil disalin ke kandidat (`duplicate_of` + `qa_issues`), **tanpa menimpa** keputusan manusia (hanya `draft`/`needs_review` yang dihitung ulang).
  - **Assisted generation** (`assisted.py`): server yang menyusun payload — model hanya mengisi teks (prompt, jawaban diterima, penjelasan, tujuan, difficulty); `code`/`skill` dirakit dari id kanonik; tipe jawaban dipatok `short_answer` agar grounding bisa diperiksa. Guardrail: sumber dibungkus `<SUMBER>…</SUMBER>` sebagai **DATA tidak tepercaya** (control char dibuang, dipotong), jawaban harus muncul di sumber (`answer_mismatch`), prompt harus memakai kosakata sumber (`ungrounded`), dan setiap kandidat AI selalu membawa `ai_generated_unverified` **blocking** + `ai_generated=true` + `created_by='pipeline:ai'` → tidak pernah keluar dari `needs_review` tanpa manusia. Cache ketat + `--dry-run` (tanpa panggilan/tanpa tulis) + `--cap`; kegagalan provider dicatat per-item (`failed` + `attempts`), berhenti setelah 3 kegagalan berturut, dan bisa dilanjutkan dari cache.
  - **Kontrak & UI:** `question-candidate.schema.json` menerima `scenario`/`comprehension` + kode QA `near_duplicate`, `related`, `ai_generated_unverified`, `ungrounded` + `ai_generated`; `material-candidate.schema.json` menerima kode dedup + `ai_generated`. Go `internal/candidates` mengekspos `ai_generated` di detail (kolom ditambah di kedua SQL detail, scan posisional); web menampilkan badge "Dibuat AI — wajib review".
  - **Terverifikasi (unit):** `ruff` bersih; `mypy --strict` bersih (56 file); `pytest` **188 tes** hijau (tes DB-free/network-free: `MockTransport` + provider/store palsu); `make test-api` (Go, unit + integrasi DB) hijau; `svelte-check` 0 error; `vitest` hijau.
  - **Terverifikasi (end-to-end, dev stack `AI_PROVIDER=mock`):** `migrate` menerapkan `0006`; `embed` → 11 chunk + 18 skill; `map --semantic` → 5 mapped (mock ber-vektor ortogonal, jadi blend jatuh ke `lexical` — perilaku benar; klasifikasi hybrid diuji unit dengan vektor buatan); `dedup` → 11 relasi tersimpan; `generate --dry-run` **tanpa** `--allow-unknown-license` → `license_skipped=10`, **dengan** flag → `would_generate=10` (nol tulis: ledger & kandidat AI tetap 0); `generate --allow-unknown-license` → 10 kandidat (5 scenario + 5 comprehension) semua `needs_review`/`ai_generated`, ledger `generated` + token; **cache ketat**: hapus satu kandidat lalu jalankan ulang → `cached=1`, total token tidak berubah (5.199); API `GET /admin/candidates/question/{id}` mengembalikan `ai_generated=true` (cloze deterministik → `false`); gerbang approve: 422 tanpa checklist, 200 dengan checklist penuh → `publish-candidates` menulis soal kanonik + `question_provenance` (dry-run dulu, lalu real, idempoten).
  - **Batasan yang diketahui:** `MockProvider` memakai vektor sha256 (teks berbeda → ortogonal), jadi pada korpus dev ambang semantik tidak pernah terlampaui — ini disengaja; deteksi near-duplicate/related yang sebenarnya memerlukan model embedding nyata (`AI_PROVIDER=openai`). Layer embedding/LLM tidak menyentuh request path produk; produk tetap `make up` tanpa Python.

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
- Baru (M5): `candidate_publications` — ledger review + antrian publish (`kind`, `candidate_id`, `decision`, `status`, `checklist`, `decided_by`, `canonical_question_id`, `attempts`, `last_error`).
- M8: `questions.creator_id` + tabel atribusi.

### Staging (dikelola Python)
- `source_registry`, `crawl_runs`, `crawl_items`, `source_documents`, `document_chunks`, `mapping_candidates`, `material_candidates`, `question_candidates`, `jobs`.
- M6: `embeddings` (cache vektor polimorfik), `duplicate_relations` (status dedup §29), `generation_ledger` (cache + audit generasi AI); kolom `ai_generated` pada `question_candidates`/`material_candidates`.
- Tidak ada `publish_queue`: antrian publish tinggal di kanonik (`candidate_publications`, M5) karena Go read-only pada staging (D2/D5).
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
| 9 | Antrian publish (M5) | **Ledger kanonik `candidate_publications`**, bukan `content_staging.publish_queue` — Go read-only pada staging (D2/D5) |
| 10 | Materi saat M5 | **Review-only** (belum ada tabel kanonik materi); publish kanonik baru untuk soal |


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
