-- Seed AI models (Claude and GPT)

-- Claude models
INSERT INTO ai_models (id, name, provider, model_version, is_active, created_at, updated_at)
VALUES
  ('00000000-0000-0000-0000-000000000101', 'Claude 3.5 Sonnet', 'anthropic', 'claude-3-5-sonnet-20241022', true, NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000102', 'Claude 3 Opus', 'anthropic', 'claude-3-opus-20240229', true, NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000103', 'Claude 3 Sonnet', 'anthropic', 'claude-3-sonnet-20240229', true, NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000104', 'Claude 3 Haiku', 'anthropic', 'claude-3-haiku-20240307', true, NOW(), NOW());

-- OpenAI GPT models
INSERT INTO ai_models (id, name, provider, model_version, is_active, created_at, updated_at)
VALUES
  ('00000000-0000-0000-0000-000000000201', 'GPT-4 Turbo', 'openai', 'gpt-4-turbo-preview', true, NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000202', 'GPT-4', 'openai', 'gpt-4', true, NOW(), NOW()),
  ('00000000-0000-0000-0000-000000000203', 'GPT-3.5 Turbo', 'openai', 'gpt-3.5-turbo', true, NOW(), NOW());
