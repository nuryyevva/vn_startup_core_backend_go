-- A story that is listed in the catalog but can't be played yet: the
-- client shows it as "coming soon", and the API refuses its scenes and
-- progress (403 story_coming_soon). The two sample fairy tales are the
-- first such stories.
ALTER TABLE stories ADD COLUMN coming_soon BOOLEAN NOT NULL DEFAULT false;

UPDATE stories SET coming_soon = true
WHERE id IN ('11111111-1111-1111-1111-111111111101', '22222222-2222-2222-2222-222222222201');
