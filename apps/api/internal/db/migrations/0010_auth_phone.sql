-- WhatsApp phone number support for OTP login
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone text;
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_idx ON users (phone) WHERE phone IS NOT NULL;
UPDATE users SET phone = '+6281234567890' WHERE email = 'demo@eduplay.local' AND phone IS NULL;
