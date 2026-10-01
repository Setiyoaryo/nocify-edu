# Nocify Edu — Kurikulum Merdeka Content Crawler & Content Intelligence Planning

**Document type:** Product + Technical Planning  
**Status:** Proposed Architecture / MVP Planning  
**Scope:** TK–SD MVP, extensible to SMP–SMA  
**Recommended implementation:** Python Content Intelligence Pipeline + existing Go application backend + PostgreSQL

---

## 1. Executive Summary

Fitur **Content Crawler** sebaiknya tidak diposisikan sebagai scraper yang mengambil semua materi dari internet. Fitur ini harus menjadi **Content Intelligence Pipeline** yang mengambil sumber yang sah dan relevan, mengarsipkannya, mengekstrak materi, memetakan ke Kurikulum Merdeka, lalu menghasilkan kandidat materi dan soal yang tetap melalui academic review sebelum dipublikasikan.

```text
Official / Licensed Sources
        ↓
Source Registry
        ↓
Crawler / Fetcher
        ↓
Raw Archive
        ↓
Document Extraction
        ↓
Normalization
        ↓
Curriculum Mapping
        ↓
Material / Skill Candidates
        ↓
Question Candidates
        ↓
Automatic QA
        ↓
Human Review
        ↓
Published Content
        ↓
Nocify Edu App
```

Tujuan akhirnya:

> **Sumber pendidikan → knowledge base terstruktur → assessment berkualitas → progress belajar**

---

# 2. Mengapa Kurikulum Merdeka Menjadi Canonical Framework

Nocify Edu sebaiknya menyimpan Kurikulum Merdeka sebagai **framework yang berversi**, bukan sekadar label `kurikulum = merdeka`.

Kurikulum Merdeka secara resmi menekankan materi esensial, fleksibilitas, kompetensi, dan pembelajaran yang dapat disesuaikan dengan kebutuhan murid. citeturn353788search2

Keputusan Kepala BSKAP Nomor 046/H/KR/2025 menetapkan Capaian Pembelajaran (CP) untuk PAUD dan berbagai jenjang pendidikan, dengan CP SD disusun berbasis fase: Fase A untuk kelas I–II, Fase B untuk kelas III–IV, dan Fase C untuk kelas V–VI. citeturn353788search24turn353788search27

Pada Juni 2026 terbit Keputusan Kepala BKPDM Nomor 020 Tahun 2026 sebagai perubahan terhadap 046/H/KR/2025. Siaran resmi Kemendikdasmen menyatakan perubahan tersebut hanya berlaku untuk mata pelajaran Pendidikan Agama dan Budi Pekerti. citeturn353788search0turn211711search3

### Implikasi teknis

Setiap CP wajib memiliki:

```text
curriculum_id
curriculum_version
revision
source_document_id
effective_date
education_level
phase
grade (nullable untuk PAUD)
subject
cp_statement
```

Dengan ini perubahan regulasi dapat diproses sebagai **versi baru**, bukan overwrite data lama.

---

# 3. TK / PAUD Harus Dipisahkan dari Struktur SD

## PAUD / TK

Jangan memaksakan struktur:

```text
TK → Kelas → Mata Pelajaran → Bab
```

Gunakan pendekatan:

```text
PAUD
└── Fase Fondasi
    ├── Elemen / aspek perkembangan
    ├── Kemampuan fondasi
    ├── Tujuan pembelajaran
    └── Aktivitas / assessment
```

Sumber resmi menjelaskan bahwa CP PAUD menggunakan **Fase Fondasi** dan dikembangkan sesuai tahapan perkembangan anak; kompetensi fondasi mencakup antara lain sosial-emosional, bahasa, kognitif, fisik-motorik, dan nilai-nilai. citeturn353788search28

## SD

Gunakan:

```text
SD
→ Fase
→ Grade
→ Subject
→ CP
→ Topic
→ Skill
→ Assessment
```

---

# 4. Source Registry

Crawler tidak boleh memiliki daftar URL yang hardcoded di source code.

