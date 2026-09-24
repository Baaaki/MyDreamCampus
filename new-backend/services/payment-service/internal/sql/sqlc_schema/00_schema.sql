-- sqlc-only bootstrap: goose never reads this directory. The real schema
-- is created by infrastructure/postgres/init-databases.sh; sqlc just needs
-- the CREATE SCHEMA before it parses the schema-qualified migrations.
CREATE SCHEMA IF NOT EXISTS payment;
