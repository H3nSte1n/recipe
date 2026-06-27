-- API keys are now stored encrypted (AES-256-GCM, base64-encoded), which is longer
-- than the raw key. Widen the column from VARCHAR(255) to TEXT to remove the limit.
ALTER TABLE user_ai_configs ALTER COLUMN api_key TYPE TEXT;
