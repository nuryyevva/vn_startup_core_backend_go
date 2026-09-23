ALTER TABLE user_profiles
    ADD COLUMN notify_new_chapters BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN notify_promotional_offers BOOLEAN NOT NULL DEFAULT false;
