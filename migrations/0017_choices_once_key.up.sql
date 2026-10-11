-- A choice with a once_key can be taken only once per visit to its scene:
-- "look around the room" hubs loop back to the same choices, and without
-- this the options already explored stay on the list forever. Choices that
-- share a once_key within a story are the same option offered from more
-- than one scene (a hub and the scene that leads into it).
ALTER TABLE choices ADD COLUMN once_key TEXT;
