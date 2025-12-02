require('dotenv').config();
const express = require('express');
const { Pool } = require('pg');

const app = express();
const PORT = process.env.PORT || 3002;

// Intentionally small connection pool (will exhaust easily)
const pool = new Pool({
    host: process.env.DB_HOST,
    port: process.env.DB_PORT,
    database: process.env.DB_NAME,
    user: process.env.DB_USER,
    password: process.env.DB_PASSWORD,
    min: parseInt(process.env.DB_POOL_MIN) || 2,
    max: parseInt(process.env.DB_POOL_MAX) || 10,
    idleTimeoutMillis: 30000,
    connectionTimeoutMillis: 5000,
});

// Middleware
app.use(express.json());

// Health check
app.get('/health', (req, res) => {
    res.json({ 
        status: 'ok',
        timestamp: new Date().toISOString() 
    });
});

// ===========================================
// BAD ENDPOINT #1: Missing Index (Sequential Scan)
// ===========================================
// This endpoint queries posts by user_id WITHOUT an index
// Guaranteed to trigger missing_index detector
app.get('/api/posts', async (req, res) => {
    const userId = req.query.user_id || Math.floor(Math.random() * 1000) + 1;
    
    try {
        const result = await pool.query(
            'SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50',
            [userId]
        );
        res.json({
            user_id: userId,
            count: result.rows.length,
            posts: result.rows
        });
    } catch (err) {
        console.error('Query error:', err);
        res.status(500).json({ error: 'Database error' });
    }
});

// ===========================================
// BAD ENDPOINT #2: Connection Leak
// ===========================================
// This endpoint acquires a connection and holds it for 30 seconds
// Guaranteed to trigger connection_pool_exhaustion
app.get('/api/leak-connection', async (req, res) => {
    try {
        const client = await pool.connect();
        
        console.log(`[LEAK] Connection acquired, holding for 30s...`);
        
        // Don't release the connection for 30 seconds!
        setTimeout(() => {
            client.release();
            console.log(`[LEAK] Connection released after 30s`);
        }, 30000);
        
        res.json({ 
            leaked: true,
            message: 'Connection will be held for 30 seconds' 
        });
    } catch (err) {
        console.error('[LEAK] Failed to acquire connection (pool exhausted!):', err.message);
        res.status(503).json({ 
            error: 'Connection pool exhausted',
            message: 'All connections are in use. This is expected during load testing.'
        });
    }
});

// ===========================================
// BAD ENDPOINT #3: Intentionally Slow Query
// ===========================================
// Forces a 2-second sleep to trigger high_latency detector
app.get('/api/slow-query', async (req, res) => {
    try {
        const sleepTime = req.query.sleep || 2;
        await pool.query(`SELECT pg_sleep(${sleepTime})`);
        res.json({ 
            slept: sleepTime,
            message: `Slept for ${sleepTime} seconds` 
        });
    } catch (err) {
        console.error('Slow query error:', err);
        res.status(500).json({ error: 'Database error' });
    }
});

// ===========================================
// BAD ENDPOINT #4: Expensive Aggregation (No Index)
// ===========================================
// Does aggregation across entire posts table without index
app.get('/api/stats', async (req, res) => {
    try {
        const result = await pool.query(`
            SELECT 
                user_id,
                COUNT(*) as post_count,
                SUM(view_count) as total_views,
                AVG(view_count) as avg_views
            FROM posts
            GROUP BY user_id
            ORDER BY total_views DESC
            LIMIT 10
        `);
        res.json({
            top_users: result.rows
        });
    } catch (err) {
        console.error('Stats error:', err);
        res.status(500).json({ error: 'Database error' });
    }
});

// ===========================================
// GOOD ENDPOINT: Simple query (for comparison)
// ===========================================
app.get('/api/users/:id', async (req, res) => {
    try {
        const result = await pool.query(
            'SELECT * FROM users WHERE id = $1',
            [req.params.id]
        );
        
        if (result.rows.length === 0) {
            return res.status(404).json({ error: 'User not found' });
        }
        
        res.json(result.rows[0]);
    } catch (err) {
        console.error('User query error:', err);
        res.status(500).json({ error: 'Database error' });
    }
});

// Start server
app.listen(PORT, () => {
    console.log(`🚀 Dummy App running on http://localhost:${PORT}`);
    console.log(`📊 Endpoints:`);
    console.log(`   GET /health - Health check`);
    console.log(`   GET /api/posts?user_id=X - Missing index (triggers detection)`);
    console.log(`   GET /api/leak-connection - Leaks connection (triggers detection)`);
    console.log(`   GET /api/slow-query?sleep=X - Slow query (triggers detection)`);
    console.log(`   GET /api/stats - Expensive aggregation`);
    console.log(`   GET /api/users/:id - Normal query`);
    console.log(`\n⚠️  This app is INTENTIONALLY BAD for testing purposes!`);
});

// Graceful shutdown
process.on('SIGTERM', async () => {
    console.log('SIGTERM received, closing pool...');
    await pool.end();
    process.exit(0);
});