# EduPlay — MVP Product, Business & Technical Specification

**Document type:** Product / Business / Technical Specification  
**Status:** MVP Source of Truth  
**Target:** Anak TK–SD, dengan fondasi extensible ke SMP–SMA  
**Primary customer:** Orang tua / wali  
**Primary user:** Anak  
**Business model:** Freemium + Monthly Subscription  
**Primary platform:** Responsive Web App, mobile-first  
**Date:** 2026-09-25

---

## 1. Executive Summary

EduPlay adalah platform pembelajaran berbasis **interactive assessment** untuk anak TK dan SD. Produk tidak diposisikan sebagai "bank soal digital", melainkan sebagai pengalaman belajar singkat yang terasa seperti bermain, dengan progress tracking yang mudah dipahami orang tua.

Prinsip produk:

> **Anak bermain dan belajar. Orang tua membayar dan memahami perkembangan anak.**

MVP berfokus pada:

- Tes / challenge mata pelajaran sekolah.
- Pengalaman visual dan interaktif untuk TK dan SD.
- Progress tracking berbasis skill, bukan hanya nilai.
- Parent account sebagai pemilik subscription.
- Child profile sebagai profil belajar.
- Monthly subscription.
- Admin/content CMS untuk mengelola soal.
- Arsitektur backend Go + PostgreSQL.
- Frontend ringan, mobile-first, dan responsive.
- Deployment menggunakan Docker.
- Brand asset harus curated, legally usable, konsisten, dan tidak boleh menghasilkan tampilan "AI slop".

---

# 2. Product Vision

## 2.1 Visi

Membangun platform latihan akademik yang membuat anak **ingin kembali belajar**, bukan merasa sedang mengerjakan ujian.

## 2.2 Misi

1. Membuat latihan mata pelajaran sekolah terasa lebih menyenangkan.
2. Memberikan pengalaman yang sesuai dengan usia anak.
3. Membantu orang tua memahami kemampuan anak berdasarkan skill.
4. Membuat konten assessment yang terstruktur dan dapat berkembang lintas jenjang.
5. Membangun fondasi teknis yang dapat berkembang dari TK–SD ke SMP–SMA.

---

# 3. Product Positioning

### Jangan memposisikan produk sebagai:

> "Website kumpulan soal anak."

### Positioning yang dituju:

> **Playful assessment platform untuk membantu anak berlatih dan membantu orang tua memahami progress belajar.**

Pengalaman dibagi menjadi dua:

| Persona | Kebutuhan | UX utama |
|---|---|---|
| Anak | Fun, mudah, visual, tidak membosankan | Challenge, visual, audio, reward |
| Orang tua | Progress, transparansi, value subscription | Dashboard, skill mastery, history, recommendations |

---

# 4. Business Model

## 4.1 Customer

Customer utama adalah **orang tua / wali**.

Anak adalah end-user, tetapi tidak menjadi pemilik subscription.

Struktur account:

```text
Parent Account
├── Child A
├── Child B
└── Child C
```

## 4.2 Subscription

MVP menggunakan **Monthly Subscription**.

Jangan memulai dengan banyak tier.

Rekomendasi awal:

```text
FREE
- 1 child
- limited daily challenge
- basic result
- basic progress

MONTHLY
- unlimited challenge
- all available subjects
- detailed progress
- skill mastery
- historical progress
- recommendations
- multiple child profiles
- parent reports
```

Harga harus divalidasi melalui eksperimen bisnis; angka pricing bukan bagian dari technical contract.

## 4.3 Conversion Funnel

```text
Landing Page
    ↓
Register Parent
    ↓
Create Child
    ↓
Try First Challenge
    ↓
See Result
    ↓
See Progress Value
    ↓
Subscription Offer
    ↓
Monthly Subscriber
```

Critical conversion moment:

> Orang tua pertama kali melihat bahwa platform dapat menjelaskan **"anak saya sudah bisa apa dan apa yang perlu dilatih"**.

---

# 5. Product Principles

## 5.1 Child-first UX

Semua UI yang digunakan anak harus mempertimbangkan:

- kemampuan membaca sesuai usia;
- kemampuan motorik;
- kemampuan memahami instruksi;
- rentang perhatian;
- penggunaan touchscreen;
- visual hierarchy;
- kebutuhan audio;
- ukuran target sentuh.

## 5.2 Parent-first business model

Parent dashboard harus menjawab:

- Anak saya berkembang atau tidak?
- Skill apa yang sudah dikuasai?
- Skill apa yang masih lemah?
- Apa yang harus dilakukan berikutnya?
- Apakah subscription ini memberikan value?

## 5.3 Learning before gamification

Game mechanic hanya menjadi wrapper.

Jangan mengorbankan:

- clarity;
- learning objective;
- assessment quality;
- accessibility;
- accuracy;

demi animation atau reward.

## 5.4 Progressive complexity

TK, SD kelas rendah, dan SD kelas tinggi tidak boleh menggunakan UX yang sama persis.

---

# 6. MVP Scope

## 6.1 In Scope

### Public / Marketing

- Landing page.
- Product explanation.
- Pricing.
- Subjects.
- How it works.
- FAQ.
- Login.
- Register.
- CTA.
- Parent-oriented conversion sections.

### Parent App

- Login.
- Register.
- Parent profile.
- Child management.
- Subscription.
- Dashboard.
- Progress.
- Assessment history.
- Recommendation.
- Account settings.

### Child App

