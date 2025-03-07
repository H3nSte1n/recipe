CREATE TABLE ai_models (
                           id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                           name VARCHAR(100) NOT NULL,
                           provider VARCHAR(50) NOT NULL, -- e.g., 'openai', 'anthropic', etc.
                           model_version VARCHAR(50) NOT NULL,
                           is_active BOOLEAN DEFAULT true,
                           created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                           updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                           UNIQUE(provider, name, model_version)
);

CREATE INDEX idx_ai_models_provider_name ON ai_models(provider, name);