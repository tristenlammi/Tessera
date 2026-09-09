-- 0080_metadata_cache: catalogue answers (TMDB browse rows, Hardcover lists and books)
-- kept across restarts. The in-memory caches emptied on every deploy, so the first
-- Discover open after an update waited on a dozen upstream requests — and Hardcover
-- paces at one a second. Rows past their TTL are still served while a refresh runs.
CREATE TABLE IF NOT EXISTS metadata_cache (
    key        TEXT PRIMARY KEY,
    value      BLOB NOT NULL,
    stored_at  INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_metadata_cache_stored ON metadata_cache(stored_at);
