-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create test table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    email VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert some data
INSERT INTO users (name, email)
SELECT 
    'User ' || generate_series,
    'user' || generate_series || '@example.com'
FROM generate_series(1, 1000);