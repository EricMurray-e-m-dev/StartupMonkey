#!/bin/bash

echo "🔧 Setting up Dummy App environment..."

# Check if venv exists
if [ ! -d "venv" ]; then
    echo "Creating Python virtual environment..."
    python3 -m venv venv
fi

# Activate venv
source venv/bin/activate

# Install Locust if not installed
if ! command -v locust &> /dev/null; then
    echo "Installing Locust..."
    pip install locust
fi

# Install Node dependencies
if [ ! -d "node_modules" ]; then
    echo "Installing Node.js dependencies..."
    npm install
fi

echo "✅ Setup complete!"
echo ""
echo "To run the dummy app:"
echo "  1. Initialize database: psql -U postgres -d dummy_app -f init-db.sql"
echo "  2. Start app: npm start"
echo "  3. Run load test: source venv/bin/activate && locust -f locustfile.py"
echo ""npm