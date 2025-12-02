-- Drop existing tables if any
DROP TABLE IF EXISTS posts CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Users table (small, will stay cached)
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Posts table - INTENTIONALLY BAD DESIGN
-- No indexes except primary key
-- Large dataset to force slow sequential scans
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,           -- NO INDEX
    title TEXT NOT NULL,
    content TEXT,
    view_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert 1000 users
INSERT INTO users (username, email)
SELECT 
    'user_' || generate_series,
    'user_' || generate_series || '@example.com'
FROM generate_series(1, 1000);

-- Insert 100,000 posts
-- Each user has ~100 posts
INSERT INTO posts (user_id, title, content, view_count)
SELECT 
    (random() * 999 + 1)::integer,                    -- Random user_id (1-1000)
    'Post Title ' || generate_series,
    repeat('This is post content. ', 50),             -- ~1KB per post
    (random() * 10000)::integer                       -- Random view count
FROM generate_series(1, 100000);

-- Analyze tables for accurate statistics
ANALYZE users;
ANALYZE posts;

-- Show table sizes
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables 
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Show current indexes (only primary keys should exist)
SELECT 
    tablename,
    indexname,
    indexdef
FROM pg_indexes
WHERE schemaname = 'public'
ORDER BY tablename, indexname;