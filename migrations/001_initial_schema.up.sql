-- 001_initial_schema.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(20) NOT NULL CHECK (role IN ('provider', 'client')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create trigger function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply trigger to users table
CREATE TRIGGER update_users_updated_at BEFORE UPDATE
ON users FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Create availability table
CREATE TABLE IF NOT EXISTS availability (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_id UUID NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_start_before_end CHECK (start_time < end_time),
    CONSTRAINT fk_provider FOREIGN KEY (provider_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Add unique constraint on (provider_id, start_time)
ALTER TABLE availability
ADD CONSTRAINT unique_provider_start_time UNIQUE (provider_id, start_time);

-- Apply trigger to availability table
CREATE TRIGGER update_availability_updated_at BEFORE UPDATE
ON availability FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Create appointments table
CREATE TABLE IF NOT EXISTS appointments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('reserved', 'confirmed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_appointment_start_before_end CHECK (start_time < end_time),
    CONSTRAINT fk_appointment_client FOREIGN KEY (client_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_appointment_provider FOREIGN KEY (provider_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Apply trigger to appointments table
CREATE TRIGGER update_appointments_updated_at BEFORE UPDATE
ON appointments FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();

-- Indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_availability_provider_start_time
ON availability (provider_id, start_time);

CREATE INDEX IF NOT EXISTS idx_appointments_provider_start_time
ON appointments (provider_id, start_time);

CREATE INDEX IF NOT EXISTS idx_appointments_status_created_at
ON appointments (status, created_at);
