# Child data protection (§40)

**Needs legal review before public launch** (UU 27/2022 PDP, PP 17/2025, Permenkomdigi 9/2026).

- **Consent:** registration requires the parent/guardian consent checkbox. `users.consent_version` and `consent_at` record it.
- **Minimum data:** a child profile holds nickname, grade, avatar, optional 4-digit PIN (bcrypt) and learning evidence. No birth date, photo, address, phone, school or location.
- **Parent boundary:** every child/assessment query is scoped by `parent_id`; other parents get 404.
- **Erasure:** deleting a child or the account cascades through all attempts, progress and inventory. Analytics rows keep only a nulled reference.
- **No advertising or profiling:** analytics are first-party, allow-listed, and hold opaque ids only.
- **Logs and metrics:** no request bodies are logged; metrics use route patterns.
- **Fonts are self-hosted** (@fontsource), so no third-party requests from child pages.
- **Speech synthesis** runs in the browser; no audio leaves the device.