- Child profile selection.
- Subject selection.
- Challenge.
- Question.
- Audio instruction.
- Feedback.
- Score/result.
- XP / stars / badge.
- Basic progression.

### Content

- Grade.
- Subject.
- Chapter.
- Skill.
- Difficulty.
- Question type.
- Question media.
- Answer.
- Explanation.
- Status workflow.

### Admin

- Question CRUD.
- Question preview.
- Publish / unpublish.
- Subject management.
- Curriculum management.
- Asset management.
- Question review.

---

# 7. Non-Goals MVP

Jangan membangun fitur berikut pada MVP kecuali ada kebutuhan bisnis yang tervalidasi:

- Live class.
- Teacher marketplace.
- Social feed.
- Global leaderboard.
- Multiplayer.
- AI tutor.
- AI-generated questions tanpa human review.
- Native Android/iOS application.
- Full video course.
- Community chat.
- Advanced ML recommendation.
- MMORPG/game world penuh.

---

# 8. Assessment Model

## 8.1 Konsep

Assessment tidak selalu ditampilkan sebagai "TEST".

Child-facing terminology:

- Challenge
- Mission
- Practice
- Quest
- Level
- Star
- Reward

Parent-facing terminology:

- Assessment
- Progress
- Skill
- Mastery
- Accuracy
- Recommendation

## 8.2 Session Structure

```text
Start
  ↓
Warm Up
  ↓
Challenge 1
  ↓
Challenge 2
  ↓
Challenge 3
  ↓
Mini Reward
  ↓
Challenge 4
  ↓
Challenge 5
  ↓
Result
  ↓
Progress Updated
```

Target baseline:

| Level | Soal/session |
|---|---:|
| TK | 5–7 |
| SD 1–3 | 7–10 |
| SD 4–6 | 10–15 |

Angka ini merupakan baseline UX untuk eksperimen dan harus divalidasi melalui data penggunaan.

---

# 9. Question Types

## 9.1 TK

Prioritaskan interaksi yang tidak membutuhkan banyak typing:

- Image choice.
- Tap picture.
- Counting.
- Matching.
- Drag and drop.
- Sort.
- Sequence.
- Shape recognition.
- Letter recognition.
- Audio choice.

Contoh:

```text
Mana yang jumlah apelnya lebih banyak?

🍎🍎🍎

atau

🍎🍎🍎🍎🍎
```

## 9.2 SD

Tambahkan:

- Multiple choice.
- Number input.
- Word problem.
- Matching.
- Ordering.
- Sequence.
- Diagram interpretation.
- Image analysis.
- Short answer.
- Audio question.

Question renderer harus component-based sehingga tipe soal baru dapat ditambahkan tanpa merombak assessment engine.

---

# 10. Feedback Logic

## Correct

Jangan hanya:

> Correct!

Gunakan:

> Hebat! Jawabanmu benar.

Tambahkan micro-animation atau sound yang singkat.

## Incorrect

Jangan:

> Salah!

Gunakan:

> Hampir benar. Coba lihat lagi petunjuknya.

Kemudian:

```text
Wrong
  ↓
Hint
  ↓
Retry / Continue
```

Feedback harus membantu anak belajar, bukan mempermalukan.

---

# 11. Difficulty & Mastery Logic

MVP tidak memerlukan ML.

Gunakan deterministic rules.

Contoh:

```text
Mastery < 40%
→ easy / reinforcement

40–70%
→ normal

70–85%
→ medium-hard

>85%
→ challenge / next skill
```

Contoh adaptive rule:

```text
3 correct
→ increase difficulty

2 wrong
→ lower difficulty + hint

Repeated wrong
→ remediation recommendation
```

Mastery harus dihitung pada level:

```text
Child
 → Subject
   → Chapter
     → Skill
```

Bukan hanya:

```text
Child → Math = 80
```

---

# 12. Reward System

MVP reward:

- XP.
- Stars.
- Badge.
- Sticker.
- Avatar unlock.
- Small scene unlock.

Tidak diwajibkan:

- Global ranking.
- Competitive leaderboard.
- Reward gambling.
- Lootbox.
- Infinite scrolling.
- Dark-pattern engagement.

Reward harus memperkuat learning behavior, bukan membuat anak mengejar engagement tanpa tujuan pendidikan.

---

# 13. Parent Dashboard

Dashboard wajib menampilkan:

```text
Child: A

Overall Progress
████████░░ 82%

Mathematics      88%
Bahasa Indonesia 74%
IPAS             81%
```

Kemudian:

### Strong Skills

- Penjumlahan.
- Nilai tempat.

### Developing

- Perkalian.
- Pecahan.

### Needs Practice

- Soal cerita.
- Pembagian.

### Recommendation

> Latih 5 challenge tentang pembagian minggu ini.

Parent harus bisa melihat:

- last activity;
- completion;
- accuracy;
- skill mastery;
- progress trend;
- assessment history;
- recommended next activity.

---

# 14. Curriculum & Content Model

Jangan hardcode grade dan subject di backend.

Gunakan hierarchy:

```text
Curriculum
  └── Education Level
       └── Grade
            └── Subject
                 └── Chapter
                      └── Skill
                           └── Question
```

Contoh:

```text
Kurikulum X
└── SD
    └── Grade 2
        └── Mathematics
            └── Addition
                └── Addition 1–20
                    ├── Q001
                    ├── Q002
                    └── Q003
```

Dengan struktur ini sistem dapat berkembang ke:

```text
TK
SD
SMP
SMA
```

tanpa mengubah architecture utama.

