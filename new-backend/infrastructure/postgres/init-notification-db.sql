-- Notification service uses a separate PostgreSQL instance / database
-- (plan section 6.2). This file initialises that database — it has no
-- per-module schemas, just the tables the notification consumer manages.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";
