-- A scene with a non-null title starts a chapter. Branching stories are
-- authored as many short scenes (one per branch beat) linked by choices, so
-- the story detail screen lists only titled scenes as chapters instead of
-- every beat; stories without any titled scene fall back to listing all
-- scenes, as before.
ALTER TABLE scenes ADD COLUMN title TEXT;
