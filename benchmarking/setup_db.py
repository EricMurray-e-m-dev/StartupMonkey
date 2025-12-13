#!/usr/bin/env python3
"""
Setup script for benchmarking database.
Creates database and initializes schema.
"""

import psycopg2
from psycopg2.extensions import ISOLATION_LEVEL_AUTOCOMMIT
import sys
import os
from dotenv import load_dotenv

load_dotenv('framework/.env')

DB_HOST = os.getenv('BENCHMARK_DB_HOST', 'localhost')
DB_PORT = int(os.getenv('BENCHMARK_DB_PORT', 5432))
DB_NAME = os.getenv('BENCHMARK_DB_NAME', 'startupmonkey_benchmarks')
DB_USER = os.getenv('BENCHMARK_DB_USER', 'postgres')
DB_PASSWORD = os.getenv('BENCHMARK_DB_PASSWORD', 'postgres')


def create_database():
    """Create the benchmarking database if it doesn't exist."""
    try:
        # Connect to default postgres database
        conn = psycopg2.connect(
            host=DB_HOST,
            port=DB_PORT,
            database='postgres',
            user=DB_USER,
            password=DB_PASSWORD
        )
        conn.set_isolation_level(ISOLATION_LEVEL_AUTOCOMMIT)
        
        with conn.cursor() as cur:
            # Check if database exists
            cur.execute(
                "SELECT 1 FROM pg_database WHERE datname = %s",
                (DB_NAME,)
            )
            exists = cur.fetchone()
            
            if not exists:
                print(f"Creating database: {DB_NAME}")
                cur.execute(f'CREATE DATABASE {DB_NAME}')
                print(f"Database {DB_NAME} created successfully")
            else:
                print(f"Database {DB_NAME} already exists")
        
        conn.close()
        return True
        
    except Exception as e:
        print(f"Error creating database: {e}")
        return False


def initialize_schema():
    """Initialize database schema."""
    try:
        from framework.database import BenchmarkDatabase
        
        print("Initializing database schema...")
        
        with BenchmarkDatabase() as db:
            db.initialize_schema()
        
        print("Schema initialized successfully")
        return True
        
    except Exception as e:
        print(f"Error initializing schema: {e}")
        return False


if __name__ == '__main__':
    print("=== Benchmarking Database Setup ===\n")
    print(f"Host: {DB_HOST}:{DB_PORT}")
    print(f"Database: {DB_NAME}")
    print(f"User: {DB_USER}\n")
    
    # Step 1: Create database
    if not create_database():
        print("\nFailed to create database")
        sys.exit(1)
    
    # Step 2: Initialize schema
    if not initialize_schema():
        print("\nFailed to initialize schema")
        sys.exit(1)
    
    print("\n=== Setup Complete ===")
    print(f"Benchmarking database is ready at: {DB_HOST}:{DB_PORT}/{DB_NAME}")