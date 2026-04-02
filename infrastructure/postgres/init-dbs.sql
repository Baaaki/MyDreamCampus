-- PostgreSQL Multi-Database Initialization Script
--
-- This script creates separate databases for each microservice
-- Run automatically by docker-entrypoint-initdb.d
--
-- Databases:
-- - mydreamcampus_auth       (Authentication & Authorization)
-- - mydreamcampus_staff      (Staff Management) ✓ ACTIVE
-- - mydreamcampus_student    (Student Management)
-- - mydreamcampus_catalog    (Course Catalog)
-- - mydreamcampus_enrollment (Course Enrollment)
-- - mydreamcampus_attendance (Attendance Tracking)
-- - mydreamcampus_meal       (Meal Management)

-- Create staff database
SELECT 'CREATE DATABASE mydreamcampus_staff' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'mydreamcampus_staff')\gexec

-- Connect to staff database and enable extensions
\c mydreamcampus_staff;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Additional databases will be added as services are developed
-- Example for future services:
-- SELECT 'CREATE DATABASE mydreamcampus_auth' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'mydreamcampus_auth')\gexec
-- \c mydreamcampus_auth;
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- CREATE EXTENSION IF NOT EXISTS "pgcrypto";
