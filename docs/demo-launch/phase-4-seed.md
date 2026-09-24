# Faz 4 — Referans veri ve seed'in yeniden yazımı

## Başlamadan
- Oku: `new-backend/skills.md`. Seed dosyaları
  `new-backend/infrastructure/seed/` altında (`seed.sh`, `data/*.json`,
  `sql/*.sql`).
- Bağımlılık yok. Bu faz **Faz 5'ten önce** bitmeli: mock içeriği silinmeden
  önce seed'e aktarılmalı.

## Bağlam
- **Mevcut seed küçük:** 3 öğretmen, 8 öğrenci, 6 ders, 4 idari personel.
  Tarihleri seed anına göre (`NOW()`).
- **Frontend'deki mock veri çok daha zengin** (`frontend/src/mock_data/`):
  - `catalog.ts`: 19 fakülte ve yaklaşık 100 bölüm (`mockFaculties` ≈10-750);
    111 ders (`mockCourseCatalog` ≈756; haftalık konular, öğrenme çıktıları,
    koordinatör dahil)
  - `staff.ts`: 13 personel
  - `staff-profile.ts`: öğretmen profilleri (eğitim, makale, proje, ödül,
    burs, idari görev)
  - `students.ts`: öğrenciler
  - `meal.ts`: kafeteryalar ve rezervasyon örnekleri
  - `grades.ts`, `attendance.ts`, `admin_grades.ts`, `admin_attendance.ts`:
    not ve yoklama ekranı örnekleri (API yanıtı biçiminde)
  - `teacher.ts`, `auth.ts`
- **Fakülte/bölüm backend'de tablo değil;** her yerde serbest metin
  (`course_catalog.course_catalog.faculty/department`, `student.students`,
  `staff.admin_staff`).
- **Kullanıcı isteği:** mock içerik kaybolmasın; koddan silinsin, seed ile
  DB'ye girsin.
- **Bu seed sistemin ilk kalıcı durumu olur** (Faz 7). Sonrasında kalıcı
  veriyi süper admin arayüzden yönetir; seed yalnız boş bir sistemde bir kez
  koşar.

## Görevler

- [x] **4.1 Fakülte ve bölüm referans tabloları**
  - catalog migration:
    - `course_catalog.faculties`: id uuid, slug text unique (örn.
      `fac-egitim`), code, name unique, sort_order
    - `course_catalog.departments`: id, faculty_id fk, slug unique (örn.
      `dept-bote`), code, name, unique(faculty_id, name)
  - sqlc, repository ve handler: `GET /api/catalog/faculties`, herkese açık
    (`/courses` gibi).
  - Yanıt frontend'in `Faculty` tipine uymalı
    (`frontend/src/lib/types.ts:158`):
    - `id` = slug (frontend URL parametreleri slug kullanıyor),
    - `departments[].facultyId` = fakültenin slug'ı.
  - Ders/öğrenci/personeldeki metin alanlarına FK ekleme; servisler ayrı
    DB'lerde. Adların birebir eşleşmesi yeterli.
  - Testler.
  - **Commit:** `feat(catalog): add faculty and department reference data`
  > Not (24.09): `departments`'a `description` (mock'taki 6 bölüm açıklaması
  > kaybolmasın, frontend tipinde de var) ve `sort_order` (mock'taki sıra
  > korunsun) eklendi. Yanıt `{"data": [...]}` zarfında, `ListCourses` gibi;
  > boş açıklama alanı yanıtta yer almıyor. Tablolar yalnız seed ile dolar;
  > yazma ucu bu fazın kapsamında değil.

