-- Destructive: drops and regenerates all product ids; run make db-reset locally.
ALTER TABLE products DROP COLUMN id;
ALTER TABLE products ADD COLUMN id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY;
