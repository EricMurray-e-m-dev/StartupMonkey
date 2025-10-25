-- Blog API Schema with intentional performance issues
-- Missing indexes on foreign keys and search columns

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    full_name VARCHAR(100) NOT NULL,
    bio TEXT,
    avatar_url VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Posts table (missing index on author_id - performance issue #1)
CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    summary TEXT,
    author_id INTEGER REFERENCES users(id),
    category VARCHAR(50),
    tags TEXT[],
    view_count INTEGER DEFAULT 0,
    like_count INTEGER DEFAULT 0,
    published BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Comments table (missing indexes on post_id and author_id - performance issue #2)
CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    uuid UUID DEFAULT uuid_generate_v4() UNIQUE,
    content TEXT NOT NULL,
    post_id INTEGER REFERENCES posts(id),
    author_id INTEGER REFERENCES users(id),
    parent_comment_id INTEGER REFERENCES comments(id),
    like_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Only create indexes on primary keys (automatic) and unique constraints
-- Deliberately missing indexes on:
-- - posts.author_id (causes slow queries when filtering by author)
-- - posts.title, posts.content (causes slow text searches)
-- - posts.category (causes slow category filtering)
-- - comments.post_id (causes slow joins between posts and comments)
-- - comments.author_id (causes slow author lookups)
-- - created_at columns (causes slow date range queries)

-- Add some basic indexes that would exist in a real app
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);

-- Note: Missing these critical indexes that should exist:
-- CREATE INDEX idx_posts_author_id ON posts(author_id);
-- CREATE INDEX idx_posts_created_at ON posts(created_at);
-- CREATE INDEX idx_posts_category ON posts(category);
-- CREATE INDEX idx_comments_post_id ON comments(post_id);
-- CREATE INDEX idx_comments_author_id ON comments(author_id);
-- CREATE INDEX idx_comments_created_at ON comments(created_at);
-- CREATE INDEX idx_posts_title_gin ON posts USING GIN(to_tsvector('english', title));
-- CREATE INDEX idx_posts_content_gin ON posts USING GIN(to_tsvector('english', content));