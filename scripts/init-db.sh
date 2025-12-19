#!/bin/bash

# Script to create adapter database in existing PostgreSQL instance

echo "Creating adapter database..."

# Connect to PostgreSQL and create adapter_db if it doesn't exist
PGPASSWORD=postgres psql -h localhost -U postgres -p 5432 << EOF
-- Create adapter database if not exists
SELECT 'CREATE DATABASE adapter_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'adapter_db')\gexec

-- Connect to adapter_db and run migrations
\c adapter_db

-- Run migrations
$(cat scripts/migrations/001_create_adapters_table.sql)

EOF

echo "Database initialization complete!"