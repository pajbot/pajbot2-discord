CREATE TABLE twitchstreamannouncer (
	id SERIAL PRIMARY KEY,
	discord_guild_id VARCHAR(64) NOT NULL,
	twitch_user_id VARCHAR(64) NOT NULL,
	twitch_stream_id VARCHAR(64) NOT NULL
);

comment on column twitchstreamannouncer.twitch_user_id is 'twitch user ID of the streamer whose stream we announced';
comment on column twitchstreamannouncer.twitch_stream_id is 'twitch stream ID of the stream we announced';

CREATE UNIQUE INDEX twitchstreamannouncer_streamer_idx ON twitchstreamannouncer (discord_guild_id, twitch_user_id);
