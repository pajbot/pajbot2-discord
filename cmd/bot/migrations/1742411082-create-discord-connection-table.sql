CREATE TABLE discord_connection (
	id SERIAL PRIMARY KEY,
	discord_user_id VARCHAR(24) NOT NULL,
	discord_user_name VARCHAR(48) NOT NULL,
	discord_guild_id VARCHAR(64) NOT NULL,
    twitch_user_id VARCHAR(64) NOT NULL,
    twitch_user_login VARCHAR(64) NOT NULL
);

CREATE UNIQUE INDEX discord_connection_idx ON discord_connection (discord_guild_id, discord_user_id);
CREATE INDEX discord_connection_twitch_user_idx ON discord_connection (twitch_user_id);
