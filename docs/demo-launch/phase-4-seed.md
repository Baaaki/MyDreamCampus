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

- [x] **4.1 Fakülte ve bölüm referans tabloları** — `acc5abd`
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

- [x] **4.2 Mock içeriğini seed verisine dönüştür** — `7888f1b`
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

- [x] **4.3 seed.sh'i yeniden yaz** — `4f6ae8e`
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
  > Not (24.09): CI'daki `backend-e2e` Faz 3 sonundan beri kırmızıydı ("no
  > administrative staff seeded"). Kök neden: Faz 1'den beri bootstrap
  > admin'in `force_password_change` bayrağı JWT'de taşınıyor ve seed'in her
  > API çağrısı 403 alıyordu; eski seed hataları yutup "tamamlandı" diyordu.
  > Yeni seed bayrağı admin girişinden önce auth DB'de kapatıyor ve 409 dışı
  > her API hatasında duruyor (409 = kayıt zaten var; yarıda kalan seed
  > kaldığı yerden sürer). 429'da `Retry-After` kadar bekleyip tekrar
  > deniyor: catalog'a ~100 çağrı gidiyor, IP sınırı dakikada 100.
  > Sıra planla aynı; id'ler (öğretmen, ders, öğrenci) API'den sonra ilgili
  > DB'den okunuyor. Öğretmen profilleri artık API'yle
  > (`PUT /api/staff/:id/profile`) yazılıyor; `sql/01-staff.sql` kalktı (JSON
  > alan adları DTO'yla uyuşmuyordu). İdari personel de idempotency
  > kontrolünün arkasına alındı. Kontrol artık "student DB'de öğrenci var mı"
  > (SQL): seed yalnız boş sistemde koşar. Catalog kuralına takılan 12 ön
  > koşul bağlanmıyor, seed logunda tek tek listeleniyor.
  > Demo ilişkisi: ahmet, zeynep dahil 14 öğrencinin danışmanı; onay bekleyen
  > programlar 4.4'te (05-enrollment.sql). `CENG201/202` artık elif.aydin'in
  > (ayse.demir mock'ta Matematik).
  > Süre: yerelde (servisler host'ta, Caddy'siz) 34 sn; CI süresi 4.5'te.

- [x] **4.4 Tarihler ve rol senaryoları** — `9d08c3c`, `255bd12`
  - **Dönem adı** seed günündeki tarihten hesaplanır, backend'in biçiminde
    (`YYYY-YYYY-Fall|Spring`; catalog dönem ve ders açılışında bunu
    doğruluyor, web kayıt sayfası da bunu soruyor — karar 24.09):
    - Eylül–Ocak → `YYYY-YYYY+1-Fall`
    - Şubat–Haziran → `YYYY-1-YYYY-Spring`
    - Temmuz–Ağustos → gelecek Fall

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
  > Not (24.09): Dönemler: aktif `:'semester'` (hard_deadline +120 gün) ve
  > `planned` durumda sonraki dönem (admin arayüzden ona ders açabilsin diye);
  > önkoşul penceresi sonraki dönemde. Periyotlar −35 … +45 gün (4 haftalık
  > yoklama geçmişi kendi periyodu içinde kalsın diye −7 değil). Açılışlar
  > yalnız Bilgisayar Mühendisliği'nde (6 CENG dersi; ahmet 101/102, elif.aydin
  > 201/202, mehmet.kaya 301/350) — diğer bölümlerin öğretmeni ya da
  > öğrencisi mock'ta birlikte yok. Program kuralı: 1. sınıf 101+102, 2. sınıf
  > 201+202, 3–4. sınıf 301+350; Deniz ve Selin CENG102'yi tekrar alıyor.
  > Ahmet'in 3 danışmanı (Ali Çelik, Zeynep Arslan, Oğuz Yıldırım) onay
  > bekliyor. Vize girilmiş ve kilitli (zeynep: 78/85 — eski mock karnesi),
  > final boş. Transkript: 2. sınıftan itibaren 101/102, 3. sınıftan itibaren
  > 201/202, dönem adları kayıt yılından; mutlak not skalası grades'teki
  > tabloyla aynı. Yoklama: 4 geçmiş hafta (3 haftada tek devamsızlık 14'te
  > 10 kuralında "kaldı" gösteriyordu), 5. hafta öğretmenin açması için boş;
  > zeynep'in tek devamsızlığı CENG301 2. hafta. Yemek: bu ayın ve gelecek
  > ayın menüsü `menu_dishes.json`'dan; `monthly_menus` yıl+ay anahtarlı,
  > kafeterya başına menü yok. `menu_data` iki biçimi birden taşıyor: web
  > `{normalMenus, veganMenus}` haftalık yapıyı, mobil `"YYYY-MM-DD" →
  > {lunch, dinner}` yapısını okuyor (eski seed yalnız mobilinkini yazıyordu,
  > web menü sayfası boştu). Rezervasyon: aktif öğrencilere bu haftanın hafta
  > içi öğle yemekleri (bugünden öncekiler kullanılmış) ve gelecek hafta
  > Pzt–Çar; Per–Cum CI'ın yeni rezervasyonu için boş.
  > Düzeltilen eski hata: kayıt programları katalog ders id'siyle
  > yazılıyordu; danışman onayı olayı grades/attendance'ta düşerdi. Artık
  > açılış id'si (yerelde onay → grades kaydı doğrulandı).
  > Yerelde doğrulandı: zeynep programı/notları/transkripti/yoklaması/
  > rezervasyonu; önkoşul penceresi (zeynep kabul, deniz
  > `PREREQUISITES_NOT_MET`, kerem `INVALID_CLASS_LEVEL`); ahmet'in CENG101
  > not listesi (11 öğrenci), final girişi, 5. hafta yoklama açma, 3 bekleyen
  > program; admin için 19 fakülte, 91 ders, 2 dönem, 4 kafeterya, 2 aylık menü.
  > Frontend'e kalan (Faz 5): web kayıt sayfası dönemi gerçek saatten
  > hesaplıyor, sonraki dönemin penceresi arayüzden görünmüyor; admin menü
  > sayfası kaydederken yalnız web biçimini yazıyor, mobil o ayı boş görür.

- [x] **4.5 Doğrulama** — `9f80276`, `38342cf`
  - Kullanıcıdan temiz bir yığın kurmasını iste; komutu kopyala-yapıştır
    olarak ver (volume'lar silinir, uyar):
    `sudo docker compose -f new-backend/infrastructure/docker-compose.yml -f new-backend/infrastructure/docker-compose.standalone.yml down -v && make deploy`
  - Kontrol:
    - seed logu hatasız;
    - üç demo hesabıyla girildiğinde ekranlar dolu (katalogda 111 ders,
      fakülte filtreleri).
  - Sonucu görevin altına yaz.
  > Not (24.09): Temiz yığın komutu kullanıcıya rapordan verildi; kullanıcı
  > makinesindeki sonuç orada beklenir. Bu oturumda iki yerde doğrulandı:
  > (1) Yerel: altyapı konteynerleri + servisler host'ta, boş DB, seed logu
  > hatasız (30 sn); CI golden path'i buna karşı baştan sona yeşil;
  > Playwright ile zeynep/ahmet/admin ekranları dolu (katalog 91 ders —
  > 111 değil, bkz. 4.2 — ve 19 fakülte, dönemler, kafeteryalar, aylık
  > menü; öğrenci kayıt/not/menü, öğretmen not/yoklama/onay). Web öğrenci
  > yoklama sayfası API'ye hiç bağlı değil (sabit "kayıt yok" metni) — veri
  > API'de var, sayfa Faz 5'e not edildi. (2) CI `backend-e2e`: golden path
  > seed içeriğini de kontrol ediyor (91 ders, 19 fakülte, demo öğrencinin
  > aktif dönemde onaylı programı). Golden path'in seed'den sonraki hiç
  > koşmamış adımları da düzeltildi (`fix(infra)` commit'i: kayıt listesine
  > dönem parametresi, `checkout()` içinde `want` değişkenini `expect()`'in
  > ezmesi, var olmayan `/api/grades` rotası, seed'in admin kovasını
  > doldurmasından sonra 429'da bekleme).

## Faz sonu
- README §5 yeşil olmalı.
- README §1 madde 6.
