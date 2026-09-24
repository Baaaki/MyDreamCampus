-- meal database: cafeterias, the student projection, this month's and next
-- month's weekday menus, and reservations for this week and next.
--
-- seed.sh passes data/cafeterias.json in as :'cafeterias' and
-- data/menu_dishes.json (the admin menu page's dish pools) as :'dishes'.
\i /seed/sql/_staging-for-services.sql

INSERT INTO meal.cafeterias (name, location, has_vegan_menu, serves_dinner, is_active)
SELECT c ->> 'name', c ->> 'location', (c ->> 'has_vegan_menu')::boolean,
       (c ->> 'serves_dinner')::boolean, (c ->> 'is_active')::boolean
FROM jsonb_array_elements(:'cafeterias'::jsonb) AS c
WHERE NOT EXISTS (SELECT 1 FROM meal.cafeterias m WHERE m.name = c ->> 'name');

INSERT INTO meal.students_view (id, student_number, first_name, last_name, is_active)
SELECT id, student_number, first_name, last_name, is_active
FROM _students
ON CONFLICT (id) DO NOTHING;

-- ============================================================
-- Monthly menus
-- ============================================================
-- One menu per month serves every cafeteria (monthly_menus is keyed on
-- year+month only). menu_data holds the month twice, because the two clients
-- read it differently:
--   web:    {"normalMenus": [week…], "veganMenus": [week…]}, week i = the
--           i-th Monday-to-Sunday week touching the month, each weekday
--           {"items": [soup, main, side, dessert, other], "calories": n}
--   mobile: {"YYYY-MM-DD": {"lunch": [...], "dinner": [...]}} per weekday
-- Both are built from the same picks, so a day's lunch is the web's normal
-- menu for that day. Picks are a stable hash of month, week and day.
CREATE TEMP TABLE _dishes AS
SELECT k.kind, e.d ->> 'category' AS category, e.d ->> 'name' AS name,
       (e.d ->> 'calories')::int AS calories,
       row_number() OVER (PARTITION BY k.kind, e.d ->> 'category' ORDER BY e.ord) - 1 AS idx,
       count(*) OVER (PARTITION BY k.kind, e.d ->> 'category') AS n
FROM (VALUES ('normal'), ('vegan')) AS k(kind)
CROSS JOIN LATERAL jsonb_array_elements(:'dishes'::jsonb -> k.kind) WITH ORDINALITY AS e(d, ord);

CREATE TEMP TABLE _menu_days AS
WITH months AS (
    SELECT (date_trunc('month', CURRENT_DATE) + m * INTERVAL '1 month')::date AS first_day
    FROM generate_series(0, 1) AS m
), shaped AS (
    SELECT first_day,
           EXTRACT(ISODOW FROM first_day)::int - 1 AS lead_days,
           EXTRACT(DAY FROM (first_day + INTERVAL '1 month' - INTERVAL '1 day'))::int AS days_in_month
    FROM months
)
SELECT s.first_day, w.week, dd.dow, dd.day_key, k.kind, meal.meal_time,
       jsonb_agg(d.name ORDER BY cat.ord) AS items,
       sum(d.calories)::int AS calories
FROM shaped s
CROSS JOIN LATERAL generate_series(0, (s.days_in_month + s.lead_days - 1) / 7) AS w(week)
CROSS JOIN (VALUES (1, 'monday'), (2, 'tuesday'), (3, 'wednesday'), (4, 'thursday'), (5, 'friday'))
    AS dd(dow, day_key)
CROSS JOIN (VALUES ('normal'), ('vegan')) AS k(kind)
CROSS JOIN (VALUES ('lunch'), ('dinner')) AS meal(meal_time)
CROSS JOIN (VALUES (1, 'soup'), (2, 'main'), (3, 'side'), (4, 'dessert'), (5, 'other')) AS cat(ord, category)
JOIN _dishes d
  ON d.kind = k.kind AND d.category = cat.category
 AND d.idx = abs(hashtext(concat_ws('|', s.first_day, w.week, dd.dow, k.kind, meal.meal_time, cat.category))) % d.n
GROUP BY s.first_day, w.week, dd.dow, dd.day_key, k.kind, meal.meal_time;

INSERT INTO meal.monthly_menus (year, month, menu_data)
SELECT EXTRACT(YEAR FROM md.first_day)::int, EXTRACT(MONTH FROM md.first_day)::int,
       jsonb_build_object('normalMenus', web.normal, 'veganMenus', web.vegan) || mobile.by_date
FROM (SELECT DISTINCT first_day FROM _menu_days) md
CROSS JOIN LATERAL (
    SELECT jsonb_agg(wk.days ORDER BY wk.week) FILTER (WHERE wk.kind = 'normal') AS normal,
           jsonb_agg(wk.days ORDER BY wk.week) FILTER (WHERE wk.kind = 'vegan')  AS vegan
    FROM (
        SELECT week, kind, jsonb_object_agg(day_key, jsonb_build_object('items', items, 'calories', calories)) AS days
        FROM _menu_days
        WHERE first_day = md.first_day AND meal_time = 'lunch'
        GROUP BY week, kind
    ) wk
) web
CROSS JOIN LATERAL (
    SELECT jsonb_object_agg(to_char(day, 'YYYY-MM-DD'), menus) AS by_date
    FROM (
        SELECT (md.first_day + n)::date AS day,
               jsonb_object_agg(x.meal_time, x.items) AS menus
        FROM generate_series(0, 30) AS n
        JOIN _menu_days x
          ON x.first_day = md.first_day AND x.kind = 'normal'
         AND x.dow = EXTRACT(ISODOW FROM md.first_day + n)
         AND x.week = (n + EXTRACT(ISODOW FROM md.first_day)::int - 1) / 7
        WHERE EXTRACT(MONTH FROM md.first_day + n) = EXTRACT(MONTH FROM md.first_day)
        GROUP BY 1
    ) days
) mobile
ON CONFLICT (year, month) DO NOTHING;

-- ============================================================
-- Reservations
-- ============================================================
-- Paid and confirmed, as a card checkout would leave them (no payment rows:
-- a cancellation still goes through, its refund just reports pending).
-- This week's weekdays, lunch, for every active student — already eaten
-- before today, a QR to show from today on. Next week's Monday to Wednesday
-- too; Thursday and Friday stay free for a fresh booking (CI books them).
INSERT INTO meal.reservations
  (student_id, cafeteria_id, reservation_date, meal_time, menu_type, status, is_used, used_at, created_at)
SELECT sv.id, caf.id, d.day, 'lunch',
       CASE WHEN caf.has_vegan_menu AND abs(hashtext(sv.id::text)) % 5 = 0 THEN 'vegan' ELSE 'normal' END::meal.menu_type_enum,
       'confirmed', d.day < CURRENT_DATE,
       CASE WHEN d.day < CURRENT_DATE THEN d.day + TIME '12:20' END,
       date_trunc('week', CURRENT_DATE) - INTERVAL '3 days'
FROM meal.students_view sv
JOIN _students st ON st.id = sv.id AND st.status = 'active'
CROSS JOIN LATERAL (
    -- zeynep.sahin eats at the central cafeteria; everyone else is spread
    -- over the active ones.
    SELECT c.* FROM meal.cafeterias c
    WHERE c.is_active
    ORDER BY (c.name = 'Merkez Kafeterya' AND sv.student_number = '2021510001') DESC,
             abs(hashtext(sv.id::text || c.id::text))
    LIMIT 1
) caf
CROSS JOIN (
    SELECT date_trunc('week', CURRENT_DATE)::date + n AS day
    FROM (VALUES (0), (1), (2), (3), (4), (7), (8), (9)) AS v(n)
) d
ON CONFLICT (student_id, reservation_date, meal_time) WHERE status IN ('pending', 'confirmed') DO NOTHING;
