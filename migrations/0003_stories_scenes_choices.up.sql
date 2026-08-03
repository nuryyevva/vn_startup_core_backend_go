CREATE TABLE stories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT,
    cover_url TEXT,
    is_published BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE scenes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    order_index INT NOT NULL,
    background_url TEXT,
    character_id UUID,
    dialogue_script JSONB NOT NULL DEFAULT '[]',
    free_dialog_enabled BOOLEAN NOT NULL DEFAULT false,
    dialog_limit_type TEXT CHECK (dialog_limit_type IN ('time', 'messages', 'none')),
    dialog_limit_value INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE choices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    text TEXT NOT NULL,
    is_paid BOOLEAN NOT NULL DEFAULT false,
    cost_diamonds INT NOT NULL DEFAULT 0,
    next_scene_id UUID REFERENCES scenes(id)
);

CREATE INDEX idx_scenes_story ON scenes(story_id);
CREATE INDEX idx_choices_scene ON choices(scene_id);
