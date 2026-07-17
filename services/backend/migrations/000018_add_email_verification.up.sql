ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMP WITH TIME ZONE NULL;

CREATE TABLE email_verification_tokens (
                                       id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                                       user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                       token VARCHAR(255) NOT NULL,
                                       expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
                                       used BOOLEAN DEFAULT FALSE,
                                       created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                       updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                                       CONSTRAINT email_verification_tokens_token_key UNIQUE (token)
);

-- Create indexes
CREATE INDEX idx_email_verification_tokens_token ON email_verification_tokens(token);
CREATE INDEX idx_email_verification_tokens_user_id ON email_verification_tokens(user_id);
