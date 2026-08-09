-- meal database: cafeterias, the student projection, three weeks of weekday
-- menus and next week's demo reservations.
\i /seed/sql/_staging-for-services.sql

INSERT INTO meal.cafeterias (name, location, has_vegan_menu, serves_dinner, is_active)
SELECT v.name, v.location, v.has_vegan_menu, v.serves_dinner, true
FROM (VALUES
   ('Merkez Yemekhane', 'Ana Kampüs A Blok', true,  true),
   ('Mühendislik Yemekhanesi', 'Mühendislik Fakültesi', false, false)
 ) AS v(name, location, has_vegan_menu, serves_dinner)
WHERE NOT EXISTS (SELECT 1 FROM meal.cafeterias c WHERE c.name = v.name);

INSERT INTO meal.students_view (id, student_number, first_name, last_name, is_active)
SELECT id, student_number, first_name, last_name, is_active
FROM _students
ON CONFLICT (id) DO NOTHING;

-- Monthly menus, generated RELATIVE to CURRENT_DATE so the "next week" booking
-- flow always has a menu to preview no matter which day the seed runs. Weekday
-- (Mon-Fri) menus for the next ~3 weeks; dishes rotate through a fixed set for
-- variety. The range can straddle a month boundary, so we group by (year,month)
-- and merge into any menu already stored for that month.
INSERT INTO meal.monthly_menus (year, month, menu_data)
SELECT y, m, jsonb_object_agg(iso, menu)
FROM (
  SELECT
    to_char(CURRENT_DATE + s.n, 'YYYY-MM-DD')            AS iso,
    EXTRACT(YEAR  FROM CURRENT_DATE + s.n)::int          AS y,
    EXTRACT(MONTH FROM CURRENT_DATE + s.n)::int          AS m,
    jsonb_build_object(
      'lunch',  (opts.lunch )[1 + (s.n % array_length(opts.lunch,  1))],
      'dinner', (opts.dinner)[1 + (s.n % array_length(opts.dinner, 1))]
    )                                                    AS menu
  FROM generate_series(0, 20) AS s(n)
  CROSS JOIN (
    SELECT
      ARRAY[
        '["Mercimek Çorbası","Tavuk Sote","Pirinç Pilavı","Yoğurt"]'::jsonb,
        '["Domates Çorbası","Fırın Köfte","Bulgur Pilavı","Cacık"]'::jsonb,
        '["Ezogelin Çorbası","Etli Kuru Fasulye","Pirinç Pilavı","Turşu"]'::jsonb,
        '["Yayla Çorbası","Fırın Makarna","Mevsim Salata","Meyve"]'::jsonb,
        '["Tarhana Çorbası","Izgara Tavuk","Şehriyeli Pilav","Ayran"]'::jsonb
      ] AS lunch,
      ARRAY[
        '["Sebze Çorbası","Izgara Köfte","Bulgur Pilavı","Salata"]'::jsonb,
        '["Mercimek Çorbası","Tavuk Şinitzel","Patates Püresi","Turşu"]'::jsonb,
        '["Şehriye Çorbası","Etli Nohut","Pirinç Pilavı","Cacık"]'::jsonb,
        '["Brokoli Çorbası","Balık Tava","Bulgur Pilavı","Limon"]'::jsonb,
        '["Ezogelin Çorbası","Karnıyarık","Şehriyeli Pilav","Yoğurt"]'::jsonb
      ] AS dinner
  ) opts
  WHERE EXTRACT(ISODOW FROM CURRENT_DATE + s.n) < 6   -- Mon..Fri only
) gen
GROUP BY y, m
ON CONFLICT (year, month) DO UPDATE
  SET menu_data  = meal.monthly_menus.menu_data || EXCLUDED.menu_data,
      updated_at = NOW();

-- Demo reservations sit in NEXT week (Mon/Tue/Wed lunch), matching the booking
-- flow which only books the coming week. next Monday = this week's Monday + 7.
INSERT INTO meal.reservations (student_id, cafeteria_id, reservation_date, meal_time, menu_type, status)
SELECT sv.id,
       (SELECT id FROM meal.cafeterias WHERE name = 'Merkez Yemekhane' LIMIT 1),
       (date_trunc('week', CURRENT_DATE)::date + 7) + d.n, 'lunch', 'normal', 'confirmed'
FROM meal.students_view sv
CROSS JOIN (VALUES (0), (1), (2)) AS d(n)
ON CONFLICT (student_id, reservation_date, meal_time) WHERE status IN ('pending', 'confirmed') DO NOTHING;
