CREATE TABLE password_reset_tokens (
                                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                       user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                       token VARCHAR(255) NOT NULL,
                                       expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
                                       used BOOLEAN DEFAULT FALSE,
                                       created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                       updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                       CONSTRAINT password_reset_tokens_token_key UNIQUE (token)
);

-- Create indexes
CREATE INDEX idx_password_reset_tokens_token ON password_reset_tokens(token);
CREATE INDEX idx_password_reset_tokens_user_id ON password_reset_tokens(user_id);