Gunakan tabel:

```text
source_registry
```

Fields:

```text
id
name
publisher
base_url
source_type
license_type
terms_url
robots_policy
crawl_frequency
priority
enabled
last_crawled_at
```

### Source tier

```text
TIER 1 — Official Government / Curriculum
TIER 2 — Official Education Institution
TIER 3 — Licensed / Partner Content
TIER 4 — Public educational source with clear rights
TIER 5 — Community / Blog
```

Tier 5 hanya cocok untuk **discovery**, bukan canonical production source.

---

# 5. Source Governance

Setiap sumber harus memiliki status:

```text
discovered
verified
approved
active
paused
retired
```

Sebelum `active`, review:

- domain ownership;
- officiality;
- copyright / license;
- terms of use;
- robots policy;
- relevance;
- update frequency;
- document stability.

---

# 6. Python sebagai Content Intelligence Layer

## Rekomendasi

Gunakan Python untuk pipeline content, sedangkan Go tetap menjadi backend user-facing.

```text
Go
→ Authentication
→ Assessment Runtime
→ Progress
→ Subscription
→ Parent Dashboard
→ Admin API

Python
→ Crawl
→ Parse
→ Extract
→ Normalize
→ Map
→ Deduplicate
→ Validate
→ Candidate Generation
```

### Mengapa Python

Karena pipeline membutuhkan ecosystem kuat untuk:

- HTTP/document crawling;
- HTML parsing;
- PDF extraction;
- DOCX/XLSX parsing;
- NLP;
- embeddings / semantic similarity;
- classification;
- batch transformation;
- optional LLM-assisted processing.

### Baseline libraries

```text
httpx
BeautifulSoup / lxml
trafilatura
Playwright (only where required)
PyMuPDF
pdfplumber
python-docx
openpyxl
Pydantic
SQLAlchemy / SQLModel
```

Jangan memasukkan seluruh library ini sekaligus. Tambahkan berdasarkan format sumber yang benar-benar dipakai.

---

# 7. Crawler Architecture

```text
Python Content Pipeline
│
├── Scheduler
├── Source Registry
├── Fetcher
├── Parser
├── Extractor
├── Normalizer
├── Deduplicator
├── Curriculum Mapper
├── Candidate Generator
├── Validator
└── Publisher / Export Adapter
```

Crawler bukan backend utama aplikasi anak.

---

# 8. Fetching Strategy

## HTTP-first

Prioritas:

```text
HTTP GET
↓
HTML / PDF / DOCX
```

Gunakan browser automation hanya jika halaman memang membutuhkan JavaScript untuk mendapatkan konten yang diizinkan.

## Conditional Fetch

Simpan:

```text
ETag
Last-Modified
Content-Length
SHA-256
```

Flow:

```text
URL
 ↓
Check metadata
 ↓
Changed?
 ├── NO → skip
 └── YES → fetch
```

---

# 9. Crawl Run

Tabel:

```text
crawl_runs
```

Fields:

```text
id
source_id
started_at
finished_at
status
pages_discovered
pages_fetched
pages_changed
documents_created
errors
```

Status:

```text
queued
running
completed
partial
failed
cancelled
```

---

# 10. Crawl Item

```text
crawl_items
```

Fields:

```text
id
crawl_run_id
source_id
url
canonical_url
http_status
content_type
etag
last_modified
content_hash
language
discovered_at
fetched_at
status
error_code
```

Deduplication utama:

```text
canonical_url + content_hash
```

---

# 11. Raw Archive

Jangan hanya menyimpan hasil extraction. Simpan raw document untuk audit dan re-processing.

```text
source_documents
```

Fields:

```text
id
source_id
crawl_item_id
title
canonical_url
document_type
mime_type
content_hash
storage_key
published_at
retrieved_at
license
language
```

Object storage:

```text
S3-compatible storage
```

Contoh:

```text
raw/
  source-001/
    2026/
      09/
        document-hash.pdf
```

---

