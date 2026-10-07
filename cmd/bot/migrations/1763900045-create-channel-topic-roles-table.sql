CREATE TABLE channel_topic_roles (
    role_id TEXT NOT NULL,
    guild_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    created_by TEXT NOT NULL,

    last_invoked TIMESTAMP with time zone NOT NULL,
    last_invoker_id TEXT NULL

    PRIMARY KEY (role_id, guild_id)
)

CREATE INDEX channel_topic_roles_last_invoker_idx ON channel_topic_roles (last_invoker_id, guild_id);

comment on column channel_topic_roles.role_id is 'Discord role ID';
comment on column channel_topic_roles.guild_id is 'Discord server/guild ID';
comment on column channel_topic_roles.channel_id is 'Discord channel ID';
comment on column channel_topic_roles.created_by is 'Discord user ID of the user who created the role';

comment on column channel_topic_roles.last_invoked is 'Timestamp of the last time the at ping command was invoked';
comment on column channel_topic_roles.last_invoker_id is 'Discord user ID of the user who last invoked the at ping command';