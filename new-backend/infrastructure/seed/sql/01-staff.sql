-- staff database. Purely local — teacher profiles hang off staff.staff, which
-- the API seed created a moment ago. Runs first because catalog needs the ids.

INSERT INTO staff.teacher_profiles (staff_id, academic_title, faculty, education, awards, articles, projects)
SELECT s.id, m.title, 'Mühendislik Fakültesi',
       m.education::jsonb, m.awards::jsonb, m.articles::jsonb, m.projects::jsonb
FROM staff.staff s
JOIN (VALUES
  ('ahmet.yilmaz@uni.edu.tr', 'Prof. Dr.',
   '[{"degree":"Lisans","school":"Boğaziçi Üniversitesi","field":"Bilgisayar Mühendisliği","year":2001},{"degree":"Doktora","school":"ODTÜ","field":"Bilgisayar Mühendisliği","year":2009}]',
   '[{"title":"TÜBİTAK Bilim Ödülü","year":2018},{"title":"Yılın Öğretim Üyesi","year":2021}]',
   '[{"title":"Deep Learning for Graph Networks","journal":"IEEE Access","year":2020}]',
   '[{"name":"Akıllı Kampüs Platformu","role":"Yürütücü","year":2022}]'),
  ('ayse.demir@uni.edu.tr', 'Doç. Dr.',
   '[{"degree":"Lisans","school":"İTÜ","field":"Bilgisayar Mühendisliği","year":2006},{"degree":"Doktora","school":"Sabancı Üniversitesi","field":"Yapay Zeka","year":2014}]',
   '[{"title":"En İyi Bildiri Ödülü, UBMK","year":2019}]',
   '[{"title":"Explainable AI in Education","journal":"Springer LNCS","year":2021}]',
   '[{"name":"Öğrenci Başarı Tahmini","role":"Araştırmacı","year":2023}]'),
  ('mehmet.kaya@uni.edu.tr', 'Dr. Öğr. Üyesi',
   '[{"degree":"Lisans","school":"Hacettepe Üniversitesi","field":"Elektrik-Elektronik Mühendisliği","year":2010},{"degree":"Doktora","school":"Bilkent Üniversitesi","field":"Gömülü Sistemler","year":2018}]',
   '[{"title":"Genç Bilim İnsanı Teşvik Ödülü","year":2020}]',
   '[{"title":"Low-Power IoT Scheduling","journal":"Elsevier IoT","year":2022}]',
   '[{"name":"Kampüs Enerji İzleme","role":"Yürütücü","year":2024}]')
 ) AS m(email, title, education, awards, articles, projects) ON m.email = s.email
ON CONFLICT (staff_id) DO NOTHING;
