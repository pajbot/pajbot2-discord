ALTER TABLE discord_messages
ADD COLUMN author_id TEXT NOT NULL DEFAULT 'unknown';
