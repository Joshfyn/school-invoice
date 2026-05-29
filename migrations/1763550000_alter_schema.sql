-- add deleted_at column to roles table
ALTER TABLE roles ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- add deleted_at column to users table
ALTER TABLE users ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- add deleted_at column to students table
ALTER TABLE students ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- add deleted_at column to guardians table
ALTER TABLE guardians ADD COLUMN deleted_at TIMESTAMP WITH TIME ZONE;

-- add deleted_at column to fee_types table


-- add nin column to guardians table
ALTER TABLE guardians ADD COLUMN nin VARCHAR(20);

CREATE TABLE student_admission (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    student_id UUID NOT NULL REFERENCES students(id) ON DELETE CASCADE,
    school_id UUID NOT NULL REFERENCES schools(id) ON DELETE CASCADE,
    admission_no VARCHAR(50) UNIQUE NOT NULL, -- unique within school
    admission_date DATE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT null,
    deleted_at TIMESTAMP WITH TIME ZONE
);

CREATE UNIQUE INDEX idx_unique_active_student 
ON student_admission (student_id) 
WHERE deleted_at IS NULL;
