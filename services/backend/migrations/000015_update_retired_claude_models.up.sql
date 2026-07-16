-- Replace retired Claude model IDs with their current equivalents.
-- claude-3-5-sonnet-20241022 and claude-3-sonnet-20240229 both map to Claude Sonnet 5;
-- keep one row per (provider, model_version) pair, so retire the duplicate Sonnet row.
UPDATE ai_models
SET name = 'Claude Sonnet 5', model_version = 'claude-sonnet-5', updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000101';

UPDATE ai_models
SET name = 'Claude Opus 4.8', model_version = 'claude-opus-4-8', updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000102';

UPDATE ai_models
SET is_active = false, updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000103';

UPDATE ai_models
SET name = 'Claude Haiku 4.5', model_version = 'claude-haiku-4-5', updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000104';
