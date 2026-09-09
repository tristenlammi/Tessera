-- 0079_book_authors: what the catalogue knows about a library author (photo, bio, key),
-- looked up lazily and kept, so the Books page's author tiles don't cost a request
-- per visit. checked_at drives a retry for authors that had nothing last time.
CREATE TABLE IF NOT EXISTS book_authors (
    name       TEXT PRIMARY KEY,
    key        TEXT NOT NULL DEFAULT '',
    image_url  TEXT NOT NULL DEFAULT '',
    bio        TEXT NOT NULL DEFAULT '',
    checked_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
