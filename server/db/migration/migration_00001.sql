-- begin transaction / auto handled by migrations



CREATE TABLE IF NOT EXISTS config (
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL
);
CREATE INDEX config_index_key on config(key);

-- CREATE TABLE IF NOT EXISTS events (
--     id integer primary key autoincrement,
--     topic text NOT NULL,
--     message text NOT NULL,
--     created_at integer(4) not null default (strftime('%s','now'))
-- );

-- commit transaction / Auto handled by migrations
