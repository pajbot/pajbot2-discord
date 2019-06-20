CREATE TABLE config (
    server_id TEXT NOT NULL,
    key TEXT NOT NULL,
    value TEXT NOT NULL,

    PRIMARY KEY (server_id, key)
);

comment on column config.server_id is 'Discord server/guild ID';
comment on column config.key is 'Config key';
comment on column config.value is 'Config value';
