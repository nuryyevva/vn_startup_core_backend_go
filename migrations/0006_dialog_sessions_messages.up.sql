CREATE TABLE dialog_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    character_id UUID NOT NULL,
    scene_id UUID NOT NULL REFERENCES scenes(id),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'ended', 'interrupted')),
    limit_type TEXT NOT NULL CHECK (limit_type IN ('time', 'messages', 'none')),
    limit_value INT,
    message_count INT NOT NULL DEFAULT 0,
    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    end_reason TEXT
);

CREATE TABLE dialog_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id UUID NOT NULL REFERENCES dialog_sessions(id) ON DELETE CASCADE,
    sender TEXT NOT NULL CHECK (sender IN ('user', 'character')),
    text TEXT NOT NULL,
    cost_diamonds INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_dialog_messages_session ON dialog_messages(session_id);
CREATE INDEX idx_dialog_sessions_user ON dialog_sessions(user_id);
CREATE INDEX idx_dialog_sessions_active ON dialog_sessions(status) WHERE status = 'active';