---

# 15. Question Schema

Contoh:

```json
{
  "id": "question_001",
  "type": "image_choice",
  "grade": 1,
  "subject": "math",
  "skill": "addition_1_20",
  "difficulty": 2,
  "content": {
    "question": "Berapa jumlah apel?",
    "image": "/assets/questions/apple-count-01.svg",
    "options": [
      { "id": "a", "label": "5" },
      { "id": "b", "label": "6" },
      { "id": "c", "label": "7" }
    ]
  },
  "answer": "b",
  "explanation": "Ada 6 apel."
}
```

Question type diperlakukan sebagai renderer.

Contoh renderer:

```text
QuestionRenderer
├── single_choice
├── image_choice
├── drag_drop
├── matching
├── sequence
├── tap_count
├── number_input
├── audio_choice
└── short_answer
```

---

# 16. Recommended Technical Stack

## Frontend

```text
SvelteKit
TypeScript
Tailwind CSS
shadcn-svelte / Bits UI
```

Svelte/SvelteKit digunakan untuk frontend karena cocok untuk aplikasi web interaktif dengan kebutuhan bundle yang tetap terkontrol. Tailwind memiliki model responsive yang mobile-first. [Svelte documentation](https://svelte.dev/docs) [Tailwind responsive design](https://tailwindcss.com/docs/responsive-design)

## Backend

```text
Go
chi
pgx
sqlc
OpenAPI
```

## Database

```text
PostgreSQL
```

## Object Storage

```text
S3-compatible storage
```

Untuk:

- illustration;
- audio;
- question image;
- downloadable report;
- optional animation.

## Infrastructure

```text
Docker
Docker Compose
Traefik
```

## CI/CD

```text
GitHub Actions
```

## Observability

```text
Prometheus
Grafana
Loki
OpenTelemetry
```

---

# 17. Architecture

```text
                         INTERNET
                            |
                         TRAEFIK
                            |
                +-----------+-----------+
                |                       |
             SVELTEKIT                 GO API
                |                       |
          +-----+-----+       +---------+----------+
          |           |       |         |          |
      Marketing     App    Auth     Assessment   Progress
                    |                 |            |
                 Child UI           Content       Rewards
                 Parent UI         Billing       Recommendation
                                      |
                              +-------+-------+
                              |               |
                         PostgreSQL       Object Storage
```

MVP menggunakan **modular monolith**, bukan microservices.

---

# 18. Go Backend Structure

Recommended:

```text
/internal
    /auth
    /users
    /children
    /subscriptions
    /curriculum
    /subjects
    /questions
    /assessment
    /progress
    /rewards
    /content
    /media
    /admin
```

Satu binary pada MVP.

Pisahkan service hanya ketika:

- scaling membutuhkan;
- deployment independent dibutuhkan;
- workload benar-benar berbeda;
- operational complexity justified.

---

# 19. Database Core Tables

Minimum:

```text
users
children
subscriptions

education_levels
grades
subjects
chapters
skills
curriculum_versions

questions
question_versions
question_options
question_media

assessments
assessment_items
assessment_attempts
question_attempts

skill_progress
daily_progress

rewards
badges
child_inventory

assets
asset_licenses

audit_logs
```

---

# 20. Account & Authorization Model

## Parent

```text
Email
Password / OTP
Payment
Subscription
Child Management
Progress
Settings
```

## Child

```text
Nickname
Avatar
PIN / child-safe access
Learning session
Progress
Reward
```

Jangan meminta data pribadi anak yang tidak dibutuhkan untuk learning experience.

Parent adalah boundary utama untuk child profile.

---

# 21. Subscription Business Logic

Status:

```text
trial
active
past_due
cancelled
expired
```

Flow:

```text
Parent Register
  ↓
Free Account
  ↓
Create Child
  ↓
Try Free Challenge
  ↓
Subscribe
  ↓
Payment Success
  ↓
Subscription Active
```

Subscription check harus dilakukan server-side.

Jangan mempercayai status subscription dari frontend.

Contoh:

```text
Frontend
    ↓
GET /api/me/subscription
    ↓
Backend
    ↓
PostgreSQL
```

---

# 22. Payment Integration

Payment gateway harus dipisahkan dari business domain subscription.

Pattern:

```text
Payment Provider
      ↓
Webhook
      ↓
Go Backend
      ↓
Subscription Service
      ↓
PostgreSQL
```

Webhook harus:

- authenticated/verified;
- idempotent;
- logged;
- retry-safe.

Jangan mengaktifkan subscription hanya berdasarkan response redirect dari browser.

---

# 23. Content CMS

Admin harus dapat membuat:

```text
Draft
  ↓
Review
  ↓
Approved
  ↓
Published
  ↓
Retired
```

Question wajib mempunyai:

- learning objective;
- subject;
- grade;
- skill;
- difficulty;
- question type;
- correct answer;
- explanation;
- source/revision metadata;
- content reviewer;
- status.

---

# 24. Content Quality Gate

Soal tidak boleh langsung publish.

Minimum review:

### Academic

- Correct answer verified.
- Explanation verified.
- Curriculum alignment verified.
- Difficulty reasonable.

### UX

- Child can understand instruction.
- Text readable.
- Touch target large enough.
- Visual does not create ambiguity.

### Accessibility

- Meaning is not conveyed by color alone.
- Audio/visual alternatives available when appropriate.
- Contrast is sufficient.
- Keyboard access is available in parent/admin area.

---

# 25. BRAND ASSET MANDATORY POLICY

## 25.1 Principle

**No random assets.**

Semua asset visual harus melalui brand asset governance.

Yang termasuk:

- logo;
- mascot;
- character;
- illustration;
- icon;
- sticker;
- reward graphics;
- background;
- decorative shape;
- animation;
- sound effect;
- educational diagram;
- photography.

## 25.2 Asset harus memiliki provenance

Setiap asset harus tercatat:

```text
Asset ID
Asset Name
Source
Creator
License
License URL
Download Date
Original File
Modified?
Modification Notes
Where Used
```

Contoh:

```json
{
  "asset_id": "ILL-0001",
  "name": "primary-school-student-01.svg",
  "source": "DrawKit",
  "creator": "Designstripe / DrawKit",
  "license": "DrawKit License",
  "license_url": "https://www.drawkit.com/license",
  "downloaded_at": "2026-09-25",
  "modified": true,
  "usage": [
    "landing-page",
    "onboarding"
  ]
}
```

## 25.3 Approved Asset Sources

### Tier A — Primary Candidate

**DrawKit**

Sangat cocok sebagai starting point karena memiliki Education, School dan Primary School collections, termasuk SVG/vector yang dapat disesuaikan. DrawKit menyatakan asset dapat digunakan dalam commercial projects sesuai lisensinya, tetapi asset tidak boleh didistribusikan ulang sebagai standalone asset. Lisensi harus disimpan bersama project.  
Source: https://www.drawkit.com/license  
Education: https://www.drawkit.com/illustrations/education-e-learning-illustrations  
Primary school: https://www.drawkit.com/illustrations/primary-school-education-illustrations

### Tier A — Supporting Marketing

**unDraw**

Cocok untuk supporting visual, marketing page, technical/product illustrations, bukan sebagai identitas utama child experience. unDraw menyediakan SVG yang dapat dipakai pada commercial/personal project tanpa attribution menurut halaman resminya.  
Source: https://undraw.co/

### Tier A — Alternative Illustration Library

**Storyset**

Cocok untuk campaign/marketing visuals dan scenes karena dapat dikustomisasi dan dianimasikan. Gunakan hanya setelah license/attribution requirement dari asset yang dipilih diverifikasi.  
Source: https://storyset.com/

### Tier B — Motion

**LottieFiles**

Dapat digunakan untuk micro-interactions atau celebration animation setelah license masing-masing file diverifikasi. Lottie Simple License mengizinkan penggunaan komersial sesuai syarat lisensinya; attribution dapat tidak diwajibkan, tetapi creator attribution sangat dianjurkan.  
Source: https://lottiefiles.com/page/license

## 25.4 Asset Selection Rule

Asset harus lolos:

```text
Visual Fit
+
Brand Fit
+
Age Fit
+
Consistency
+
License Verification
+
Performance
```

Kalau satu gagal, asset tidak boleh masuk production.

---

# 26. BRAND ASSET ART DIRECTION

## 26.1 Required Visual Language

Brand harus menggunakan satu visual language.

Contoh direction:

```text
2D editorial / playful illustration
+
soft geometric forms
+
warm friendly character
+
limited color palette
+
hand-crafted visual details
+
clear educational context
```

## 26.2 Character System

Jangan membuat karakter random per halaman.

Bangun:

```text
Mascot
├── Neutral
├── Happy
├── Thinking
├── Celebration
├── Encouragement
└── Explanation
```

Karakter harus memiliki:

- consistent face;
- consistent proportions;
- consistent clothing system;
- consistent line/shape language;
- consistent color palette.

## 26.3 Subject Illustration System

Setiap subject boleh mempunyai visual motif:

```text
Math
→ number / shape / blocks

Bahasa Indonesia
→ book / story / words

IPAS
→ nature / experiment / observation

English
→ vocabulary / objects / conversation
```

Tetapi style illustration harus tetap satu keluarga.

---

# 27. ANTI AI SLOP POLICY

## 27.1 Absolute Rule

**Do not use AI-generated imagery as production brand assets.**

Termasuk:

- AI-generated hero image;
- AI-generated characters;
- AI-generated mascot;
- AI-generated background;
- AI-generated educational diagrams;
- AI-generated stock-like illustrations.

AI boleh digunakan untuk:

- brainstorming;
- copy ideation;
- technical assistance;
- internal concept exploration;

tetapi bukan sebagai final visual brand asset tanpa explicit creative review dan approval.

## 27.2 AI Slop Indicators

Production UI harus ditolak jika memiliki kombinasi:

- terlalu banyak gradient;
- random 3D objects;
- glossy floating shapes;
- generic smiling children;
- excessive neon colors;
- inconsistent anatomy/proportions;
- random decorative blobs;
- inconsistent illustration styles;
- visual elements yang tidak punya fungsi;
- fake depth/shadow berlebihan;
- excessive glassmorphism;
- terlalu banyak rounded cards;
- "startup template" appearance;
- setiap section memakai style berbeda;
- hero image terasa seperti generated stock image.

## 27.3 Anti-Slop Rule

> **Setiap visual harus memiliki alasan desain.**

Tanyakan:

1. Apa fungsi visual ini?
2. Siapa yang melihat?
3. Apakah sesuai usia?
4. Apakah konsisten dengan brand?
5. Apakah dibutuhkan?
6. Dari mana asset ini berasal?
7. Apakah lisensinya jelas?

Jika jawabannya tidak jelas → jangan gunakan.

---

# 28. CUSTOMIZATION RULE

Jangan:

```text
Download
→ Copy
→ Paste
→ Production
```

Lebih baik:

```text
Source Asset
    ↓
Brand Color Mapping
    ↓
Composition Adjustment
    ↓
Crop / Layout
    ↓
Optional Custom Illustration
    ↓
Production
```

Tujuannya agar produk tidak terlihat seperti template asset marketplace.

---

# 29. Typography

Recommended direction:

```text
Display:
Fredoka / similar friendly display family

Body/UI:
Nunito / similar highly readable sans-serif
```

Aturan:

- maksimum 2 font families;
- body text harus sangat readable;
- heading dapat playful;
- jangan memakai display font untuk paragraph;
- angka/score harus mudah dibaca.

Font license wajib diverifikasi sebelum production.

---

# 30. Color System

Gunakan:

```text
Primary
Secondary
Accent
Success
Warning
Error
Neutral
Background
```

Contoh awal:

```text
Primary:   #5B5CE2
Yellow:    #FFC857
Green:     #54C98A
Orange:    #FF8A4C
Background #FFFDF8
```

Warna ini hanya sebagai starting tokens, bukan final brand identity.

Final palette harus berasal dari brand design review.

---

# 31. Mobile-First Mandatory Policy

**Mobile support bukan feature tambahan.**

Mobile adalah baseline.

Desain harus dimulai dari:

```text
320px+
```

dan diuji sampai desktop.

## 31.1 Required Viewports

Minimum testing:

```text
320 × 568
360 × 800
390 × 844
412 × 915
768 × 1024
1024 × 768
1280 × 720
1440 × 900
```

## 31.2 Touch Target

Child UI harus menggunakan touch target besar.

Target baseline:

```text
minimum 44×44px
preferred 48×48px+
```

untuk interactive controls.

## 31.3 No Horizontal Scroll

Tidak boleh ada accidental horizontal overflow pada:

- landing;
- login;
- child app;
- parent dashboard;
- question page;
- result;
- admin.

## 31.4 Responsive Question UI

Question layout harus dapat berpindah:

```text
Mobile:
Question
Image
Options
CTA

Desktop:
Question + visual
Options / interaction
```

Jangan hanya mengecilkan desktop layout.

---

# 32. Mobile Navigation

### Child

Gunakan navigation minimal.

Contoh:

```text
Home
Challenges
Rewards
Profile
```

Jangan menampilkan admin-like navigation.

### Parent

Boleh lebih information-dense:

```text
Overview
Children
Progress
History
Subscription
Settings
```

Desktop dapat menggunakan sidebar.

Mobile dapat menggunakan bottom navigation / drawer tergantung information architecture.

---

# 33. Safe Area & Mobile Browser

Wajib memperhatikan:

- browser address bar;
- mobile viewport height;
- `dvh/svh`;
- safe-area insets;
- keyboard opening;
- input focus;
- sticky CTA.

Jangan menggunakan `100vh` secara naïf untuk full-screen mobile app.

Gunakan dynamic viewport units sesuai kebutuhan platform.

---

# 34. Responsive Design Rules

Gunakan mobile-first CSS.

Contoh:

```html
<div class="
  grid-cols-1
  md:grid-cols-2
  lg:grid-cols-4
">
```

Base styles harus mengarah ke mobile, lalu enhancement ke breakpoint yang lebih besar. Tailwind mendukung pendekatan responsive mobile-first ini secara native.  
Source: https://tailwindcss.com/docs/responsive-design

---

# 35. Performance Budget

Target awal:

```text
LCP        < 2.5s
INP        < 200ms
CLS        < 0.1
```

Target perlu diukur pada real deployment, bukan hanya local development.

## Asset performance

- SVG untuk simple illustration.
- WebP/AVIF untuk raster image.
- Lazy-load non-critical image.
- Jangan memuat semua animation di initial page.
- Compress audio.
- Preload hanya critical assets.
- Hindari giant hero image.

## JS

- Jangan import library besar untuk fungsi sederhana.
- Hindari global state berlebihan.
- Split heavy interactive components.
- Jangan menjalankan animation library pada seluruh page.

---

# 36. Brand Asset Performance Rules

Asset visual harus memiliki budget.

Contoh:

```text
Hero illustration:
≤ 200–300 KB optimized

Icon:
SVG preferred

Background:
compressed

Animation:
lazy / on demand
```

Angka di atas adalah working budget, bukan hard law; actual budget harus disesuaikan dengan visual quality dan real-device performance.

---

# 37. Accessibility

Minimum:

- semantic HTML;
- keyboard navigation pada parent/admin;
- focus state;
- sufficient contrast;
- alt text;
- error state yang jelas;
- tidak menggunakan warna sebagai satu-satunya signal;
- reduced-motion support;
- audio control;
- readable typography.

Child interface harus tetap sederhana, tetapi tidak boleh menghilangkan usability fundamentals.

---

# 38. Audio Policy

Audio adalah enhancement penting terutama untuk TK.

Jenis:

```text
Instruction
Correct feedback
Encouragement
Optional explanation
```

Rules:

- Jangan autoplay audio pada marketing page.
- Child audio boleh dipicu melalui explicit interaction.
- Provide mute control.
- Audio file harus compressed.
- Semua production audio harus mempunyai provenance/license.

---

# 39. Security

Minimum:

- HTTPS.
- Secure cookie/session.
- CSRF protection sesuai auth architecture.
- Password hashing.
- Rate limiting.
- Input validation.
- Authorization server-side.
- Subscription validation server-side.
- Object storage access control.
- Admin RBAC.
- Audit logging.
- Secret management via environment/secret store.
- Database backup.
- Dependency scanning.

Jangan menyimpan:

- payment credentials;
- unnecessary child PII;
- secret API keys;
- password plaintext.

---

# 40. Child Data Protection

Karena platform memproses data anak, privacy harus menjadi architectural requirement.

UU No. 27 Tahun 2022 mengatur bahwa pemrosesan data pribadi anak diselenggarakan secara khusus dan memerlukan persetujuan orang tua dan/atau wali sesuai ketentuan. PP No. 17 Tahun 2025 tentang Tata Kelola Penyelenggaraan Sistem Elektronik dalam Pelindungan Anak berlaku sejak 27 Maret 2025. Permenkomdigi No. 9 Tahun 2026 mengatur ketentuan pelaksanaan terkait pelindungan anak pada sistem elektronik.  

Referensi resmi:

- https://jdih.komdigi.go.id/produk_hukum/view/id/832/t/undangundang%2Bnomor%2B27%2Btahun%2B2022
- https://peraturan.bpk.go.id/Details/316698/pp-no-17-tahun-2025
- https://peraturan.bpk.go.id/Details/346040/permenkomdigi-no-9-tahun-2026

Implementasi produk harus ditinjau secara legal sebelum public launch.

## Privacy by Design

Collect minimum data.

Contoh child profile:

```text
Allowed / useful:
- nickname
- grade
- avatar
- learning progress

Avoid unless necessary:
- full address
- phone number
- unnecessary location
- social profile
- public photo
```

Jangan menggunakan data child untuk advertising/profile commercial targeting tanpa dasar hukum dan review compliance yang sesuai.

---

# 41. API Design

Base:

```text
/api/v1
```

Contoh:

```text
POST   /auth/register
POST   /auth/login
POST   /auth/logout
GET    /me

GET    /children
POST   /children
GET    /children/:id
PATCH  /children/:id

GET    /subjects
GET    /skills

POST   /assessments
GET    /assessments/:id
POST   /assessments/:id/answers
POST   /assessments/:id/complete

GET    /children/:id/progress
GET    /children/:id/history
GET    /children/:id/recommendations

GET    /subscription
POST   /subscription/checkout

POST   /webhooks/payment
```

API contract sebaiknya didokumentasikan dengan OpenAPI.

---

# 42. Assessment Transaction

Question attempt harus immutable setelah finalized.

Contoh:

```text
AssessmentAttempt
├── started_at
├── completed_at
├── score
├── accuracy
└── items

QuestionAttempt
├── question_id
├── selected_answer
├── correct
├── response_time_ms
├── hint_used
└── created_at
```

Raw attempts disimpan agar progress dapat dihitung ulang jika algoritma mastery berubah.

---

# 43. Progress Engine

Jangan hanya menyimpan:

```text
score = 85
```

Simpan raw evidence.

Contoh:

```text
QuestionAttempt
      ↓
Skill Aggregation
      ↓
Skill Mastery
      ↓
Subject Progress
      ↓
Overall Progress
```

Dengan demikian algorithm mastery dapat berubah tanpa kehilangan historical attempts.

---

# 44. Admin Asset Library

Admin UI harus mempunyai:

```text
Asset Library
├── Search
├── Filter
├── Preview
├── Source
├── License
├── Creator
├── Usage
└── Approval Status
```

Asset lifecycle:

```text
Imported
  ↓
License Verified
  ↓
Design Approved
  ↓
Performance Checked
  ↓
Published
```

---

# 45. Design Review Gate

Tidak boleh langsung coding setelah menerima asset.

Workflow:

```text
Reference Collection
        ↓
Moodboard
        ↓
Brand Direction
        ↓
Asset Shortlist
        ↓
Design System
        ↓
Prototype
        ↓
Mobile Review
        ↓
Development
```

---

# 46. Anti-AI-Slop Design Review Checklist

Sebelum halaman dianggap selesai:

### Visual

- [ ] One coherent illustration style.
- [ ] No random asset style mixing.
- [ ] No unnecessary gradients.
- [ ] No excessive glassmorphism.
- [ ] No generic AI-looking character.
- [ ] No decorative visual without purpose.
- [ ] Color palette consistent.
- [ ] Typography hierarchy consistent.

### Brand

- [ ] Logo follows brand guideline.
- [ ] Character system consistent.
- [ ] Illustration style consistent.
- [ ] Icon style consistent.
- [ ] Subject visuals follow same art direction.

### Asset

- [ ] Source recorded.
- [ ] License verified.
- [ ] Creator recorded.
- [ ] Original asset preserved.
- [ ] Modification recorded.
- [ ] Attribution requirement checked.

### UX

- [ ] Visual helps comprehension.
- [ ] Visual does not distract.
- [ ] Child can understand primary action.
- [ ] Parent can understand value.
- [ ] Mobile interaction feels intentional.

---

# 47. Mobile QA Checklist

For every page:

- [ ] 320px width works.
- [ ] 360px width works.
- [ ] 390px width works.
- [ ] 412px width works.
- [ ] Tablet works.
- [ ] Desktop works.
- [ ] No horizontal overflow.
- [ ] Touch targets are adequate.
- [ ] Text is readable.
- [ ] CTA is reachable.
- [ ] Keyboard does not cover critical controls.
- [ ] Modal fits viewport.
- [ ] Sticky elements do not cover content.
- [ ] Images scale correctly.
- [ ] SVG remains sharp.
- [ ] Animation does not break interaction.

---

# 48. Testing Strategy

## Frontend

```text
Unit
Component
Integration
E2E
```

Recommended:

```text
Vitest
Playwright
```

## Backend

```text
Unit
Integration
API contract
Database
```

Recommended:

```text
Go testing
httptest
Testcontainers
```

## Critical E2E

At minimum:

```text
Register Parent
→ Create Child
→ Start Challenge
→ Answer Questions
→ Complete
→ View Result
→ View Parent Progress
→ Subscribe
→ Payment Webhook
→ Subscription Active
```

---

# 49. Observability

Track:

```text
API latency
HTTP error rate
DB latency
DB connections
assessment completion
question completion
payment failures
subscription status
frontend JS errors
Core Web Vitals
```

Do not log sensitive child data unnecessarily.

---

# 50. Analytics

Business analytics:

```text
landing_visit
register
child_created
challenge_started
question_answered
challenge_completed
progress_viewed
pricing_viewed
checkout_started
payment_success
subscription_cancelled
```

Product analytics:

```text
session_length
questions_per_session
completion_rate
accuracy
hint_usage
retry_usage
return_session
```

---

# 51. Core Business KPIs

Primary:

```text
Activation Rate
Challenge Completion Rate
Week-1 Retention
Week-4 Retention
Free → Paid Conversion
Monthly Churn
Paid Subscriber Count
```

Learning-oriented:

```text
Skill Mastery Improvement
Repeated Error Rate
Challenge Completion
Recommended Practice Completion
```

Do not optimize only for:

```text
time spent
number of clicks
number of questions
```

Longer usage is not automatically better learning.

---

# 52. SEO & Marketing

Marketing site should be indexable.

Pages:

```text
/
 /features
 /subjects
 /pricing
 /for-parents
 /faq
```

SEO requirements:

- semantic HTML;
- metadata;
- Open Graph;
- structured internal linking;
- sitemap;
- robots.txt;
- optimized images;
- Core Web Vitals.

Child learning app routes may be authenticated/private and do not need to be indexed.

---

# 53. Deployment

Production baseline:

```text
Docker
  ├── frontend
  ├── backend
  ├── worker
  ├── postgres
  └── reverse-proxy
```

Optional:

```text
redis
object-storage
observability
```

Do not add infrastructure without an actual requirement.

---

# 54. Environment Separation

Minimum:

```text
development
staging
production
```

Secrets must not be committed.

Example:

```text
DATABASE_URL
SESSION_SECRET
PAYMENT_SECRET
STORAGE_ACCESS_KEY
STORAGE_SECRET_KEY
```

---

# 55. Repository Structure

```text
edu-platform/
│
├── apps/
│   ├── web/
│   └── api/
│
├── packages/
│   ├── contracts/
│   ├── question-schema/
│   └── design-tokens/
│
├── content/
│   ├── curriculum/
│   ├── questions/
│   └── audio/
│
├── assets/
│   ├── illustrations/
│   ├── characters/
│   ├── icons/
│   ├── animations/
│   ├── sounds/
│   └── licenses/
│
├── infra/
│   ├── docker/
│   ├── compose/
│   └── monitoring/
│
├── docs/
│
└── README.md
```

---

# 56. Definition of Done — MVP

MVP tidak dianggap selesai hanya karena semua page sudah dibuat.

## Product

- [ ] Parent dapat register.
- [ ] Parent dapat membuat child.
- [ ] Child dapat memulai challenge.
- [ ] Minimal beberapa question types tersedia.
- [ ] Feedback berjalan.
- [ ] Assessment dapat diselesaikan.
- [ ] Progress tersimpan.
- [ ] Parent dapat melihat progress.
- [ ] Subscription berjalan.
- [ ] Admin dapat membuat dan publish question.

## Design

- [ ] Brand system defined.
- [ ] Asset provenance recorded.
- [ ] No unlicensed asset.
- [ ] No AI-generated production artwork.
- [ ] No random illustration mixing.
- [ ] Child UX validated.
- [ ] Parent UX validated.

## Technical

- [ ] Backend Go.
- [ ] PostgreSQL.
- [ ] Docker deployment.
- [ ] HTTPS.
- [ ] Authentication.
- [ ] Authorization.
- [ ] Database migration.
- [ ] Backup.
- [ ] Logging.
- [ ] Monitoring.
- [ ] CI/CD.

## Mobile

- [ ] 320px supported.
- [ ] 360px supported.
- [ ] 390px supported.
- [ ] 412px supported.
- [ ] Tablet supported.
- [ ] Desktop supported.
- [ ] No horizontal scroll.
- [ ] Child touch interactions are reliable.
- [ ] Performance tested on mobile hardware.

---

# 57. Recommended Development Order

## Phase 0 — Design Foundation

```text
Brand direction
→ Moodboard
→ Asset selection
→ Typography
→ Color tokens
→ Component tokens
```

## Phase 1 — Core Architecture

```text
Repository
→ Docker
→ Go API
→ PostgreSQL
→ SvelteKit
→ Auth
```

## Phase 2 — Content Engine

```text
Curriculum
→ Subjects
→ Skills
→ Question schema
→ Question renderer
→ Admin CMS
```

## Phase 3 — Child Experience

```text
Child profile
→ Subject
→ Challenge
→ Answer
→ Feedback
→ Result
→ Reward
```

## Phase 4 — Parent Experience

```text
Dashboard
→ Progress
→ History
→ Skill mastery
→ Recommendation
```

## Phase 5 — Subscription

```text
Pricing
→ Checkout
→ Webhook
→ Subscription state
→ Access control
```

## Phase 6 — Hardening

```text
Security
→ Privacy
→ Performance
→ Mobile QA
→ Analytics
→ Monitoring
→ Production deployment
```

---

# 58. Important Product Rule

The product must always answer these two questions:

### For the child

> "What should I do next, and why is it fun?"

### For the parent

> "What did my child learn, and what should they practice next?"

If a feature does not clearly help one of these two questions, it should not automatically enter the MVP.

---

# 59. Final Architecture Principle

Build the platform as:

```text
CONTENT
   ↓
QUESTION ENGINE
   ↓
ASSESSMENT ENGINE
   ↓
PROGRESS ENGINE
   ↓
PARENT INSIGHT
```

not:

```text
PAGE
→ FORM
→ SCORE
```

This distinction is important.

The long-term value of the product is not the number of questions stored in PostgreSQL.

The long-term value is the ability to transform:

```text
Question Attempts
       ↓
Learning Evidence
       ↓
Skill Mastery
       ↓
Actionable Recommendation
```

---

# 60. Final Design Principle

The visual identity should feel:

```text
Playful
Human
Warm
Educational
Confident
Simple
Memorable
```

and avoid:

```text
Generic
Over-polished
Artificial
Noisy
Template-like
AI-generated
```

The goal is not to make the website look "childish".

The goal is to make it look **designed specifically for children**.

---

# 61. Source & Reference Links

### Technical

- Svelte documentation: https://svelte.dev/docs
- Tailwind responsive design: https://tailwindcss.com/docs/responsive-design

### Brand / Visual Assets

- DrawKit license: https://www.drawkit.com/license
- DrawKit Education & e-Learning: https://www.drawkit.com/illustrations/education-e-learning-illustrations
- DrawKit Primary School: https://www.drawkit.com/illustrations/primary-school-education-illustrations
- Storyset: https://storyset.com/
- unDraw: https://undraw.co/
- LottieFiles license: https://lottiefiles.com/page/license

### Indonesia — Child & Data Protection

- UU No. 27 Tahun 2022 tentang Pelindungan Data Pribadi: https://jdih.komdigi.go.id/produk_hukum/view/id/832/t/undangundang%2Bnomor%2B27%2Btahun%2B2022
- PP No. 17 Tahun 2025: https://peraturan.bpk.go.id/Details/316698/pp-no-17-tahun-2025
- Permenkomdigi No. 9 Tahun 2026: https://peraturan.bpk.go.id/Details/346040/permenkomdigi-no-9-tahun-2026

---

# 62. Document Status

This document is intended to be the **MVP source of truth**.

Any implementation that materially changes:

- business logic;
- subscription logic;
- data model;
- assessment engine;
- child data handling;
- brand identity;
- asset sourcing;
- mobile requirements;

must update this document before the implementation is considered final.

---

# Appendix A. Implementation notes (2026-09-25)

Recorded per §62. Full rationale: `docs/decisions.md`.

**Data model (§19, §42)**
- `assessments` stores the child's challenge session and plays the role of *AssessmentAttempt*; there is no separate `assessment_attempts` table because sessions are generated adaptively per child.
- Question options live inside the immutable `question_versions.content` JSONB; there is no `question_options` table. The shape is documented in `packages/question-schema`.
- Added: `sessions`, `question_reviews` (quality-gate evidence), `payment_checkouts`, `payment_events` (webhook idempotency inbox), `analytics_events`.
- `question_attempts` rows cannot be updated (DB trigger). They are deleted only when a parent erases a child or the account.

**Business rules made explicit**
- Free tier: 1 child, 3 challenges/day (reset at Asia/Jakarta midnight), overall and subject progress plus one strong and one weak skill, last 3 history entries, first recommendation. All values are configurable.
- Subscription access: `active`/`trial`/`cancelled` until period end; `past_due` gets a 3-day grace; `expired` has none.
- Mastery bands: <40 needs practice, 40–70 developing, 70–85 proficient, >85 mastered. The parent view groups ≥70 as *strong*.

**Stack deviations**: hand-written pgx instead of sqlc; plain Svelte components instead of shadcn-svelte; browser speech synthesis instead of recorded audio; emoji as interim visuals until brand assets pass review.

# Appendix B. Production iteration (2026-09-25)

- **Payments:** Midtrans Snap. Webhook = signature check (§22) plus a Status API re-check; idempotent on `transaction_id:status`. Monthly plan is **pay-per-month without auto-renewal** (auto-renew deferred). Each payment adds one month; 3-day grace after the end date, then back to free.
- **Storage:** S3-compatible object storage (bucket private, served through the API). SeaweedFS in development.
- **Learning logic added:** skill-focused sessions; **parent missions** (turn a recommendation into "N challenges on skill X", max 3 active, shown first on the child home, badge on completion); learning **streak**; mastery aggregated per **chapter** (Child → Subject → Chapter → Skill, §11).
- **Data model:** `assessments.skill_id`, `assessments.mission_id`, new table `child_missions`.
- **Brand/visuals:** no emoji anywhere. Question content uses `figures` (approved icon names + palette colours) instead of free-text visuals. Icons: Phosphor (MIT). Illustrations: unDraw, recoloured, provenance registered. Typography: Plus Jakarta Sans (one family).
- **Deferred by product owner:** transactional email, auto-renewal.
