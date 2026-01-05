-- Enable pg_stat_statements for monitoring
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create posts table (NO index on user_id - this is intentional)
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Seed 100,000 rows
-- user_id will be random between 1-1000 (simulates 1000 users)
INSERT INTO posts (user_id, title, body, created_at)
SELECT 
    (random() * 999 + 1)::INTEGER as user_id,
    'Post title ' || generate_series as title,
    'This is the body content for post number ' || generate_series || '. It contains some text to make the row larger and more realistic.' as body,
    NOW() - (random() * INTERVAL '365 days') as created_at
FROM generate_series(1, 100000);

-- Analyze table for accurate stats
ANALYZE posts;

-- Verify row count
DO $$
BEGIN
    RAISE NOTICE 'Seeded % rows into posts table', (SELECT COUNT(*) FROM posts);
END $$;