# 12. Extraction Pipeline

```text
Raw Document
    ↓
File Type Detector
    ↓
Parser
    ↓
Text / Table Extraction
    ↓
Heading Detection
    ↓
Metadata Extraction
    ↓
Normalized Document
```

Support awal:

```text
HTML
PDF
DOCX
TXT
Markdown
```

---

# 13. PDF Handling

Untuk PDF text-based:

```text
PyMuPDF / pdfplumber
```

Jika tidak ada text layer:

```text
PDF
 ↓
Detect scan
 ↓
OCR queue
```

OCR adalah fallback, bukan default.

Simpan:

```text
page_number
bbox (optional)
source_document_id
text
```

agar reviewer dapat kembali ke halaman asli.

---

# 14. Normalized Document

Contoh:

```json
{
  "document_id": "doc_001",
  "title": "Capaian Pembelajaran Matematika",
  "language": "id",
  "sections": [
    {
      "heading": "Bilangan",
      "page": 12,
      "text": "..."
    }
  ]
}
```

---

# 15. Semantic Chunking

Jangan selalu memotong berdasarkan jumlah karakter.

Urutan prioritas:

```text
Document
→ Heading
→ Section
→ Paragraph
→ Semantic chunk fallback
```

Setiap chunk:

```text
chunk_id
document_id
heading
page_start
page_end
sequence
text
token_count
hash
```

---

# 16. Curriculum Mapping

Ini adalah bagian inti.

Input:

```text
Normalized Content Chunk
```

Output:

```text
Education Level
Phase
Grade
Subject
CP
Topic
Skill
Learning Objective
```

Contoh:

```text
"Peserta didik membandingkan bilangan sampai 100"
                ↓
Fase A
                ↓
Grade 1/2 candidate
                ↓
Matematika
                ↓
Bilangan
                ↓
Membandingkan bilangan
```

---

# 17. Hybrid Mapping Engine

Gunakan tiga layer.

## Layer 1 — Deterministic

- source metadata;
- subject;
- phase;
- grade;
- keyword rules;
- known CP identifiers.

## Layer 2 — Semantic

- embedding similarity;
- taxonomy matching;
- semantic classifier.

## Layer 3 — Optional LLM Assistance

LLM hanya menghasilkan **candidate mapping**:

```json
{
  "candidate_skill": "number_comparison",
  "confidence": 0.91,
  "reason": "..."
}
```

Reviewer tetap menjadi final authority.

---

# 18. Confidence Score

```text
0.90–1.00 → High confidence
0.75–0.89 → Needs review
<0.75     → Manual review
```

Confidence tidak berarti auto-publish.

---

# 19. Curriculum Taxonomy

Gunakan:

```text
curriculum_frameworks
curriculum_versions
education_levels
phases
grades
subjects
curriculum_cp
curriculum_topics
curriculum_skills
```

Relasi:

```text
curriculum_version
        ↓
education_level
        ↓
phase
        ↓
grade
        ↓
subject
        ↓
CP
        ↓
topic
        ↓
skill
```

---

# 20. Source-to-Curriculum Provenance

Setiap production question wajib memiliki lineage:

```text
Question
  ↓
Skill
  ↓
CP
  ↓
Source Chunk
  ↓
Source Document
  ↓
Original URL
```

Ini dibutuhkan untuk audit, correction, legal review, dan curriculum revision.

---

# 21. Material Candidates

Data hasil mapping masuk ke:

```text
material_candidates
```

Status:

```text
draft
needs_review
approved
rejected
published
retired
```

Fields:

```text
id
skill_id
source_chunk_id
title
summary
learning_objective
difficulty_hint
confidence
created_by
reviewed_by
```

---

# 22. Question Candidate Pipeline

Jangan mengubah semua text menjadi soal secara otomatis.

```text
Source
 ↓
Curriculum Mapping
 ↓
Learning Objective
 ↓
Material Candidate
 ↓
Question Candidate
 ↓
Academic Review
 ↓
Publish
```

---

