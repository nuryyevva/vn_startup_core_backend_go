-- How the character sprite is framed on screen at the start of a scene:
-- 'full' shows the whole figure, 'knees' a closer shot cropped just below
-- the knees. Only the shot is stored here — how much of the image that
-- crops is the client's business. A dialogue_script line can switch it
-- mid-scene with its own "sprite_framing" key (see README "Сцены").
ALTER TABLE scenes
    ADD COLUMN sprite_framing TEXT NOT NULL DEFAULT 'full'
        CONSTRAINT scenes_sprite_framing_check CHECK (sprite_framing IN ('full', 'knees'));
