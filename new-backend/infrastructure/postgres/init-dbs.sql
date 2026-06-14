-- Initialise the monolith database with one schema per module.
-- Plan section 4.1 — schema-per-module on a single PostgreSQL instance.
-- Cross-schema FK / JOIN are forbidden by convention (plan section 4.2).
--
-- This script runs once on first container start (Postgres entrypoint
-- /docker-entrypoint-initdb.d). Schema creation is idempotent so it is
-- safe to re-run after recreating the container against a fresh volume.

-- Enable required extensions on the public schema.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Per-module schemas (plan section 4.1).
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS staff;
CREATE SCHEMA IF NOT EXISTS student;
CREATE SCHEMA IF NOT EXISTS course_catalog;
CREATE SCHEMA IF NOT EXISTS enrollment;
CREATE SCHEMA IF NOT EXISTS attendance;
CREATE SCHEMA IF NOT EXISTS grades;
CREATE SCHEMA IF NOT EXISTS meal;
CREATE SCHEMA IF NOT EXISTS payment;
