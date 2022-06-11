
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
    monitored INT
);
