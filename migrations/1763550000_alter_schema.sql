-- add deleted_at column to roles table
ALTER TABLE roles ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- add deleted_at column to users table
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- add deleted_at column to students table
ALTER TABLE students ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- add deleted_at column to guardians table
ALTER TABLE guardians ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- add deleted_at column to fee_types table
