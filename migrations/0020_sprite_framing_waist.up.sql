-- A third shot: 'waist' — the character cropped at the waist, the usual shot of a conversation.
ALTER TABLE scenes DROP CONSTRAINT scenes_sprite_framing_check;
ALTER TABLE scenes
    ADD CONSTRAINT scenes_sprite_framing_check CHECK (sprite_framing IN ('full', 'knees', 'waist'));
