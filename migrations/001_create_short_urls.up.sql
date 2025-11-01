CREATE TABLE IF NOT EXISTS short_urls (
    id VARCHAR(255) PRIMARY KEY,
    short_url VARCHAR(500) NOT NULL,
    original_url TEXT NOT NULL UNIQUE,
    user_id VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_deleted BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_short_url ON short_urls(short_url);
CREATE INDEX IF NOT EXISTS idx_original_url ON short_urls(original_url);