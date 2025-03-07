CREATE TABLE user_ai_configs (
                                 id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                 ai_model_id UUID NOT NULL REFERENCES ai_models(id),
                                 api_key VARCHAR(255) NOT NULL,
                                 is_default BOOLEAN DEFAULT false,
                                 settings JSONB DEFAULT '{}',  -- For model-specific settings
                                 created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                 updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                 CONSTRAINT user_ai_configs_user_model_key UNIQUE(user_id, ai_model_id)
);

CREATE INDEX idx_user_ai_configs_user_id ON user_ai_configs(user_id);
CREATE INDEX idx_user_ai_configs_ai_model_id ON user_ai_configs(ai_model_id);