CREATE TABLE products (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    price_minor_units BIGINT NOT NULL,
    currency CHAR(3) NOT NULL
);