# 23. Question Generation Strategy

## Deterministic templates

Gunakan untuk:

- counting;
- matching;
- ordering;
- simple arithmetic;
- recognition;
- pattern.

## Assisted generation

Untuk:

- scenario;
- contextual problem;
- reading comprehension;
- explanation candidates.

AI boleh membantu menghasilkan candidate, tetapi production question tetap membutuhkan human QA.

---

# 24. AI Content Policy

AI bukan source of truth.

```text
Curriculum / licensed source
        ↓
Human-verified reference
        ↓
AI candidate generation
        ↓
Academic review
        ↓
Production
```

Bukan:

```text
AI → Production
```

---

# 25. Copyright & Licensing

Crawler tidak berarti seluruh internet bebas digunakan.

Jangan melakukan:

```text
crawl
→ copy full copyrighted material
→ transform
→ sell
```

tanpa hak penggunaan yang sesuai.

Untuk source pihak ketiga:

```text
License verified?
├── YES → allowed workflow
└── NO  → discovery / metadata only
```

Untuk dokumen pemerintah, tetap simpan source, identitas dokumen, URL, tanggal retrieval, dan catatan penggunaan; commercial use spesifik tetap perlu legal review.

---

# 26. Source Policy Matrix

| Source | Crawl | Store Raw | Transform | Production |
|---|---|---|---|---|
| Official curriculum | Yes, controlled | Yes | Yes | Human review |
| Licensed publisher | Yes, within license | Yes | Within license | Yes |
| Public educational site | Depends | Depends | Depends | License review |
| Blog with unclear rights | Discovery only | Minimal metadata | No | No |
| Copyrighted book without license | No | No | No | No |

---

# 27. Curriculum Versioning

Jika dokumen kurikulum berubah:

```text
New document
      ↓
New revision
      ↓
Compare CP statements
      ↓
Identify affected skills
      ↓
Identify affected questions
      ↓
Create review tasks
      ↓
Publish revised content
```

**Jangan overwrite historical curriculum.**

---

# 28. Update Detection

Gunakan:

```text
ETag
Last-Modified
content hash
semantic diff
```

Jika content identik:

```text
skip reprocessing
```

Jika berubah:

```text
new source document version
```

---

# 29. Deduplication

Gunakan kombinasi:

```text
URL canonicalization
+
SHA-256
+
text fingerprint
+
semantic similarity
```

Output:

```text
exact duplicate
near duplicate
related
unique
```

---

# 30. Child Readability Layer

Knowledge extraction tidak cukup. Content harus disesuaikan dengan usia.

Metadata:

```text
age_band
reading_level
word_count
sentence_length
complexity
visual_required
audio_required
```

Contoh:

```text
TK
→ visual-first
→ very short instruction
→ audio-friendly

SD 1–2
→ short text

SD 3–4
→ moderate reading

SD 5–6
→ more reasoning and contextual text
```

---

# 31. Database Architecture

Gunakan schema PostgreSQL terpisah.

```text
app
content
content_staging
audit
```

### app

```text
users
children
subscriptions
assessments
question_attempts
progress
```

### content

```text
curriculum
subjects
skills
questions
question_versions
assets
```

### content_staging

```text
sources
crawl_runs
crawl_items
source_documents
document_chunks
mapping_candidates
material_candidates
question_candidates
```

### audit

```text
review_logs
publish_logs
source_changes
```

---

# 32. Python ↔ Go Boundary

Jangan membuat dependency langsung yang rapuh.

Use:

```text
PostgreSQL schemas
+
JSON schemas
+
versioned import / publish contract
```

Flow:

```text
Python
  ↓
content_staging
  ↓
Reviewer
  ↓
Publish
  ↓
content
  ↓
Go API
```

Go hanya membaca canonical published content.

---

# 33. Python Service Layout

