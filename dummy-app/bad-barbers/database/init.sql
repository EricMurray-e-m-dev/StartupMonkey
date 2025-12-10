-- BarberBook Database Schema
-- Intentionally missing indexes for StartupMonkey to detect

-- Enable pg_stat_statements extension (required for StartupMonkey)
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Configure pg_stat_statements (optional but useful)
ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
ALTER SYSTEM SET pg_stat_statements.track = 'all';

-- Shops table
CREATE TABLE shops (
    shop_id SERIAL PRIMARY KEY,
    shop_name VARCHAR(100) NOT NULL,
    address TEXT,
    phone VARCHAR(20),
    opening_time TIME DEFAULT '09:00',
    closing_time TIME DEFAULT '18:00'
);

-- Barbers table (NO INDEX on shop_id - will cause slow joins)
CREATE TABLE barbers (
    barber_id SERIAL PRIMARY KEY,
    shop_id INT NOT NULL,  -- NO INDEX! Frequently joined
    barber_name VARCHAR(100) NOT NULL,
    speciality VARCHAR(50),
    experience_years INT,
    rating DECIMAL(2,1) DEFAULT 5.0
);

-- Bookings table (NO INDEXES on frequently queried columns)
CREATE TABLE bookings (
    booking_id SERIAL PRIMARY KEY,
    customer_name VARCHAR(100) NOT NULL,
    customer_phone VARCHAR(20) NOT NULL,  -- NO INDEX! Frequently searched
    customer_email VARCHAR(100),
    barber_id INT NOT NULL,
    booking_date DATE NOT NULL,  -- NO INDEX! Frequently filtered
    booking_time TIME NOT NULL,
    service VARCHAR(50) NOT NULL,
    price DECIMAL(6,2),
    status VARCHAR(20) DEFAULT 'confirmed',  -- confirmed, completed, cancelled
    notes TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Services table
CREATE TABLE services (
    service_id SERIAL PRIMARY KEY,
    service_name VARCHAR(100) NOT NULL,
    duration_minutes INT NOT NULL,
    price DECIMAL(6,2) NOT NULL,
    description TEXT
);

-- Insert sample shop
INSERT INTO shops (shop_name, address, phone) VALUES
('Classic Cuts Barbershop', '123 Main Street, Dublin', '+353 1 234 5678');

-- Insert sample barbers
INSERT INTO barbers (shop_id, barber_name, speciality, experience_years, rating) VALUES
(1, 'Tony Romano', 'Traditional cuts', 15, 4.9),
(1, 'Mike Sullivan', 'Fades & Styling', 8, 4.8),
(1, 'James Murphy', 'Beard grooming', 12, 4.7),
(1, 'Liam OConnor', 'Modern styles', 5, 4.6);

-- Insert sample services
INSERT INTO services (service_name, duration_minutes, price, description) VALUES
('Haircut', 30, 25.00, 'Standard haircut'),
('Haircut & Beard Trim', 45, 35.00, 'Haircut with beard styling'),
('Beard Trim', 20, 15.00, 'Beard shaping and trim'),
('Hot Towel Shave', 30, 30.00, 'Traditional hot towel shave'),
('Kids Haircut', 20, 18.00, 'Haircut for children under 12');

-- Generate realistic booking data (50,000 records to make sequential scans painful)
-- Bookings over the past 6 months
INSERT INTO bookings (customer_name, customer_phone, customer_email, barber_id, booking_date, booking_time, service, price, status)
SELECT 
    'Customer ' || generate_series AS customer_name,
    '+353 ' || LPAD((RANDOM() * 900000000 + 100000000)::BIGINT::TEXT, 9, '0') AS customer_phone,
    'customer' || generate_series || '@email.com' AS customer_email,
    (RANDOM() * 3 + 1)::INT AS barber_id,  -- Random barber (1-4)
    CURRENT_DATE - (RANDOM() * 180)::INT AS booking_date,  -- Last 6 months
    (TIME '09:00' + (RANDOM() * INTERVAL '9 hours')) AS booking_time,
    (ARRAY['Haircut', 'Haircut & Beard Trim', 'Beard Trim', 'Hot Towel Shave'])[FLOOR(RANDOM() * 4 + 1)] AS service,
    (RANDOM() * 30 + 15)::DECIMAL(6,2) AS price,
    (ARRAY['confirmed', 'completed', 'completed', 'completed'])[FLOOR(RANDOM() * 4 + 1)] AS status  -- Most completed
FROM generate_series(1, 50000);

-- Add some recent bookings for today (for demo purposes)
INSERT INTO bookings (customer_name, customer_phone, customer_email, barber_id, booking_date, booking_time, service, price, status)
VALUES 
('John Smith', '+353 87 123 4567', 'john@email.com', 1, CURRENT_DATE, '10:00', 'Haircut', 25.00, 'confirmed'),
('Mary Jones', '+353 86 234 5678', 'mary@email.com', 2, CURRENT_DATE, '10:30', 'Haircut & Beard Trim', 35.00, 'confirmed'),
('Pat Murphy', '+353 85 345 6789', 'pat@email.com', 1, CURRENT_DATE, '11:00', 'Beard Trim', 15.00, 'confirmed'),
('Sarah Kelly', '+353 87 456 7890', 'sarah@email.com', 3, CURRENT_DATE, '11:30', 'Haircut', 25.00, 'confirmed'),
('Tom Brady', '+353 86 567 8901', 'tom@email.com', 4, CURRENT_DATE, '12:00', 'Hot Towel Shave', 30.00, 'confirmed');

-- Create a view for popular barbers (will be queried frequently without caching)
CREATE VIEW popular_barbers AS
SELECT 
    b.barber_id,
    b.barber_name,
    b.speciality,
    b.rating,
    COUNT(bk.booking_id) as total_bookings
FROM barbers b
LEFT JOIN bookings bk ON b.barber_id = bk.barber_id
WHERE bk.status = 'completed'
GROUP BY b.barber_id, b.barber_name, b.speciality, b.rating
ORDER BY total_bookings DESC;