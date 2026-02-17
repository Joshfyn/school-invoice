-- Rollback initial schema

-- Drop triggers
DROP TRIGGER IF EXISTS update_payments_updated_at ON payments;
DROP TRIGGER IF EXISTS update_invoice_items_updated_at ON invoice_items;
DROP TRIGGER IF EXISTS update_invoices_updated_at ON invoices;
DROP TRIGGER IF EXISTS update_fee_class_amounts_updated_at ON fee_class_amounts;
DROP TRIGGER IF EXISTS update_fee_types_updated_at ON fee_types;
DROP TRIGGER IF EXISTS update_student_guardians_updated_at ON student_guardians;
DROP TRIGGER IF EXISTS update_guardians_updated_at ON guardians;
DROP TRIGGER IF EXISTS update_student_enrollments_updated_at ON student_enrollments;
DROP TRIGGER IF EXISTS update_students_updated_at ON students;
DROP TRIGGER IF EXISTS update_user_class_access_updated_at ON user_class_access;
DROP TRIGGER IF EXISTS update_classes_updated_at ON classes;
DROP TRIGGER IF EXISTS update_terms_updated_at ON terms;
DROP TRIGGER IF EXISTS update_academic_sessions_updated_at ON academic_sessions;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_roles_updated_at ON roles;
DROP TRIGGER IF EXISTS update_schools_updated_at ON schools;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS invoice_items;
DROP TABLE IF EXISTS invoices;
DROP TABLE IF EXISTS fee_class_amounts;
DROP TABLE IF EXISTS fee_types;
DROP TABLE IF EXISTS student_guardians;
DROP TABLE IF EXISTS guardians;
DROP TABLE IF EXISTS student_enrollments;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS user_class_access;
DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS terms;
DROP TABLE IF EXISTS academic_sessions;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS schools;

-- Drop extension (optional, might affect other databases)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
