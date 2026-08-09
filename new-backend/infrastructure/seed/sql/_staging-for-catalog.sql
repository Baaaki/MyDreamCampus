-- Staging rows the catalog seed needs from ANOTHER database.
--
-- staff.staff used to be one JOIN away; it is a separate database now, so
-- seed.sh exports the three columns this seed reads and each target session
-- loads them into a TEMP table. TEMP because the table belongs to one psql
-- session: a re-run cannot collide with a leftover, and no seed-only table
-- ever lands in a service's schema.
CREATE TEMP TABLE _staff (
    id         uuid,
    email      text,
    first_name text,
    last_name  text
);
\copy _staff FROM '/tmp/seed-stage/staff.csv' CSV
