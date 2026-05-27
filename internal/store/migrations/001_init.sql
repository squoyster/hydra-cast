-- 001_init.sql: Initial schema for HydraCast

CREATE TABLE IF NOT EXISTS media_items (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source_name TEXT NOT NULL,
    source_type TEXT NOT NULL,
    external_id TEXT NOT NULL,
    source_url TEXT NOT NULL,
    title TEXT,
    media_type TEXT NOT NULL,
    published_at TEXT,
    detected_at TEXT NOT NULL,
    fingerprint TEXT NOT NULL,
    raw_metadata_json TEXT,
    UNIQUE(source_name, external_id)
);

CREATE TABLE IF NOT EXISTS jobs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_item_id INTEGER NOT NULL,
    job_type TEXT NOT NULL,
    status TEXT NOT NULL,
    started_at TEXT,
    finished_at TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    FOREIGN KEY(media_item_id) REFERENCES media_items(id)
);

CREATE TABLE IF NOT EXISTS publish_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    media_item_id INTEGER NOT NULL,
    destination_name TEXT NOT NULL,
    destination_type TEXT NOT NULL,
    status TEXT NOT NULL,
    remote_id TEXT,
    remote_url TEXT,
    published_at TEXT,
    error_message TEXT,
    UNIQUE(media_item_id, destination_name),
    FOREIGN KEY(media_item_id) REFERENCES media_items(id)
);

CREATE TABLE IF NOT EXISTS job_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp TEXT NOT NULL,
    job_id INTEGER,
    level TEXT NOT NULL,
    component TEXT NOT NULL,
    message TEXT NOT NULL,
    context_json TEXT
);

-- Index for recent events query
CREATE INDEX IF NOT EXISTS idx_job_events_timestamp ON job_events(timestamp DESC);

-- Index for job lookup by media item
CREATE INDEX IF NOT EXISTS idx_jobs_media_item_id ON jobs(media_item_id);

-- Index for publish results lookup
CREATE INDEX IF NOT EXISTS idx_publish_results_media_item_id ON publish_results(media_item_id);
