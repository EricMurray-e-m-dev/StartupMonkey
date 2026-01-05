const express = require('express');
const { Pool } = require('pg');

const app = express();
const port = 4001;

// Pool with size close to postgres max_connections (30)
// Under load with slow queries, this will saturate
const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
  max: 25,  // Close to postgres max of 30
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

// SLOW ENDPOINT - holds connection for 500ms
// Under load, connections accumulate and hit the threshold
app.get('/slow', async (req, res) => {
  try {
    // pg_sleep holds the connection for 0.5 seconds
    const result = await pool.query(`
      SELECT pg_sleep(0.5), id, name 
      FROM items 
      ORDER BY random() 
      LIMIT 1
    `);
    res.json({
      item: result.rows[0],
      held_for: '500ms'
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// VERY SLOW ENDPOINT - for more aggressive testing
app.get('/very-slow', async (req, res) => {
  try {
    // pg_sleep holds the connection for 2 seconds
    const result = await pool.query(`
      SELECT pg_sleep(2), id, name 
      FROM items 
      ORDER BY random() 
      LIMIT 1
    `);
    res.json({
      item: result.rows[0],
      held_for: '2000ms'
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Fast endpoint for comparison
app.get('/fast', async (req, res) => {
  try {
    const result = await pool.query('SELECT id, name FROM items ORDER BY random() LIMIT 1');
    res.json({ item: result.rows[0] });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

// Stats endpoint - shows connection state
app.get('/stats', async (req, res) => {
  try {
    // Active connections
    const connections = await pool.query(`
      SELECT count(*) as active_connections,
             (SELECT setting::int FROM pg_settings WHERE name = 'max_connections') as max_connections
      FROM pg_stat_activity 
      WHERE state = 'active' OR state = 'idle'
    `);
    
    // Pool stats
    res.json({
      postgres: connections.rows[0],
      pool: {
        total: pool.totalCount,
        idle: pool.idleCount,
        waiting: pool.waitingCount
      },
      usage_ratio: (connections.rows[0].active_connections / connections.rows[0].max_connections).toFixed(2)
    });
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.listen(port, () => {
  console.log(`Connection Pool Test API running on port ${port}`);
  console.log('Endpoints:');
  console.log('  GET /health     - Health check');
  console.log('  GET /slow       - Slow query (500ms hold)');
  console.log('  GET /very-slow  - Very slow query (2s hold)');
  console.log('  GET /fast       - Fast query (baseline)');
  console.log('  GET /stats      - Connection statistics');
});