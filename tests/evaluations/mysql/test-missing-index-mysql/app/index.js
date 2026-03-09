const express = require('express');
const mysql = require('mysql2/promise');

const app = express();
const port = 4001;

let pool;

async function initPool() {
  pool = mysql.createPool({
    host: process.env.DATABASE_URL ? new URL(process.env.DATABASE_URL).hostname : 'mysql',
    user: 'testuser',
    password: 'testpass',
    database: 'testdb',
    waitForConnections: true,
    connectionLimit: 20,
    queueLimit: 0
  });
}

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
app.get('/posts/user/:userId', async (req, res) => {
  const userId = parseInt(req.params.userId);
  
  try {
    const [rows] = await pool.query(
      'SELECT * FROM posts WHERE user_id = ? ORDER BY created_at DESC LIMIT 10',
      [userId]
    );
    res.json({
      user_id: userId,
      count: rows.length,
      posts: rows
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Stats endpoint - check indexes and table scans
app.get('/stats', async (req, res) => {
  try {
    // Check for indexes on posts table
    const [indexes] = await pool.query(`
      SHOW INDEX FROM posts
    `);
    
    // Get table scan stats from performance_schema
    const [scans] = await pool.query(`
      SELECT 
        COUNT_READ,
        COUNT_WRITE,
        COUNT_FETCH
      FROM performance_schema.table_io_waits_summary_by_table
      WHERE OBJECT_SCHEMA = 'testdb' AND OBJECT_NAME = 'posts'
    `);
    
    res.json({
      indexes: indexes,
      scans: scans[0] || {},
      has_user_id_index: indexes.some(i => i.Column_name === 'user_id')
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

initPool().then(() => {
  app.listen(port, () => {
    console.log(`MySQL Test API running on port ${port}`);
    console.log('Endpoints:');
    console.log('  GET  /health           - Health check');
    console.log('  GET  /posts/user/:id   - Get posts by user (SLOW - no index)');
    console.log('  GET  /stats            - Check indexes and scan counts');
    
    console.log('Starting load generator...');
    
    // Self-generate load to trigger full table scans
    const generateLoad = async () => {
      const userId = Math.floor(Math.random() * 1000) + 1;
      try {
        await pool.query(
          'SELECT * FROM posts WHERE user_id = ? ORDER BY created_at DESC LIMIT 10',
          [userId]
        );
      } catch (err) {
        console.error('Load generator error:', err.message);
      }
    };

    setInterval(generateLoad, 100);
    console.log('Load generator started - 10 queries/sec');
  });
});