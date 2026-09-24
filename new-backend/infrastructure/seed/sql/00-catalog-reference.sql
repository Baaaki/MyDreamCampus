-- catalog database: the faculty/department tree behind GET
-- /api/catalog/faculties and every faculty/department picker. The rows other
-- services store name these by exact text, so this runs before anyone is
-- created. seed.sh passes data/faculties.json in as :'faculties'.
INSERT INTO course_catalog.faculties (slug, code, name, sort_order)
SELECT f ->> 'slug', f ->> 'code', f ->> 'name', (f ->> 'sort_order')::smallint
FROM jsonb_array_elements(:'faculties'::jsonb) AS f
ON CONFLICT (slug) DO NOTHING;

INSERT INTO course_catalog.departments (faculty_id, slug, code, name, description, sort_order)
SELECT fac.id, d ->> 'slug', d ->> 'code', d ->> 'name', d ->> 'description', (d ->> 'sort_order')::smallint
FROM jsonb_array_elements(:'faculties'::jsonb) AS f
CROSS JOIN LATERAL jsonb_array_elements(f -> 'departments') AS d
JOIN course_catalog.faculties fac ON fac.slug = f ->> 'slug'
ON CONFLICT (slug) DO NOTHING;
