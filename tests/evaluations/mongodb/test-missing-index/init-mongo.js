v// Switch to testdb
db = db.getSiblingDB('testdb');

// Create application user
db.createUser({
  user: 'testuser',
  pwd: 'testpass',
  roles: [
    { role: 'readWrite', db: 'testdb' },
    { role: 'dbAdmin', db: 'testdb' }
  ]
});

// Create posts collection with 100k documents (no index on user_id)
print('Creating posts collection with 100k documents...');

const batch = [];
for (let i = 0; i < 100000; i++) {
  batch.push({
    user_id: Math.floor(Math.random() * 1000) + 1,
    title: `Post Title ${i}`,
    content: `This is the content for post number ${i}`,
    created_at: new Date()
  });
  
  if (batch.length === 1000) {
    db.posts.insertMany(batch);
    batch.length = 0;
  }
}

if (batch.length > 0) {
  db.posts.insertMany(batch);
}

print('Created 100k documents in posts collection');
print('No index on user_id - queries will use COLLSCAN');