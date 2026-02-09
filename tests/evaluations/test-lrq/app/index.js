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

async function runSlowQuery() {
  console.log('Starting slow query...');
  const start = Date.now();

  try {
    // This query will be slow - full table scan with no index, plus expensive operations
    const result = await pool.query(`SELECT pg_sleep(45), * FROM large_table LIMIT 1`);

    const duration = (Date.now() - start) / 1000;
    console.log(`Query completed in ${duration.toFixed(1)}s, returned ${result.rowCount} rows`);
  } catch (err) {
    const duration = (Date.now() - start) / 1000;
    if (err.message.includes('cancel') || err.message.includes('terminate')) {
      console.log(`Query was terminated after ${duration.toFixed(1)}s (expected behaviour)`);
    } else {
      console.error(`Query failed after ${duration.toFixed(1)}s:`, err.message);
    }
  }
}

async function main() {
  await waitForDatabase();

  console.log('Starting slow query generator...');
  console.log('Queries will run every 60 seconds');

  // Run first query after 10 seconds (let system stabilise)
  await new Promise(r => setTimeout(r, 10000));

  while (true) {
    await runSlowQuery();
    // Wait before next slow query
    await new Promise(r => setTimeout(r, 60000));
  }
}

main().catch(console.error);