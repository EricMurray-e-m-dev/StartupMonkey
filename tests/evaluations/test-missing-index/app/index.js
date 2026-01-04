const express = require('express');
const { Pool } = require('pg');

const app = express();
const port = 4000;

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  max: 20,
});

// Health check
app.get('/health', async (req, res) => {
  try {
    await pool.query('SELECT 1');
    res.json({ status: 'healthy' });
  } catch (err) {
    res.status(500).json({ status: 'unhealthy', error: err.message });
  }
});

// THE SLOW ENDPOINT - queries non-indexed user_id column
// This will cause sequential scans until StartupMonkey creates an index
app.get('/posts/user/:userId', async (req, res) => {
  const userId = parseInt(req.params.userId);
  
  try {
    const result = await pool.query(
      'SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC LIMIT 10',
      [userId]
    );
    res.json({
      user_id: userId,
      count: result.rows.length,
      posts: result.rows
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Stats endpoint - check if index exists
app.get('/stats', async (req, res) => {
  try {
    // Check for indexes on posts table
    const indexes = await pool.query(`
      SELECT indexname, indexdef 
      FROM pg_indexes 
      WHERE tablename = 'posts'
    `);
    
    // Get seq_scan vs idx_scan counts
    const scans = await pool.query(`
      SELECT relname, seq_scan, idx_scan 
      FROM pg_stat_user_tables 
      WHERE relname = 'posts'
    `);
    
    res.json({
      indexes: indexes.rows,
      scans: scans.rows[0] || {},
      has_user_id_index: indexes.rows.some(i => i.indexdef.includes('user_id'))
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Reset stats (useful between test runs)
app.post('/reset-stats', async (req, res) => {
  try {
    await pool.query('SELECT pg_stat_reset()');
    res.json({ message: 'Stats reset' });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.listen(port, () => {
  console.log(`Test API running on port ${port}`);
  console.log('Endpoints:');
  console.log('  GET  /health           - Health check');
  console.log('  GET  /posts/user/:id   - Get posts by user (SLOW - no index)');
  console.log('  GET  /stats            - Check indexes and scan counts');
  console.log('  POST /reset-stats      - Reset pg_stat counters');
});