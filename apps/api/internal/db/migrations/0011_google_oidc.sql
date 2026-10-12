-- Add Google OpenID Connect identities without touching legacy WhatsApp OTP.
CREATE TABLE IF NOT EXISTS auth_identities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider text NOT NULL,
    provider_subject text NOT NULL,
    email_at_link text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_subject),
    UNIQUE (user_id, provider)
);

-- OAuth code verifier/nonce live only on the server. State is hashed at rest.
CREATE TABLE IF NOT EXISTS google_oauth_flows (
    state_hash text PRIMARY KEY,
    nonce text NOT NULL,
    code_verifier text NOT NULL,
    link_user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS google_oauth_flows_expiry_idx ON google_oauth_flows(expires_at);

-- Consent is requested AFTER Google login and BEFORE a new family is created.
CREATE TABLE IF NOT EXISTS google_signup_flows (
    ticket_hash text PRIMARY KEY,
    google_sub text NOT NULL,
    email text NOT NULL,
    display_name text NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS google_signup_flows_expiry_idx ON google_signup_flows(expires_at);
