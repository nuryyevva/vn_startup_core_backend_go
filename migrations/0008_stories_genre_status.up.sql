ALTER TABLE stories
    ADD COLUMN genre TEXT NOT NULL DEFAULT 'romance',
    ADD COLUMN status TEXT NOT NULL DEFAULT 'ongoing' CHECK (status IN ('ongoing', 'completed'));

ALTER TABLE stories
    ADD CONSTRAINT stories_genre_check CHECK (genre IN (
        'romance', 'fantasy', 'sci_fi', 'mystery', 'drama', 'slice_of_life', 'horror', 'adventure'
    ));

CREATE INDEX idx_stories_genre ON stories(genre);
