-- The name of the character a free dialog in the scene is held with, shown by the client in the chat header and on the
-- talk button. Without it the client only knows the name from the scene's lines, and a pure talk beat has none.
ALTER TABLE scenes ADD COLUMN character_name TEXT;
