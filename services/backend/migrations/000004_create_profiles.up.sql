CREATE TABLE profiles (
                          id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
                          user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                          bio TEXT,
                          location VARCHAR(255),
                          avatar_url VARCHAR(255),
                          website_url VARCHAR(255),
                          created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                          updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
                          CONSTRAINT profiles_user_id_key UNIQUE (user_id)
);

CREATE INDEX idx_profiles_user_id ON profiles(user_id);