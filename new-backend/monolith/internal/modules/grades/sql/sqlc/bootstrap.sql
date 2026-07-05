-- sqlc catalog bootstrap — NOT a goose migration, never applied to a database.
-- The real schema is created by infrastructure/postgres/init-dbs.sql before
-- migrations run; sqlc builds its catalog only from the files listed in
-- sqlc.yaml, so the schema must be declared before the migrations are parsed.
CREATE SCHEMA grades;
