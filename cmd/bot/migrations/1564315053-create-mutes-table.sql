CREATE TABLE discord_mutes (
    id SERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    reason TEXT NOT NULL,
    mute_start TIMESTAMP with time zone NOT NULL,
    mute_end TIMESTAMP with time zone NOT NULL
);

CREATE UNIQUE INDEX guild_user_idx ON discord_mutes (guild_id, user_id);
CREATE INDEX mute_end_idx ON discord_mutes (mute_end);
