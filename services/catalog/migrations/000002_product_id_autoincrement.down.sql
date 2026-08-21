-- Irreversible for existing rows: original TEXT ids were dropped by 000002 up
-- and cannot be recovered. Only safe to run against an empty table; otherwise
-- use `make db-reset`.
ALTER TABLE products DROP COLUMN id;
ALTER TABLE products ADD COLUMN id TEXT PRIMARY KEY;
