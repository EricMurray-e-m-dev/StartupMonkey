const express = require('express');
const cors = require('cors');
const helmet = require('helmet');
const morgan = require('morgan');
const db = require('./database');

const app = express();
const PORT = process.env.PORT || 3000;

app.use(helmet());
app.use(cors());
app.use(morgan('combined'));
app.use(express.json());

// Health check endpoint
app.get('/health', (req, res) => {
  res.json({ status: 'ok', timestamp: new Date().toISOString() });
});

// 1. GET /api/posts - Queries without index on author_id (Performance Issue #1)
app.get('/api/posts', async (req, res) => {
  try {
    const { author_id, category, limit = 50, offset = 0 } = req.query;

    let query = 'SELECT p.*, u.username as author_name FROM posts p JOIN users u ON p.author_id = u.id WHERE p.published = true';
    const params = [];
    let paramCount = 0;

    // This will cause a full table scan on posts table due to missing index on author_id
    if (author_id) {
      paramCount++;
      query += ` AND p.author_id = $${paramCount}`;
      params.push(author_id);
    }

    // This will cause a full table scan due to missing index on category
    if (category) {
      paramCount++;
      query += ` AND p.category = $${paramCount}`;
      params.push(category);
    }

    // Order by created_at without index - another performance issue
    query += ` ORDER BY p.created_at DESC`;

    paramCount++;
    query += ` LIMIT $${paramCount}`;
    params.push(limit);

    paramCount++;
    query += ` OFFSET $${paramCount}`;
    params.push(offset);

    const result = await db.query(query, params);

    res.json({
      posts: result.rows,
      total: result.rows.length,
      limit: parseInt(limit),
      offset: parseInt(offset)
    });
  } catch (err) {
    console.error('Error in /api/posts:', err);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// 2. GET /api/heavy-query - Slow join between posts and comments (Performance Issue #2)
app.get('/api/heavy-query', async (req, res) => {
  try {
    const { limit = 20 } = req.query;

    // This query is intentionally expensive:
    // - Joins posts and comments without proper indexes
    // - Counts comments for each post (N+1 style query)
    // - Orders by multiple unindexed columns
    // - Uses subqueries that scan large tables
    const query = `
      SELECT
        p.id,
        p.title,
        p.content,
        p.author_id,
        u.username as author_name,
        p.created_at,
        p.view_count,
        p.like_count,
        (
          SELECT COUNT(*)
          FROM comments c
          WHERE c.post_id = p.id
        ) as comment_count,
        (
          SELECT AVG(c.like_count)
          FROM comments c
          WHERE c.post_id = p.id
        ) as avg_comment_likes,
        (
          SELECT string_agg(DISTINCT cu.username, ', ')
          FROM comments c
          JOIN users cu ON c.author_id = cu.id
          WHERE c.post_id = p.id
          LIMIT 5
        ) as recent_commenters
      FROM posts p
      JOIN users u ON p.author_id = u.id
      LEFT JOIN comments c ON p.id = c.post_id
      WHERE p.published = true
      GROUP BY p.id, u.username
      HAVING COUNT(c.id) > 0
      ORDER BY p.view_count DESC, p.created_at DESC, COUNT(c.id) DESC
      LIMIT $1
    `;

    const result = await db.query(query, [limit]);

    res.json({
      posts: result.rows,
      query_info: {
        description: 'Heavy query with multiple joins and subqueries',
        performance_issues: [
          'Missing indexes on foreign keys',
          'Expensive subqueries in SELECT',
          'Multiple JOINs without proper indexing',
          'ORDER BY on unindexed columns',
          'String aggregation in subquery'
        ]
      }
    });
  } catch (err) {
    console.error('Error in /api/heavy-query:', err);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// 3. GET /api/random-posts - Reads random data to cause cache misses (Performance Issue #3)
app.get('/api/random-posts', async (req, res) => {
  try {
    const { count = 10 } = req.query;

    // This query causes cache misses by:
    // - Using RANDOM() which defeats query caching
    // - Accessing random rows across the entire table
    // - Forcing full table scans
    // - Making result set unpredictable for database buffer pool
    const query = `
      WITH random_posts AS (
        SELECT p.id
        FROM posts p
        WHERE p.published = true
        AND random() < 0.1
        ORDER BY random()
        LIMIT $1 * 3
      )
      SELECT
        p.*,
        u.username as author_name,
        u.avatar_url,
        (
          SELECT COUNT(*)
          FROM comments c
          WHERE c.post_id = p.id
        ) as comment_count,
        (
          SELECT json_agg(
            json_build_object(
              'id', c.id,
              'content', LEFT(c.content, 100),
              'author', cu.username,
              'created_at', c.created_at
            )
          )
          FROM comments c
          JOIN users cu ON c.author_id = cu.id
          WHERE c.post_id = p.id
          ORDER BY c.created_at DESC
          LIMIT 3
        ) as recent_comments
      FROM random_posts rp
      JOIN posts p ON rp.id = p.id
      JOIN users u ON p.author_id = u.id
      ORDER BY random()
      LIMIT $1
    `;

    const result = await db.query(query, [count]);

    res.json({
      posts: result.rows,
      cache_info: {
        description: 'Random posts query designed to cause cache misses',
        performance_issues: [
          'RANDOM() prevents query plan caching',
          'Random row access defeats buffer pool efficiency',
          'Unpredictable access patterns',
          'Multiple random operations per request'
        ]
      }
    });
  } catch (err) {
    console.error('Error in /api/random-posts:', err);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// 4. GET /api/search - Unindexed ILIKE text search (Performance Issue #4)
app.get('/api/search', async (req, res) => {
  try {
    const { q, limit = 25, offset = 0 } = req.query;

    if (!q || q.trim().length < 2) {
      return res.status(400).json({ error: 'Search query must be at least 2 characters' });
    }

    const searchTerm = `%${q.toLowerCase()}%`;

    // This query is intentionally slow:
    // - Uses ILIKE (case-insensitive LIKE) without full-text search indexes
    // - Searches across multiple text columns
    // - No GIN or GiST indexes for text search
    // - Forces sequential scans on large text columns
    const query = `
      SELECT
        p.id,
        p.title,
        p.summary,
        LEFT(p.content, 200) as content_preview,
        p.author_id,
        u.username as author_name,
        p.category,
        p.tags,
        p.created_at,
        p.view_count,
        p.like_count,
        ts_rank(
          to_tsvector('english', p.title || ' ' || p.content),
          plainto_tsquery('english', $1)
        ) as relevance_score
      FROM posts p
      JOIN users u ON p.author_id = u.id
      WHERE p.published = true
        AND (
          LOWER(p.title) ILIKE $2
          OR LOWER(p.content) ILIKE $2
          OR LOWER(p.category) ILIKE $2
          OR EXISTS (
            SELECT 1 FROM unnest(p.tags) as tag
            WHERE LOWER(tag) ILIKE $2
          )
          OR LOWER(u.username) ILIKE $2
        )
      ORDER BY
        CASE
          WHEN LOWER(p.title) ILIKE $2 THEN 1
          WHEN LOWER(p.category) ILIKE $2 THEN 2
          ELSE 3
        END,
        p.view_count DESC,
        p.created_at DESC
      LIMIT $3 OFFSET $4
    `;

    const result = await db.query(query, [q, searchTerm, limit, offset]);

    res.json({
      results: result.rows,
      search_term: q,
      total: result.rows.length,
      limit: parseInt(limit),
      offset: parseInt(offset),
      performance_info: {
        description: 'Text search without proper indexing',
        performance_issues: [
          'ILIKE operations on large text columns',
          'No full-text search indexes (GIN)',
          'Sequential scans on posts table',
          'Multiple ILIKE operations per row',
          'Complex ORDER BY without covering indexes'
        ]
      }
    });
  } catch (err) {
    console.error('Error in /api/search:', err);
    res.status(500).json({ error: 'Internal server error' });
  }
});

// Basic frontend for testing
app.get('/', (req, res) => {
  res.send(`
    <!DOCTYPE html>
    <html>
    <head>
        <title>Startup Blog API</title>
        <style>
            body { font-family: Arial, sans-serif; margin: 40px; }
            .endpoint { margin: 20px 0; padding: 10px; border: 1px solid #ccc; }
            .method { color: #2196F3; font-weight: bold; }
            .path { color: #4CAF50; font-weight: bold; }
            .description { color: #666; margin: 5px 0; }
            .issue { color: #f44336; font-style: italic; }
        </style>
    </head>
    <body>
        <h1>Startup Blog API - Performance Testing Endpoints</h1>
        <p>This API simulates common database performance issues found in startup applications.</p>

        <div class="endpoint">
            <div><span class="method">GET</span> <span class="path">/api/posts</span></div>
            <div class="description">Fetch blog posts with filtering</div>
            <div class="issue">Issue: Missing indexes on author_id and category columns</div>
        </div>

        <div class="endpoint">
            <div><span class="method">GET</span> <span class="path">/api/heavy-query</span></div>
            <div class="description">Complex query with multiple joins and subqueries</div>
            <div class="issue">Issue: Expensive JOINs without proper indexing</div>
        </div>

        <div class="endpoint">
            <div><span class="method">GET</span> <span class="path">/api/random-posts</span></div>
            <div class="description">Fetch random posts (cache miss generator)</div>
            <div class="issue">Issue: Random access patterns defeat database caching</div>
        </div>

        <div class="endpoint">
            <div><span class="method">GET</span> <span class="path">/api/search?q=keyword</span></div>
            <div class="description">Text search across posts</div>
            <div class="issue">Issue: ILIKE operations without full-text search indexes</div>
        </div>

        <h2>Example Requests:</h2>
        <ul>
            <li><a href="/api/posts?limit=10">/api/posts?limit=10</a></li>
            <li><a href="/api/posts?author_id=1&category=Technology">/api/posts?author_id=1&category=Technology</a></li>
            <li><a href="/api/heavy-query?limit=5">/api/heavy-query?limit=5</a></li>
            <li><a href="/api/random-posts?count=5">/api/random-posts?count=5</a></li>
            <li><a href="/api/search?q=startup">/api/search?q=startup</a></li>
        </ul>
    </body>
    </html>
  `);
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Server running on port ${PORT}`);
  console.log(`Visit http://localhost:${PORT} for API documentation`);
});