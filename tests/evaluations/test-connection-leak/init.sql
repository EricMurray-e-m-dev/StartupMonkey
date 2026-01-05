-- Enable pg_stat_statements for monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Simple table for queries
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Seed some data
INSERT INTO items (name)
SELECT 'Item ' || generate_series
FROM generate_series(1, 1000);

ANALYZE items;