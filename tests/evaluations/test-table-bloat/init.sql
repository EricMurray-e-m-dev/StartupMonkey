-- Enable pg_stat_statements
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Create test table
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Insert initial data
INSERT INTO posts (user_id, title, body)
SELECT 
    (random() * 1000)::int,
    'Post ' || generate_series,
    'This is the body of post ' || generate_series
FROM generate_series(1, 100000);

-- Create index to avoid missing_index detection
CREATE INDEX idx_posts_user_id ON posts(user_id);