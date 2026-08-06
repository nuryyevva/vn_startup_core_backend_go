ALTER TABLE scenes ADD COLUMN unlock_cost_diamonds INT;

CREATE TABLE scene_unlocks (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scene_id UUID NOT NULL REFERENCES scenes(id) ON DELETE CASCADE,
    unlocked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, scene_id)
);

CREATE INDEX idx_scene_unlocks_user ON scene_unlocks(user_id);
