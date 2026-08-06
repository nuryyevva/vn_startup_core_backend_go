DROP INDEX IF EXISTS idx_stories_genre;
ALTER TABLE stories DROP CONSTRAINT IF EXISTS stories_genre_check;
ALTER TABLE stories
    DROP COLUMN status,
    DROP COLUMN genre;
