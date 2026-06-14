-- +goose Up

-- INVARIANT: Only one semester can be active at any given time.
-- Two semesters cannot be active simultaneously — activating a new semester
-- requires completing (or manually closing) the current active one first.
--
-- This is enforced at the database level via a partial unique index on a constant
-- expression. Since every qualifying row has the same indexed value (true),
-- PostgreSQL will reject any INSERT/UPDATE that would create a second active row.
--
-- Application layer also checks this before attempting activation, but the DB
-- constraint is the ultimate safety net (e.g. against race conditions, SQL injection).
-- See: docs/semester-wizard-plan.md "Tek Aktif Dönem Kuralı"
CREATE UNIQUE INDEX idx_semesters_single_active
    ON semesters ((true))
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS idx_semesters_single_active;
