// BarberBook Backend - AI Generated Code - File contains intentional flaws for benchmarking & testing

const express = require('express');
const cors = require('cors');
const { Client } = require('pg');  // INTENTIONAL FLAW: Using Client instead of Pool
require('dotenv').config();

const app = express();
const PORT = process.env.PORT || 3001;

// Middleware
app.use(cors());
app.use(express.json());
app.use(express.static('public'));

// INTENTIONAL FLAW #1: Creates new connection for EVERY request
function getDbClient() {
  return new Client({
    host: process.env.DB_HOST || 'localhost',
    port: process.env.DB_PORT || 5432,
    database: process.env.DB_NAME || 'barberbook',
    user: process.env.DB_USER || 'postgres',
    password: process.env.DB_PASSWORD || 'postgres',
  });
}

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'barberbook-backend' });
});

// ENDPOINT 1: Get all barbers
app.get('/api/barbers', async (req, res) => {
  const client = getDbClient();
  
  try {
    await client.connect();
    
    const result = await client.query(`
      SELECT barber_id, barber_name, speciality, rating 
      FROM barbers 
      ORDER BY barber_name
    `);
    
    res.json(result.rows);
  } catch (err) {
    console.error('Error fetching barbers:', err);
    res.status(500).json({ error: 'Failed to fetch barbers' });
  }
  // INTENTIONAL FLAW: Sometimes forgets to close connection
});

// ENDPOINT 2: Get popular barbers (homepage)
app.get('/api/popular-barbers', async (req, res) => {
  const client = getDbClient();
  
  try {
    await client.connect();
    
    // INTENTIONAL FLAW: This view does COUNT(*) and GROUP BY on 50k records
    const result = await client.query(`
      SELECT * FROM popular_barbers LIMIT 5
    `);
    
    res.json(result.rows);
  } catch (err) {
    console.error('Error fetching popular barbers:', err);
    res.status(500).json({ error: 'Failed to fetch popular barbers' });
  } finally {
    await client.end();
  }
});

// ENDPOINT 3: Search bookings by phone
// INTENTIONAL FLAW: No index on customer_phone = sequential scan on 50k records
app.get('/api/bookings/search', async (req, res) => {
  const { phone } = req.query;
  
  if (!phone) {
    return res.status(400).json({ error: 'Phone number required' });
  }
  
  const client = getDbClient();
  
  try {
    await client.connect();
    
    // INTENTIONAL FLAW: Full table scan on 50k records
    const result = await client.query(
      'SELECT * FROM bookings WHERE customer_phone = $1 ORDER BY booking_date DESC',
      [phone]
    );
    
    res.json(result.rows);
  } catch (err) {
    console.error('Error searching bookings:', err);
    res.status(500).json({ error: 'Failed to search bookings' });
  } finally {
    await client.end();
  }
});

// ENDPOINT 4: Get today's bookings
// INTENTIONAL FLAW: No index on booking_date = sequential scan
app.get('/api/bookings/today', async (req, res) => {
  const client = getDbClient();
  
  try {
    await client.connect();
    
    // INTENTIONAL FLAW: Sequential scan on booking_date
    const result = await client.query(`
      SELECT 
        b.booking_id,
        b.customer_name,
        b.customer_phone,
        b.booking_time,
        b.service,
        b.status,
        br.barber_name
      FROM bookings b
      JOIN barbers br ON b.barber_id = br.barber_id
      WHERE b.booking_date = CURRENT_DATE
      ORDER BY b.booking_time
    `);
    
    res.json(result.rows);
  } catch (err) {
    console.error('Error fetching today bookings:', err);
    res.status(500).json({ error: 'Failed to fetch bookings' });
  } finally {
    await client.end();
  }
});