```text
services/
└── content-pipeline/
    ├── pyproject.toml
    ├── src/
    │   └── content_pipeline/
    │       ├── cli.py
    │       ├── config.py
    │       ├── crawler/
    │       ├── fetcher/
    │       ├── parsers/
    │       ├── extractors/
    │       ├── normalize/
    │       ├── curriculum/
    │       ├── mapping/
    │       ├── candidates/
    │       ├── validators/
    │       └── storage/
    └── tests/
```

CLI awal:

```bash
python -m content_pipeline crawl
python -m content_pipeline extract
python -m content_pipeline map
python -m content_pipeline validate
```

---

# 34. Scheduler

MVP dapat memakai cron / scheduled worker.

Contoh:

```text
Official curriculum → weekly
Licensed source     → daily / per agreement
Static guide        → monthly
```

Crawler tidak boleh melakukan polling terlalu agresif.

---

# 35. Queue Strategy

MVP:

```text
PostgreSQL job table
```

gunakan locking seperti:

```sql
SELECT ...
FOR UPDATE SKIP LOCKED;
```

Ketika volume meningkat:

```text
Redis / message queue
```

Kafka tidak diperlukan pada MVP kecuali ada kebutuhan nyata.

---

# 36. Admin Content Review

UI:

```text
Content Review
├── Sources
├── Documents
├── Curriculum Mapping
├── Materials
├── Question Candidates
├── Duplicates
└── Publish Queue
```

Actions:

```text
Approve
Reject
Edit
Remap
Request Revision
```

---

# 37. Reviewer Experience

Saat reviewer membuka candidate:

```text
LEFT
Source Document
page + excerpt

CENTER
Question / Material candidate

RIGHT
Curriculum mapping

Phase
Grade
Subject
CP
Skill
Difficulty
Confidence
```

Reviewer harus dapat melihat **sumber dan hasil transformasi dalam satu layar**.

---

# 38. Provenance UI

Internal metadata minimum:

```text
Source
Document
Page
CP
Skill
Reviewer
Reviewed at
Version
```

Tidak semuanya harus ditampilkan kepada anak/parent.

---

# 39. Creator / Revenue Share Readiness

Karena roadmap bisnis mencakup revenue sharing untuk pembuat bank soal, sejak awal simpan:

```text
creator_id
question_id
published_at
usage_count
completion_count
quality_score
```

Untuk adaptive content:

```text
source_document_id
adaptation_type
adapted_by
reviewed_by
```

Dengan ini revenue share dapat dikembangkan tanpa migrasi besar.

---

# 40. Automatic Quality Checks

Automatic:

```text
Missing source
Missing CP mapping
Missing skill
Duplicate
Empty answer
Invalid options
Answer mismatch
Text too long
Broken media
```

Academic review:

```text
Correctness
Curriculum alignment
Difficulty
Child readability
Explanation quality
Ambiguity
```

---

# 41. Content Lifecycle

```text
DISCOVERED
   ↓
FETCHED
   ↓
EXTRACTED
   ↓
NORMALIZED
   ↓
MAPPED
   ↓
CANDIDATE
   ↓
REVIEW
   ↓
APPROVED
   ↓
PUBLISHED
   ↓
REVISED
   ↓
RETIRED
```

Tidak ada shortcut:

```text
FETCHED → PUBLISHED
```

---

# 42. Security

Crawler wajib memiliki:

- source allowlist;
- max response size;
- timeout;
- redirect limit;
- MIME validation;
- rate limiting;
- safe temporary directory;
- outbound restrictions;
- file scanning jika dibutuhkan.

## SSRF protection

Reject:

```text
localhost
127.0.0.1
private IP ranges
link-local
cloud metadata endpoints
internal-only DNS targets
```

Gunakan domain allowlist dan validasi resolved IP.

---

# 43. Robots / Terms / Access Compliance

Sebelum source diaktifkan:

```text
robots.txt
Terms of Use
License
```

harus diperiksa.

Jika automated access dilarang:

```text
source.status = paused
```

Jangan bypass dengan browser automation.

---

# 44. Curriculum Change Watcher

Ini salah satu fitur yang lebih bernilai daripada crawling tanpa tujuan.

