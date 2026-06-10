CREATE TABLE IF NOT EXISTS scrapes (
    id SERIAL PRIMARY KEY,
    scraped_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    duration_ms INT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scrapes_scraped_at ON scrapes(scraped_at);