// ENDPOINT 5: Get barber's bookings
// INTENTIONAL FLAW: Loads ALL bookings then filters (instead of WHERE clause)
app.get('/api/barbers/:id/bookings', async (req, res) => {
  const { id } = req.params;
  const client = getDbClient();
  
  try {
    await client.connect();
    
    // INTENTIONAL FLAW: AI suggested "just load everything and filter in JavaScript!"
    const result = await client.query('SELECT * FROM bookings ORDER BY booking_date DESC');
    
    // Filter in application code instead of database (TERRIBLE!)
    const filtered = result.rows.filter(b => b.barber_id === parseInt(id));
    
    res.json(filtered);
  } catch (err) {
    console.error('Error fetching barber bookings:', err);
    res.status(500).json({ error: 'Failed to fetch bookings' });
  } finally {
    await client.end();
  }
});

// ENDPOINT 6: Create new booking
// INTENTIONAL FLAW: Creates connection but sometimes doesn't close it
app.post('/api/bookings', async (req, res) => {
  const { customer_name, customer_phone, customer_email, barber_id, booking_date, booking_time, service, price } = req.body;
  
  // Basic validation
  if (!customer_name || !customer_phone || !barber_id || !booking_date || !booking_time || !service) {
    return res.status(400).json({ error: 'Missing required fields' });
  }
  
  const client = getDbClient();
  
  try {
    await client.connect();
    
    const result = await client.query(
      `INSERT INTO bookings (customer_name, customer_phone, customer_email, barber_id, booking_date, booking_time, service, price, status)
       VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'confirmed')
       RETURNING *`,
      [customer_name, customer_phone, customer_email, barber_id, booking_date, booking_time, service, price || 25.00]
    );
    
    res.status(201).json(result.rows[0]);
  } catch (err) {
    console.error('Error creating booking:', err);
    res.status(500).json({ error: 'Failed to create booking' });
  }
  // INTENTIONAL FLAW: Forgot to close connection on success path
  // Only closes on error
});

// ENDPOINT 7: Get available time slots for a barber
// INTENTIONAL FLAW: Inefficient nested queries
app.get('/api/barbers/:id/available-slots', async (req, res) => {
  const { id } = req.params;
  const { date } = req.query;
  
  if (!date) {
    return res.status(400).json({ error: 'Date required' });
  }
  
  const client = getDbClient();
  
  try {
    await client.connect();
    
    // INTENTIONAL FLAW: Could do this in a single query, but AI suggested nested queries
    const bookedSlots = await client.query(
      'SELECT booking_time FROM bookings WHERE barber_id = $1 AND booking_date = $2',
      [id, date]
    );
    
    // Generate all possible slots (9am - 6pm, 30min intervals)
    const allSlots = [];
    for (let hour = 9; hour < 18; hour++) {
      allSlots.push(`${hour.toString().padStart(2, '0')}:00:00`);
      allSlots.push(`${hour.toString().padStart(2, '0')}:30:00`);
    }
    
    // Filter out booked slots in JavaScript (instead of SQL)
    const booked = bookedSlots.rows.map(row => row.booking_time);
    const available = allSlots.filter(slot => !booked.includes(slot));
    
    res.json({ available_slots: available });
  } catch (err) {
    console.error('Error fetching available slots:', err);
    res.status(500).json({ error: 'Failed to fetch available slots' });
  } finally {
    await client.end();
  }
});

// ENDPOINT 8: Get services
app.get('/api/services', async (req, res) => {
  const client = getDbClient();
  
  try {
    await client.connect();
    
    const result = await client.query('SELECT * FROM services ORDER BY price');
    
    res.json(result.rows);
  } catch (err) {
    console.error('Error fetching services:', err);
    res.status(500).json({ error: 'Failed to fetch services' });
  } finally {
    await client.end();
  }
});

// Start server
app.listen(PORT, () => {
  console.log(`BarberBook backend running on http://localhost:${PORT}`);
  console.log(`Database: ${process.env.DB_HOST || 'localhost'}:${process.env.DB_PORT || 5432}/${process.env.DB_NAME || 'barberbook'}`);
});