```text
Curriculum Watcher
        ↓
Official source changed
        ↓
Document diff
        ↓
Changed CPs
        ↓
Affected skills
        ↓
Affected questions
        ↓
Review queue
```

Tujuannya memastikan content engine tetap aligned terhadap perubahan resmi.

---

# 45. Recommended Initial Official Source Allowlist

Mulai kecil dengan sumber resmi yang relevan:

```text
kurikulum.kemendikdasmen.go.id
guru.kemendikdasmen.go.id
kemendikdasmen.go.id
repositori.kemendikdasmen.go.id
peraturan.bpk.go.id
```

Portal guru resmi menyediakan akses ke informasi Kurikulum Merdeka, CP/ATP, dan kumpulan panduan; ini dapat menjadi salah satu source discovery resmi. citeturn353788search3turn353788search7

---

# 46. Current Regulatory References for the Pipeline

### CP 046/H/KR/2025

https://kurikulum.kemendikdasmen.go.id/file/1718851677_manage_file.pdf

Dokumen ini menetapkan CP untuk PAUD, SD/MI, SMP/MTs, SMA/MA, SMK/MAK, program kesetaraan, dan pendidikan khusus. citeturn353788search24

### 2026 CP Revision

https://www.kemendikdasmen.go.id/siaran-pers/15636-capaian-pembelajaran-baru-telah-terbit-yang-berubah-hanya-mata-pelajaran-agama-dan-budi-pekerti

Pengumuman resmi Juni 2026 mengenai Keputusan Kepala BKPDM Nomor 020 Tahun 2026. citeturn353788search0

### Permendikdasmen No. 12 Tahun 2025 — Standard Isi

https://peraturan.bpk.go.id/Details/322494/

Peraturan ini menjadi rujukan standar isi untuk PAUD, pendidikan dasar, dan pendidikan menengah dan berlaku sejak 15 Juli 2025. citeturn211711search0

### Permendikdasmen No. 13 Tahun 2025 — Perubahan Kurikulum

https://peraturan.bpk.go.id/Details/322506

Peraturan ini mengubah Permendikbudristek No. 12 Tahun 2024 mengenai kurikulum dan berlaku sejak 15 Juli 2025. citeturn211711search2

---

# 47. MVP Implementation Phases

## Phase 1 — Curriculum Ingestion

```text
Source Registry
→ Official Crawler
→ Raw Archive
→ PDF/HTML extraction
→ Curriculum taxonomy
```

Target:

```text
PAUD / TK Fase Fondasi
SD Fase A
SD Fase B
SD Fase C
```

## Phase 2 — Content Candidates

```text
Chunking
→ Curriculum mapping
→ Learning objective
→ Material candidate
→ Question candidate
```

## Phase 3 — Reviewer CMS

```text
Review
→ Edit
→ Remap
→ Approve
→ Publish
```

## Phase 4 — Intelligence

```text
Embeddings
→ Semantic matching
→ Duplicate detection
→ LLM-assisted mapping
```

## Phase 5 — Creator Marketplace

```text
Creator
→ Submit bank soal
→ Review
→ Publish
→ Usage tracking
→ Revenue attribution
```

---

# 48. What the MVP Must NOT Do

```text
❌ Crawl the entire internet
❌ Scrape search engines endlessly
❌ Bypass robots.txt
❌ Bypass Terms of Use
❌ Download everything
❌ Publish AI-generated questions automatically
❌ Copy copyrighted books/articles without rights
❌ Overwrite old curriculum versions
❌ Mix staging and published content
```

---

# 49. Recommended Repository Structure

```text
nocify-edu/
│
├── apps/
│   ├── api/                   # Go
│   └── web/                   # SvelteKit
│
├── services/
│   └── content-pipeline/      # Python
│       ├── pyproject.toml
│       ├── src/
│       │   └── content_pipeline/
│       │       ├── crawler/
│       │       ├── fetcher/
│       │       ├── parsers/
│       │       ├── extractors/
│       │       ├── normalize/
│       │       ├── curriculum/
│       │       ├── mapping/
│       │       ├── candidates/
│       │       ├── validators/
│       │       └── storage/
│       └── tests/
│
├── content/
│   ├── curriculum/
│   ├── taxonomy/
│   └── seeds/
│
├── infra/
└── docs/
```

