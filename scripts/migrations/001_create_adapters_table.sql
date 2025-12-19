-- Drop existing tables if they exist
DROP TABLE IF EXISTS adapters CASCADE;

-- Create adapters table
CREATE TABLE IF NOT EXISTS adapters (
    id VARCHAR(40) PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    url VARCHAR(500) NOT NULL,
    callback_url VARCHAR(500) NOT NULL,
    api_key VARCHAR(100) NOT NULL UNIQUE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index on api_key for faster authentication
CREATE INDEX IF NOT EXISTS idx_adapters_api_key ON adapters(api_key);

-- Create index on is_active for filtering active adapters
CREATE INDEX IF NOT EXISTS idx_adapters_is_active ON adapters(is_active);

-- Create index on name for searching
CREATE INDEX IF NOT EXISTS idx_adapters_name ON adapters(name);

-- Create trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_adapters_updated_at ON adapters;
CREATE TRIGGER update_adapters_updated_at BEFORE UPDATE
    ON adapters FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();