- [x] **4.2 Mock içeriğini seed verisine dönüştür**
  - **Dönüştürücü geçicidir ve repoya girmez.** Oturumun scratchpad dizininde
    bun ile yaz: `frontend/src/mock_data/*.ts` dosyalarını import edip
    `new-backend/infrastructure/seed/data/` altına JSON üretsin.
  - Üretilecek dosyalar:
    - `faculties.json`
    - `courses.json`: 111 ders, catalog create DTO biçiminde (bkz.
      `services/catalog-service/internal/dto`)
    - `teachers.json`: mevcut 3 öğretmen + mock personel
    - `teacher_profiles.json`: `PUT /api/staff/:id/profile` biçiminde
    - `students.json`
    - `cafeterias.json`
    - `menu_dishes.json`: menü üretimi için yemek havuzu
    - `admin_staff.json` olduğu gibi kalır.
  - Eşlemeler:
    - Türkçe enum'lar backend değerlerine çevrilir (örn. `Lisans` →
      `undergraduate`, `Örgün Öğretim` → `on_campus`, zorunlu/seçmeli →
      `mandatory`/`elective`). Doğru değerleri create DTO'nun `binding`
      etiketlerinden al.
    - E-posta alan adı tek olsun: `uni.edu.tr`. Demo hesapları bu alan adında.
    - Mock id'ler kullanılmaz; id'leri API üretir.
  - API yanıtı biçimindeki mock'lar (not, yoklama, rezervasyon ekranları)
    tabloya doğrudan girmez. Aynı içeriği üretecek ham kayıtları 4.4'te SQL
    seed'de oluştur (puan gibi değerler taşınabilir).
  - `auth.ts` (`mockUsers`, `mockSessions`) ve `teacher.ts` taşınmaz; bu
    içerik gerçek auth ve ders verisinden zaten gelir. Bunu görevin altına not
    et.
  - **Kontrol:** JSON'larda 19 fakülte, 111 ders ve tüm öğretmen profilleri
    var; sayıları görevin altına yaz.
  - **Commit:** `chore(infra): move demo content into seed data`
  > Not (24.09): Sayılar: 19 fakülte, 103 bölüm; 91 ders (mock'ta 85 + seed'in
  > 6 CENG dersi — plandaki "111", `catalog.ts`'teki `course_code:` satır
  > sayısıydı: ön koşul referansları ve `mockAvailableCourses` dahil); 13
  > öğretmen, 7 tam profil (kalan 6 öğretmenin mock'ta profili yok, profil
  > kaydına yalnız fakülte yazılır: staff kaydında fakülte alanı yok); 52
  > öğrenci (8 seed + 44 mock); 4 kafeterya; 149 normal + 62 vegan yemek
  > (admin menü sayfasındaki iki havuz).
  > Birleştirme kararları: seed'in 3 öğretmeni mock'ta da var, mock'un
  > bilgileri alındı (ayse.demir → Fen/Matematik, mehmet.kaya → Bilgisayar
  > Mühendisliği). `elif.aydin` hem mock öğretmeni hem seed öğrencisiydi;
  > öğrenci `eylul.aydin` oldu. Mock'un kendi fakülte ağacında olmayan iki
  > kayıt düzeltildi: esra.yavuz (İşletme Fak./YBS → İİBF/YBS), umut.yavuz ve
  > ebru.sari ("Engineering Faculty/Computer Engineering" → Mühendislik/
  > Bilgisayar Mühendisliği). Mock'un 4 kafeteryası seed'in aynı roldeki 2
  > kafeteryasının yerini aldı. Ders koordinatörü e-postası da `uni.edu.tr`.
  > Seed'e özel alanlar: `courses.json` ön koşulları id'siz
  > (`course_code` + `course_name`; id'yi 4.3 çözer), `students.json`'da
  > `advisor_email` ve aktif olmayanlar için `status`. 24 ön koşuldan 12'si
  > catalog'un "ön koşulun sınıf seviyesi daha düşük olmalı" kuralına
  > takılıyor (örn. MAT 1010 ← MAT 1009, ikisi de 1. sınıf); veride
  > duruyorlar, uygulama kararı 4.3'te.
  > Taşınmayanlar: `auth.ts` (`mockUsers`, `mockSessions`) ve `teacher.ts`
  > gerçek auth ve ders verisinden gelir; öğrenci kafeterya sayfasındaki
  > örnek haftalık menü havuzdaki yemeklerden oluşur. API yanıtı biçimindeki
  > not/yoklama/rezervasyon mock'ları 4.4'te SQL ile üretilir.

- [ ] **4.3 seed.sh'i yeniden yaz**
  - Sıra:
    1. fakülte/bölüm (SQL, catalog)
    2. öğretmenler (API) ve profilleri (API)
    3. dersler (API)
    4. öğrenciler (API, `advisor_id` ile)
    5. idari personel (API)
    6. dönem, ders açılışları ve periyotlar (SQL)
    7. grades, attendance, enrollment, meal (SQL)
  - Demo ilişkisi:
    - `ahmet.yilmaz@uni.edu.tr`, `zeynep.sahin@uni.edu.tr`'nin danışmanı olsun;
    - ahmet'in danışmanlığındaki en az 2 başka öğrencinin programı onay
      beklesin.
  - Mevcut idempotency kontrolünü koru: sistem doluysa seed atlanır.
  - Seed süresini ölçüp görevin altına yaz.
  - **Commit:** `feat(infra): rewrite the demo seed around the full content set`

- [ ] **4.4 Tarihler ve rol senaryoları**
  - **Dönem adı** seed günündeki tarihten hesaplanır:
    - Eylül–Ocak → `YYYY-YYYY+1 Güz`
    - Şubat–Haziran → `YYYY-1-YYYY Bahar`
    - Temmuz–Ağustos → gelecek Güz

    SQL dosyalarında sabit `'2025-2026 Güz'` yerine bu değişkeni kullan
    (psql `-v semester=...`). Mevcut `'2025-2026 Bahar'` önkoşul penceresini
    de dönem adından türet.
  - **Açık periyotlar:** kayıt, not girişi ve yoklama bugün açık olsun (örn.
    −7 gün … +45 gün); `hard_deadline` +120 gün.
  - **Her rolün yapacak bir işi olsun:**
    - Öğrenci (zeynep): mevcut dönemde onaylı programı, notları (vize
      girilmiş, final boş), yoklama geçmişi, bu hafta rezervasyonu. Ayrıca
      mevcut önkoşul senaryosuyla kayıt olabileceği açık bir pencere
      (`seed/sql/02-catalog.sql` yorumları).
    - Öğretmen (ahmet): notu girilmemiş final sütunu, yoklama açabileceği
      dersler, onay bekleyen programlar.
    - Admin: dolu katalog, dönemler, kafeteryalar, menüler.
  - **Menü:** tüm aktif kafeteryalar için bu ayın ve gelecek ayın hafta içi
    günleri, `menu_dishes.json` havuzundan.
  - **Commit:** `feat(infra): seed dates and scenarios relative to the seed day`

- [ ] **4.5 Doğrulama**
  - Kullanıcıdan temiz bir yığın kurmasını iste; komutu kopyala-yapıştır
    olarak ver (volume'lar silinir, uyar):
    `sudo docker compose -f new-backend/infrastructure/docker-compose.yml -f new-backend/infrastructure/docker-compose.standalone.yml down -v && make deploy`
  - Kontrol:
    - seed logu hatasız;
    - üç demo hesabıyla girildiğinde ekranlar dolu (katalogda 111 ders,
      fakülte filtreleri).
  - Sonucu görevin altına yaz.

## Faz sonu
- README §5 yeşil olmalı.
- README §1 madde 6.