---

# 50. End-to-End Example

```text
Official curriculum document changes
            ↓
Crawler detects new file
            ↓
Download + hash
            ↓
Raw archive
            ↓
Extract pages
            ↓
Create semantic chunks
            ↓
Identify CP
            ↓
Map phase / grade / subject
            ↓
Map topic / skill
            ↓
Create material candidate
            ↓
Generate question candidates
            ↓
Automatic QA
            ↓
Academic reviewer
            ↓
Publish
            ↓
Go API
            ↓
Child challenge
            ↓
Question attempt
            ↓
Skill mastery
            ↓
Parent progress
```

---

# 51. Acceptance Criteria

## Crawler

- [ ] Source registry works.
- [ ] Allowlist works.
- [ ] Crawl run is recorded.
- [ ] Conditional fetch works.
- [ ] Duplicate documents are detected.
- [ ] Raw source can be reprocessed.
- [ ] Failed pages retry safely.
- [ ] Crawl can resume.

## Curriculum

- [ ] Curriculum version is stored.
- [ ] PAUD Fase Fondasi supported.
- [ ] SD Fase A/B/C supported.
- [ ] Grade mapping supported.
- [ ] Subject mapping supported.
- [ ] CP provenance is stored.
- [ ] Revisions do not overwrite history.

## Content

- [ ] Every candidate has source lineage.
- [ ] Every question has skill mapping.
- [ ] Duplicate detection works.
- [ ] Human review is required before publish.

## Technical

- [ ] Python pipeline isolated from Go API.
- [ ] PostgreSQL staging/canonical schemas separated.
- [ ] Dockerized.
- [ ] Metrics and logs available.
- [ ] Secrets are not committed.

---

# 52. Final Architecture

```text
                         NOCIFY EDU
                             │
                ┌────────────┴─────────────┐
                │                          │
             GO APP                  PYTHON CONTENT
                │                          │
        User-facing runtime          Intelligence layer
                │                          │
          PostgreSQL <──────────── PostgreSQL
                │                   staging/canonical
                │                          │
                │                     S3 Raw Archive
                │
        Published Content
                │
        Assessment / Progress
                │
              Parent
```

### Core principle

**Go menjalankan produk.**  
**Python menjaga knowledge/content pipeline.**  
**PostgreSQL menjadi system of record.**  
**Object Storage menyimpan source archive.**  
**Human reviewer menjadi final academic gate.**

---

# 53. Final Recommendation

Jangan membangun fitur ini sebagai **"crawler pembuat soal"**.

Bangun sebagai:

> **Content Intelligence Engine yang menjaga knowledge base pendidikan tetap terstruktur, versioned, terhubung ke Kurikulum Merdeka, memiliki provenance, dan siap diubah menjadi assessment berkualitas.**

Dengan model tersebut, ekspansi:

```text
TK
 ↓
SD
 ↓
SMP
 ↓
SMA
```

tidak memerlukan rewrite besar.

Dan ketika kurikulum berubah:

```text
New Curriculum Revision
        ↓
Impact Analysis
        ↓
Affected Skills
        ↓
Affected Questions
        ↓
Review
        ↓
Publish
```

---

# 54. Next Milestone

Urutan implementasi yang disarankan:

```text
M1 Source Registry + Official Crawler
M2 Extraction + Raw Archive
M3 Curriculum Taxonomy + CP Mapping
M4 Material / Question Candidates
M5 Reviewer CMS + Publish Workflow
M6 Semantic Mapping + Deduplication
M7 Curriculum Change Detection
M8 Creator / Revenue Attribution
```

**Jangan melompat ke M6–M8 sebelum M1–M5 stabil.**
