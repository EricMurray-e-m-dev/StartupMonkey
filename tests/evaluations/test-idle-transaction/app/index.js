const { Client } = require('pg');

async function waitForDatabase() {
  const client = new Client({ connectionString: process.env.DATABASE_URL });
  console.log('Waiting for database...');
  
  for (let i = 0; i < 30; i++) {
    try {
      await client.connect();
      await client.query('SELECT 1');
      await client.end();
      console.log('Database ready');
      return;
    } catch (err) {
      console.log('Database not ready, retrying...');
      await new Promise(r => setTimeout(r, 1000));
    }
  }
  throw new Error('Database connection timeout');
}

async function createIdleTransaction() {
  // Use Client not Pool - we want to keep this specific connection open
  const client = new Client({ connectionString: process.env.DATABASE_URL });
  
  try {
    await client.connect();
    console.log('Connected to database');
    
    // Start transaction
    await client.query('BEGIN');
    console.log('Transaction started');
    
    // Run a query
    const result = await client.query('SELECT * FROM users LIMIT 10');
    console.log(`Query executed, returned ${result.rowCount} rows`);
    
    // Intentionally do NOT commit or rollback
    // This leaves the connection in "idle in transaction" state
    console.log('Leaving transaction open (idle in transaction)...');
    console.log('Connection will remain idle until terminated by StartupMonkey');
    
    // Keep the connection alive
    while (true) {
      await new Promise(r => setTimeout(r, 60000));
      console.log('Still idle in transaction...');
    }
  } catch (err) {
    if (err.message.includes('terminate') || err.message.includes('cancel') || err.message.includes('Connection terminated')) {
      console.log('Connection was terminated (expected behaviour from StartupMonkey)');
      // Restart after being terminated
      return true;
    }
    console.error('Error:', err.message);
    return false;
  }
}

async function main() {
  await waitForDatabase();
  
  console.log('Starting idle transaction generator...');
  console.log('Will create a transaction and leave it idle');
  
  // Wait a bit for system to stabilise
  await new Promise(r => setTimeout(r, 5000));
  
  while (true) {
    const wasTerminated = await createIdleTransaction();
    
    if (wasTerminated) {
      console.log('Restarting idle transaction in 30 seconds...');
      await new Promise(r => setTimeout(r, 30000));
    } else {
      console.log('Retrying in 10 seconds...');
      await new Promise(r => setTimeout(r, 10000));
    }
  }
}

main().catch(console.error);