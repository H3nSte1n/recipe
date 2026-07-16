UPDATE ai_models
SET name = 'Claude 3.5 Sonnet', model_version = 'claude-3-5-sonnet-20241022', updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000101';

UPDATE ai_models
SET name = 'Claude 3 Opus', model_version = 'claude-3-opus-20240229', updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000102';

UPDATE ai_models
SET is_active = true, updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000103';

UPDATE ai_models
SET name = 'Claude 3 Haiku', model_version = 'claude-3-haiku-20240307', updated_at = NOW()
WHERE id = '00000000-0000-0000-0000-000000000104';
