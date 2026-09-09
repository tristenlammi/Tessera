-- 0078_book_series_key: the catalogue's own id for a book's series (Hardcover "hc:s:<id>"),
-- so the series page can ask the catalogue for every entry rather than inferring gaps
-- from what the library happens to hold. Empty for series learned from release names.
ALTER TABLE books ADD COLUMN series_key TEXT NOT NULL DEFAULT '';
