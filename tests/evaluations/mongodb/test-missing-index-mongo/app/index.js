const { MongoClient } = require('mongodb');
const express = require('express');

const app = express();
const uri = process.env.MONGODB_URI || 'mongodb://testuser:testpass@localhost:27017/testdb?authSource=testdb';

let db;

async function connect() {
  const client = new MongoClient(uri);
  await client.connect();
  db = client.db('testdb');
  console.log('Connected to MongoDB');
}

// API endpoint that queries without index (COLLSCAN)
app.get('/api/posts', async (req, res) => {
  try {
    const userId = Math.floor(Math.random() * 1000) + 1;
    const posts = await db.collection('posts')
      .find({ user_id: userId })
      .limit(10)
      .toArray();
    res.json(posts);
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
});

app.get('/health', (req, res) => {
  res.json({ status: 'ok' });
});

async function startLoadGenerator() {
  console.log('Starting load generator - 10 queries/sec on unindexed user_id');
  
  setInterval(async () => {
    try {
      const userId = Math.floor(Math.random() * 1000) + 1;
      await db.collection('posts').find({ user_id: userId }).limit(10).toArray();
    } catch (err) {
      console.error('Query error:', err.message);
    }
  }, 100); // 10 queries per second
}

connect()
  .then(() => {
    app.listen(3000, () => {
      console.log('Load generator API running on port 3000');
      startLoadGenerator();
    });
  })
  .catch(err => {
    console.error('Failed to connect:', err);
    process.exit(1);
  });