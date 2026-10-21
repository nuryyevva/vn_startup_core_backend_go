UPDATE scenes SET sprite_framing = 'knees' WHERE sprite_framing = 'waist';
ALTER TABLE scenes DROP CONSTRAINT scenes_sprite_framing_check;
ALTER TABLE scenes
    ADD CONSTRAINT scenes_sprite_framing_check CHECK (sprite_framing IN ('full', 'knees'));
