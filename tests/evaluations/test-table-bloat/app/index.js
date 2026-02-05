const { Pool } = require('pg');

const pool = new Pool({
  connectionString: process.env.DATABASE_URL,
});

async function waitForDatabase() {
  console.log('Waiting for database...');
  for (let i = 0; i < 30; i++) {
    try {
      await pool.query('SELECT 1');
      console.log('Database ready');
      return;
    } catch (err) {
      console.log('Database not ready, retrying...');
      await new Promise(r => setTimeout(r, 1000));
    }
  }
  throw new Error('Database connection timeout');
}

async function generateBloat() {
  console.log('Starting bloat generation cycle...');

  // Delete random rows (creates dead tuples)
  const deleteResult = await pool.query(`
    DELETE FROM posts 
    WHERE id IN (
      SELECT id FROM posts ORDER BY random() LIMIT 5000
    )
  `);
  console.log(`Deleted ${deleteResult.rowCount} rows`);

  // Re-insert to maintain table size
  const insertResult = await pool.query(`
    INSERT INTO posts (user_id, title, body)
    SELECT 
      (random() * 1000)::int,
      'Regenerated Post ' || generate_series,
      'Regenerated post body'
    FROM generate_series(1, 5000)
  `);
  console.log(`Inserted ${insertResult.rowCount} rows`);

  // Check current bloat
  const stats = await pool.query(`
    SELECT relname, n_live_tup, n_dead_tup,
           CASE WHEN n_live_tup > 0 
                THEN round((n_dead_tup::numeric / n_live_tup) * 100, 2)
                ELSE 0 
           END as bloat_percent
    FROM pg_stat_user_tables
    WHERE relname = 'posts'
  `);
  
  if (stats.rows[0]) {
    const { n_live_tup, n_dead_tup, bloat_percent } = stats.rows[0];
    console.log(`Current stats: ${n_live_tup} live, ${n_dead_tup} dead, ${bloat_percent}% bloat`);
  }
}

async function main() {
  await waitForDatabase();
  
  console.log('Starting bloat generator...');
  
  // Run bloat generation every 30 seconds
  while (true) {
    try {
      await generateBloat();
    } catch (err) {
      console.error('Error generating bloat:', err.message);
    }
    await new Promise(r => setTimeout(r, 30000));
  }
}

main().catch(console.error);