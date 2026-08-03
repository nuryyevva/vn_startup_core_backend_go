CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name TEXT,
    gender TEXT,
    hobbies TEXT[] DEFAULT '{}',
    theme TEXT NOT NULL DEFAULT 'dark',
    language TEXT NOT NULL DEFAULT 'ru',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
