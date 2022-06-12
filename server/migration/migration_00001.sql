-- begin transaction / auto handled by migrations



CREATE TABLE IF NOT EXISTS config (
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS config_index_key on config(key);

CREATE TABLE IF NOT EXISTS sonarr (
    id INT,
    title TEXT NOT NULL,
    status TEXT,
    overview TEXT,
    previous_airing TEXT,
    network TEXT,
    added TEXT,
    genres TEXT,
    seasons INT,
    monitored INT,
    RAW TEXT
);
CREATE TABLE IF NOT EXISTS radarr (
    id INT,
    title TEXT NOT NULL,
    status TEXT,
    overview TEXT,
    added TEXT,
    genres TEXT,
    is_available INT,
    monitored INT,
    RAW TEXT
);

-- CREATE TABLE IF NOT EXISTS events (
--     id integer primary key autoincrement,
--     topic text NOT NULL,
--     message text NOT NULL,
--     created_at integer(4) not null default (strftime('%s','now'))
-- );

-- commit transaction / Auto handled by migrations
