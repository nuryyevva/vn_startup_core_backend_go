-- Per-story look of the scene player (dialogue box, choice buttons): a
-- small JSON object naming a client-side preset and/or overriding single
-- design tokens, e.g. {"preset": "mist", "accent_color": "#9FC3D9"}.
-- NULL keeps the app's default look. JSONB rather than columns because the
-- token vocabulary is the client's and grows without schema changes.
ALTER TABLE stories ADD COLUMN player_theme JSONB;
