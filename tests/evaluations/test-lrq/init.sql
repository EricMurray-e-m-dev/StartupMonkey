-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create large table for slow queries
CREATE TABLE large_table (
    id SERIAL PRIMARY KEY,
    data TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert lots of data
INSERT INTO large_table (data)
SELECT md5(random()::text)
FROM generate_series(1, 500000);

-- No indexes on data column - queries will be slow