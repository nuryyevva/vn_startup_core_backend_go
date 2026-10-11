UPDATE scenes SET title = chapter_title WHERE chapter_title IS NOT NULL;
ALTER TABLE scenes DROP COLUMN chapter_title;
