-- Two levels of structure for branching stories: chapter_title marks the
-- scene that starts a chapter, and title (0015) now marks the scene that
-- starts a named section inside it — what the chapter list shows when a
-- chapter is expanded, so a reader can find where they stopped. Everything
-- titled so far was a chapter start, so carry those over.
ALTER TABLE scenes ADD COLUMN chapter_title TEXT;
UPDATE scenes SET chapter_title = title, title = NULL WHERE title IS NOT NULL;
