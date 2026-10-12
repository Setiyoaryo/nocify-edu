# Nocify EDU — Product Requirements Document

**Google OAuth2 / OpenID Connect + Family Authentication**

Version 1.0 | 12 October 2026 | Status: initial implementation and phased rollout

## 1. Overview

Google OAuth 2.0 + OpenID Connect for parent sign-in while retaining existing WhatsApp OTP and email/password. Existing Go backend remains the source of truth for cookie sessions, family ownership and RBAC. First slice delivers parent OIDC only; child devices require separate authorization hardening before public launch.

## 2. Scope and objectives

New parent Google onboarding; returning linked parent login; preserve original parent UUID, children and subscriptions; explicit parental privacy consent; recovery password; Google-disabled fallback; no auto-link by email; WhatsApp send/verify routes untouched. Non-goals: Gmail/Drive access, Google accounts for children, removing WhatsApp, and shipping child pairing in this slice.

## 3. Personas and permissions

Parent: Google OIDC or existing sign-in, manages child profiles, progress and subscriptions. Child: profile + server-verified PIN on a paired browser, restricted to learning. Staff: existing credential + RBAC; Google does not confer admin access. Parent and child sessions must be separate at the server authorization layer.

## 4. Login journeys

Returning parent: choose Orang Tua > Google button > Google authorization > Go validates provider response > look up auth identity by Google sub > issue existing HttpOnly session > /app/parent. New parent: Google verification > ten-minute onboarding ticket > name, recovery password, explicit legal consent > insert user and identity atomically > family child creation. Existing email without linked subject: reject with account_exists; parent logs in existing method, then initiates explicit link flow. Provider errors redirect to login with generic error. WhatsApp remains visible as alternative.

## 5. Child and pairing journeys (P0 security phase)

On new browser, parent approves device via authenticated Google parent session or approves short-lived one-time pairing challenge on their own browser. Create unique opaque device credential, hash at rest and place in Secure HttpOnly cookie. Device resolves one family only, never a parent authorization. GET child profiles from verified family. Child chooses one profile and supplies 4-6 digit PIN; backend verifies child ownership, bcrypt PIN, rate limit and device state; create child-scoped session. Existing browser may resume active child session. Revoked/expired device must re-pair. No MAC/IP fingerprinting. Parent reauthentication required for parent mode; arithmetic ParentGate is not a security boundary.

## 6. Current database

Retain users, children, sessions and OTP phone schema. New migration 0011_google_oidc.sql creates auth_identities(provider,provider_subject,user_id,email_at_link) with provider-subject uniqueness, google_oauth_flows (single-use hashed state, PKCE verifier, nonce, expiry, optional link_user_id) and google_signup_flows (single-use ticket, Google sub, verified email, expiry). Use stable Google sub instead of email as login principal. For child phase, add trusted_devices, actor_type and child_id session constraints, PIN attempt tracking, pairing challenges.

## 7. Security

Authorization Code flow with PKCE S256, CSRF state, OIDC nonce, 10-minute one-time state. Validate signed RS256 ID token using trusted Google JWKS, exact audience, issuer, exp, iat and email_verified. Do not store Google tokens. Existing session: opaque random secret in HttpOnly cookie, HMAC hashed DB key, Secure on HTTPS, SameSite=Lax. Do not promote child to parent from query param, route choice or frontend sessionStorage. Do not autolink accounts by email. Require consent before first family creation. OAuth callback safe for multi-instance DB backend; remove expired pending rows regularly. Add real-provider staging tests and data-protection review.

## 8. Current API contract

GET /api/v1/auth/google/status returns enabled boolean. GET /api/v1/auth/google/start redirects to Google. GET /api/v1/auth/google/callback validates state, token, account and redirects. GET /api/v1/auth/google/pending returns temporary pre-signup identity. POST /api/v1/auth/google/complete accepts display_name,password,consent and creates family+session. GET /api/v1/auth/google/start?link=1 has backend link support but settings UI and step-up auth are follow-on tasks. Existing email and WhatsApp endpoints remain untouched.

## 9. Proposed child API contract

POST /api/v1/auth/devices/pair/start; POST /api/v1/auth/devices/pair/approve; GET /api/v1/auth/devices; DELETE /api/v1/auth/devices/{id}; GET /api/v1/auth/child/profiles; POST /api/v1/auth/child/login; POST /api/v1/auth/child/logout. Every child endpoint must authorize actor_type=child, matching child_id, verified trusted device and parent_id ownership; all billing, account edits and parent profile management must reject child credentials.

## 10. UX and errors

Keep top-level mode choice Orang Tua vs Anak. Only parent screen gains Google button; existing WhatsApp and email alternatives stay accessible. Google button is hidden if not configured. Show errors for cancelled login, expired flow, duplicate existing account, provider down, invalid identity; avoid exposing raw token/provider details. New Google family completes recovery password and explicit parental consent. Child profile names are not placeholders; load only from backend-verified family. New browser without binding requests parent authorization, never shows arbitrary profiles.

## 11. Deployment

Register Google Cloud OAuth Web Application, configure Authorized Redirect URI equal to PUBLIC_URL + /api/v1/auth/google/callback. Inject GOOGLE_CLIENT_ID, GOOGLE_CLIENT_SECRET and GOOGLE_REDIRECT_URI into Go API service using secrets; HTTPS production. Empty values disable Google cleanly. Deploy DB migration before enabling Google. Rehearse staging with Google test users and real HTTPS callback. Maintain WhatsApp and email rollback path. Disable Google credentials/button if rollback needed, preserve identity records.

## 12. Tests and acceptance criteria

Unit/integration: state mismatch/missing/replayed/expired; invalid Google signature, key, issuer, audience, nonce, expiry, unverified email; complete requires consent/password, atomic insert, concurrent unique conflicts, no account merging; linked Google retains family data; disabled feature fallback. Web: Google visibility configured/unconfigured, login redirect, consent UI, existing WhatsApp intact. Staging: successful first login and returning login with registered Google client; callback SameSite/HTTPS. Child security before public launch: child cannot GET /me or call /subscription, parent CRUD and account deletion; sibling/cross-family scoping; device revoke kills child entry; PIN rate limits; role-switch reauth. Release requirement: 100% security tests, no cross-account merges, zero child-to-parent privilege escalation.

## 13. Risks and phased delivery

Phase 1 (current implementation): OIDC parent login, onboarding, migration, config, consent UI, retained WhatsApp. Phase 1.1: linking/unlink UI and step-up, account recovery, token verifier tests and staging validation. Phase 2 P0: parent-approved child pairing, distinct child session, server-side scoped authorization and child access tests. Phase 3: production privacy/legal review, rate/monitoring dashboards, cleanup of stale OAuth rows, runbooks. Existing WhatsApp send OTP is a stub and existing child mode inherits parent session: both are preexisting launch blockers, not solved by this OAuth change.
