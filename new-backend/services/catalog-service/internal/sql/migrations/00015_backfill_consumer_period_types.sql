-- +goose Up
-- Until the period projection existed, enrollment, grades and attendance all
-- read catalog's single row per semester — that shared row *was* their
-- deadline. Copying it into the three consumer types preserves exactly what
-- those services were already enforcing.
--
-- Without this the migration path is silently broken: the old internal HTTP
-- fan-out never worked, so no consumer-typed row exists anywhere,
-- /internal/periods/republish finds nothing to publish, and every projection
-- starts empty — which makes the period checks fail open.
INSERT INTO course_catalog.academic_periods
  (semester, period_start, period_end, is_active, period_type)
-- is_active is nullable here but NOT NULL in the projection tables, so the
-- copy has to settle it rather than pass NULL along.
SELECT p.semester, p.period_start, p.period_end, COALESCE(p.is_active, true), t.period_type
FROM course_catalog.academic_periods p
CROSS JOIN (VALUES ('enrollment'), ('grading'), ('attendance')) AS t(period_type)
WHERE p.period_type = 'catalog'
ON CONFLICT (semester, period_type) DO NOTHING;

-- +goose Down
DELETE FROM course_catalog.academic_periods WHERE period_type <> 'catalog';
