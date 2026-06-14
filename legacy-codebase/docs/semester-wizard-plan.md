# Dönem Açma Wizard'ı — Uygulama Planı

## Tek Aktif Dönem Kuralı

> **INVARIANT: Sistemde aynı anda yalnızca BİR dönem aktif olabilir.**
> İki dönem aynı anda aktif durumda bulunamaz. Yeni bir dönem aktifleştirmek için
> mevcut aktif dönemin önce tamamlanması (completed) gerekir.

**Neden?**
Tüm sistem (ders kayıt, notlar, yoklama, yemekhane) tek bir aktif dönem üzerinden çalışır.
İki aktif dönem olsaydı:
- Öğrenci hangi döneme kayıt olacağını bilemezdi
- Hoca hangi dönemin notlarını gireceğini bilemezdi
- Period kontrolleri hangi dönemin period'larına bakacağını bilemezdi

**Nasıl enforce ediliyor?**
1. **Veritabanı seviyesi**: `idx_semesters_single_active` partial unique index — PostgreSQL,
   `status = 'active'` olan ikinci bir satır eklenmesini fiziksel olarak engeller (race condition dahil).
2. **Uygulama seviyesi**: `ActivateSemester` handler'ı aktivasyon öncesi `HasActiveSemester` kontrolü yapar
   ve kullanıcıya anlaşılır bir hata mesajı döner.
3. **Trigger seviyesi**: Mevcut `prevent_semester_reactivation()` trigger'ı durum geçişlerini
   sadece `planned → active → completed` yönünde izin verir.

---

## İki Katmanlı Değişmezlik Modeli

Sistemde iki farklı türde veri var ve farklı zamanlarda kilitleniyorlar:

### Dönemlik Ders Açılışları / Semester Courses (hangi ders bu dönem açılıyor, hangi hoca, hangi sınıf, kapasite)
- **Kilitlenme anı:** Dönem aktifleştirildiğinde
- **Kontrol mekanizması:** Admin aktivasyon öncesi review yapar
- **Aktifleştirmeden sonra:** Dönemlik ders eklenemez, silinemez, hoca değiştirilemez — KİMSE TARAFINDAN

> **ÖNEMLİ: Ders Kataloğu (course_catalog) ile karıştırılmamalı!**
> - **Ders Kataloğu** = üniversitedeki TÜM derslerin tanım listesi (CSE101, MAT201 vb.)
>   → Herhangi bir zamanda yeni ders eklenebilir, silinebilir. Dönemden bağımsızdır.
>   → Kataloğa ders eklemek operasyonel hiçbir şeyi etkilemez, sadece bir sonraki dönem
>     açılışında seçilebilecek ders havuzunu genişletir.
> - **Dönemlik Ders Açılışı (semester_courses)** = bu dönem fiilen açılan dersler
>   (CSE101 bu dönem Dr. Ahmet ile A-201'de açılıyor)
>   → Dönem aktifleştirildiğinde donar. Öğrenciler bu listeye göre kayıt olur.

### Operasyonel Veri (not, yoklama, kayıt)
- **Kilitlenme anı:** hard_deadline geçtiğinde
- **Kontrol mekanizması:** 3 katmanlı enforcement model (aşağıda)
- **hard_deadline sonrası:** Hiçbir veri değiştirilemez — KİMSE TARAFINDAN

```
Dönem Yaşam Döngüsü:

  planned ──→ active ──→ completed (hard_deadline geçince)
     │           │              │
     │           │              └─ HER ŞEY DONDU
     │           │                 Ders yapısı ❌  Operasyonel veri ❌
     │           │
     │           └─ DERS YAPISI DONDU, operasyonel veri period'lara göre açık
     │              Ders yapısı ❌  Kayıt/Not/Yoklama ✅ (period içinde)
     │
     └─ HER ŞEY DEĞİŞEBİLİR
        Ders ekle/sil ✅  Period'lar tanımla ✅

⚠️  AYNI ANDA SADECE 1 DÖNEM AKTİF OLABİLİR
    Yeni dönem aktifleştirmek → önce mevcut aktif dönemi tamamla
    (DB partial unique index + uygulama katmanı kontrolü ile enforce edilir)
```

### Not & Yoklama İçin: 3 Katmanlı Enforcement

```
┌─────────────────────────────────────────────────────────┐
│  Katman 1: hard_deadline geçti mi?                      │
│  EVET → HERKES İÇİN REDDET (admin dahil, istisna yok)  │
│  HAYIR ↓                                                │
├─────────────────────────────────────────────────────────┤
│  Katman 2: Çağıran admin mi?                            │
│  EVET → İZİN VER (admin hard_deadline'e kadar her       │
│         zaman düzeltme yapabilir)                        │
│  HAYIR ↓                                                │
├─────────────────────────────────────────────────────────┤
│  Katman 3: Şu an servisin period'u içinde mi?           │
│  EVET → İZİN VER                                        │
│  HAYIR → REDDET                                         │
└─────────────────────────────────────────────────────────┘
```

| Durum | Öğrenci/Hoca | Admin |
|-------|-------------|-------|
| Period içinde | ✅ | ✅ |
| Period dışında, hard_deadline geçmedi | ❌ | ✅ Manuel düzeltme |
| hard_deadline geçti | ❌ | ❌ Kimse değiştiremez |

> **Bu model sadece NOT ve YOKLAMA servisleri için geçerlidir.**

### Ders Kayıt (Enrollment) İçin: Sıkı Period Kilidi

```
┌─────────────────────────────────────────────────────────┐
│  Period içinde mi?                                      │
│  EVET → Sadece öğrenci kayıt yapabilir                  │
│  HAYIR → HERKES İÇİN REDDET (admin dahil, istisna yok) │
└─────────────────────────────────────────────────────────┘
```

| Durum | Öğrenci | Admin |
|-------|---------|-------|
| Period içinde | ✅ Kayıt yapabilir | ❌ Müdahale edemez |
| Period dışında | ❌ | ❌ Kimse değiştiremez |

> **Neden enrollment farklı?**
> Ders kaydı öğrencinin kendi sorumluluğundadır. Admin müdahalesi gerekmez.
> Period açıkken öğrenci kaydını yapar, period kapandığında kimse değiştiremez.
> hard_deadline kontrolüne gerek yok — period zaten yeterli kilittir.

### Yemekhane İçin: Sadece Kapalı Gün Kontrolü

```
┌─────────────────────────────────────────────────────────┐
│  Randevu tarihi kapalı gün mü? (tatil, bayram vb.)     │
│  EVET → Randevu alınamaz                                │
│  HAYIR → Normal akış                                    │
└─────────────────────────────────────────────────────────┘
```

> Yemekhane akademik takvimden bağımsızdır. Period veya hard_deadline kontrolü yoktur.
> Sadece admin tarafından tanımlanan kapalı günlerde (Ramazan Bayramı, resmi tatil vb.)
> randevu alınamaz. Başka kısıtlama yoktur.

### Dönemlik Ders Açılışı Kilidi (aktifleştirme sonrası)

| İşlem | Aktifleştirme öncesi | Aktifleştirme sonrası |
|-------|---------------------|----------------------|
| Dönemlik ders aç (semester_course) | ✅ Admin (gelecekte: + bölüm başkanı) | ❌ Kimse |
| Dönemlik ders sil | ✅ Admin (gelecekte: + bölüm başkanı) | ❌ Kimse |
| Dersi verecek hocayı belirle | ✅ Admin (gelecekte: + bölüm başkanı) | ❌ Kimse |
| Kontenjan (max_capacity) belirle | ✅ Admin (gelecekte: + bölüm başkanı) | ❌ Kimse |
| Ders saati belirle (gün + saat slotları) | ✅ Admin (gelecekte: + bölüm başkanı) | ❌ Kimse |
| Sınıf/derslik ata (classroom_location) | ✅ Admin (gelecekte: + bölüm başkanı) | ❌ Kimse |
| Değerlendirme şeması belirle (vize %40, final %60 vb.) | ✅ Admin (gelecekte: + bölüm başkanı) | ❌ Kimse |

### Ders Kataloğu (course_catalog) — DÖNEMDEN BAĞIMSIZ

| İşlem | Ne zaman | Kim |
|-------|----------|-----|
| Kataloğa yeni ders ekle (MAT301 vb.) | Her zaman | Admin |
| Katalogdan ders sil/güncelle | Her zaman | Admin |

Kataloğa ders eklemek/silmek **hiçbir aktif dönemi etkilemez**.
Sadece bir sonraki dönem açılışında seçilebilecek ders havuzunu değiştirir.

---

## Gelecek Uyumluluk: Bölüm Başkanı Rolü

### Şu anki akış (küçük okul, tek admin):
```
Admin → Dönem oluştur + tarihleri belirle
Admin → Dersleri ekle (tüm bölümler)
Admin → Dönemi başlat
```

### Gelecekteki akış (büyük üniversite, bölüm başkanları):
```
Admin  → Dönem oluştur + tarihleri belirle (status: "planning")
         ↓
Bölüm  → Her bölüm başkanı kendi bölümünün derslerini ekler
Başkanları  (sadece kendi department_id'sine ait course_code'ları görebilir)
         ↓
Admin  → Tüm bölümlerin hazır olduğunu görür → "Dönemi Başlat"
```

### Bu esnekliği sağlamak için mimari kurallar:

1. **Backend'de TEK atomik endpoint YAPMA** — adımları ayrı tut:
   - `POST /semesters` → dönem + period'lar oluşturur (status: "planning")
   - `POST /semesters/:semester/courses` → ders ekler (mevcut endpoint)
   - `PUT /semesters/:id/activate` → dönemi başlatır (mevcut endpoint)

2. **Frontend wizard bu 3 API'yi sırayla çağırır** — tek sayfada ama ayrı adımlar

3. **Semester status'a yeni bir ara durum ekle:**
   ```
   planned → planning → active → completed
   ```
   - `planned`: sadece oluşturuldu, henüz tarihler bile girilmedi (eski davranış)
   - `planning`: tarihler belirlendi, ders ekleme açık (wizard Step 2-3 arası)
   - `active`: dönem başladı, öğrenci/hoca işlem yapabilir
   - `completed`: dönem bitti, kimse değiştiremez

   **Şimdilik:** `planned` → `active` → `completed` yeterli.
   **Gelecekte:** `planning` durumu eklendiğinde bölüm başkanları
   sadece `planning` durumundaki döneme ders ekleyebilir.

4. **Ders ekleme endpoint'i department-scoped olsun:**
   - Şu an: `POST /semesters/:semester/courses` — herhangi bir dersi ekleyebilir
   - Zaten course_code → course_catalog → faculty_id ilişkisi var
   - Gelecekte: middleware'da `X-User-Department` header'ını kontrol et
   - Bölüm başkanı sadece kendi faculty_id'sine ait dersleri ekleyebilir
   - **Şimdi kod değişikliği gerekmez** — sadece route'a `RequireAdmin()` yerine
     `RequireRole("admin", "department_head")` konur

5. **Frontend wizard'da "Ders Ekle" step'i component olarak ayrı tutulsun:**
   - Gelecekte bu component bölüm başkanının kendi dashboard'ına da konabilir
   - Aynı component, farklı context (wizard vs standalone)

---

## FAZ 1 — Shared: Ortak Enforcement Fonksiyonu

### Neden önce bu?
Tüm servisler aynı 3 katmanlı kontrolü kullanacak. Tekrar yazmamak için shared'a ekliyoruz.

### Değişiklikler

**1. `backend/shared/rules/semester_rules.go` (YENİ DOSYA)**

> **Zaman Makinesi Uyumu:** Tüm zaman karşılaştırmaları `clock.Now()` kullanacak (`time.Now()` DEĞİL).
> Mevcut `period_rules.go` ve `grading_rules.go` zaten `clock.Now()` kullanıyor.
> Bu sayede admin zaman makinesini ayarladığında period ve hard_deadline kontrolleri
> simüle edilmiş zamana göre çalışır. Ek bir iş gerekmez.

İki ayrı fonksiyon — farklı servisler farklı kurallar kullanır:

```
CanOperateInSemester(params) → {Allowed, Reason}
  ↳ Not ve Yoklama servisleri için (3 katmanlı, admin bypass var)

params:
  - HardDeadline    time.Time
  - PeriodStart     *time.Time
  - PeriodEnd       *time.Time
  - IsAdminAction   bool

Mantık:
  1. NOW > HardDeadline → REDDET (reason: "semester_ended", herkes için)
  2. IsAdminAction → İZİN VER
  3. PeriodStart == nil → İZİN VER (period tanımlanmamışsa serbest)
  4. NOW < PeriodStart → REDDET (reason: "period_not_started")
  5. NOW > PeriodEnd → REDDET (reason: "period_ended")
  6. İZİN VER
```

```
CanEnrollInSemester(params) → {Allowed, Reason}
  ↳ Ders kayıt servisi için (sıkı period kilidi, admin bypass YOK)

params:
  - PeriodStart     *time.Time
  - PeriodEnd       *time.Time

Mantık:
  1. PeriodStart == nil → REDDET (period tanımlanmamışsa kayıt kapalı)
  2. NOW < PeriodStart → REDDET (reason: "enrollment_not_started")
  3. NOW > PeriodEnd → REDDET (reason: "enrollment_ended")
  4. İZİN VER
  // NOT: hard_deadline veya admin bypass kontrolü YOK — period tek kilittir
```

**2. `backend/shared/rules/grading_rules.go` (GÜNCELLEME)**
- Mevcut `CanEditGrade()` fonksiyonuna `HardDeadline` parametresi ekle
- Admin bypass'ını hard_deadline SONRASINA taşı (şu an admin her şeyi bypass ediyor, hard_deadline kontrolü yok)
- Yeni sıra: hard_deadline → admin bypass → lock check → period check

---

## FAZ 2 — Backend: Catalog Service'in Internal API'sini Genişlet

### Neden?
Diğer servisler (grades, attendance, enrollment) semester hard_deadline'ı bilmiyor. Bunu öğrenmeleri için catalog service'in internal endpoint'ini genişletmemiz lazım.

### Değişiklikler

**1. `GET /api/catalog/internal/semesters/:name/info` (YENİ ENDPOINT)**
```json
Response:
{
  "name": "2025-2026-Spring",
  "status": "active",
  "hard_deadline": "2026-06-30T23:59:59Z",
  "is_past_deadline": false
}
```

**Dosyalar:**
- `backend/services/course-catalog-service/internal/handler/semester_status_handler.go` — yeni handler method
- `backend/services/course-catalog-service/internal/repository/semester_status_repository.go` — yeni query

**2. `backend/shared/client/semester_client.go` (YENİ DOSYA)**
- Diğer servislerin catalog service'i çağırması için HTTP client
- `GetSemesterInfo(ctx, semesterName) → SemesterInfo{Name, Status, HardDeadline}`
- Response'u Redis'te cache'le (TTL: 5 dakika) — her istek için HTTP çağrısı yapmamak için

---

## FAZ 3 — Backend: Attendance Service'e Period + Hard Deadline Desteği

### Neden?
Attendance service'te şu an HİÇBİR tarih kontrolü yok. En büyük güvenlik açığı burası.

### Değişiklikler

**1. Migration: `simple_periods` tablosu ekle**
- `backend/services/attendance-service/sql/migrations/XXXX_create_simple_periods_table.sql`
- Enrollment service ile aynı şema (semester, period_start, period_end, is_active)

**2. sqlc query'leri**
- `backend/services/attendance-service/sql/queries/simple_periods.sql`
- GetActivePeriodBySemester, CreatePeriod, UpdatePeriod, DeletePeriod, ListPeriods

**3. Period CRUD handler (admin)**
- Mevcut `backend/shared/handler/simple_period_handler.go` kullanılacak (zaten enrollment'ta kullanılıyor)
- `cmd/main.go`'da admin route grubuna ekle: `/admin/periods`

**4. Enforcement: 3 katmanlı kontrol ekle**
Her write endpoint'e kontrol ekle:

| Endpoint | Method | Kontrol |
|----------|--------|---------|
| CreateSession | POST /sessions | semester_client → hard_deadline + period + role check |
| ScanQR | POST /scan | semester_client → hard_deadline + period + role check |
| ManualMark | POST /sessions/:id/manual | semester_client → hard_deadline + period + role check |

**5. Admin manuel düzeltme endpoint'leri (YENİ)**
- `POST /admin/attendance/sessions/:sessionId/mark` — admin tek öğrencinin yoklamasını düzeltir
- `POST /admin/attendance/sessions/:sessionId/bulk-mark` — admin toplu yoklama düzeltir
- Bu endpoint'ler sadece hard_deadline kontrolü yapar (period bypass)

**Dosyalar:**
- `sql/migrations/XXXX_create_simple_periods_table.sql` (yeni)
- `sql/queries/simple_periods.sql` (yeni)
- `internal/service/attendance_service.go` (güncelleme — enforcement ekle)
- `internal/handler/attendance_handler.go` (güncelleme — admin endpoint'leri)
- `cmd/main.go` (güncelleme — route'lar)

---

## FAZ 4 — Backend: Grades Service'e Hard Deadline Koruması

### Neden?
Grades service'te period kontrolü var ama hard_deadline kontrolü YOK. Admin her şeyi bypass edebiliyor, dönem bitse bile.

### Değişiklikler

**1. `checkCanEditGrade()` güncelle**
- `backend/services/grades-service/internal/service/grade_service.go`
- Semester client ile hard_deadline bilgisini al
- `CanEditGrade()` parametrelerine `HardDeadline` ekle
- Yeni kontrol sırası:
  1. hard_deadline geçti mi? → **REDDET** (admin dahil)
  2. Admin mi? → İZİN VER
  3. Score locked mı? → REDDET
  4. Period geçti mi? → REDDET

**2. Admin endpoint'lerini de hard_deadline ile koru**
- `/admin/appeal` — hard_deadline kontrolü ekle
- `/admin/scores/unlock` — hard_deadline kontrolü ekle
- `/admin/scores/lock` — hard_deadline kontrolü ekle
- Şu an bu endpoint'lerde HİÇBİR tarih kontrolü yok

**Dosyalar:**
- `backend/shared/rules/grading_rules.go` (güncelleme)
- `backend/services/grades-service/internal/service/grade_service.go` (güncelleme)
- `backend/services/grades-service/cmd/main.go` (semester client injection)

---

## FAZ 5 — Backend: Enrollment Service — Sıkı Period Kilidi

### Neden?
Enrollment'ta period kontrolü kısmen var ama: (1) period dışında hâlâ kayıt yapılabiliyor (period tanımlı değilse serbest), (2) admin bypass mevcut. Yeni model: period içinde sadece öğrenci, period dışında KİMSE.

### Değişiklikler

**1. Enrollment service'in period kontrolünü sıkılaştır**
- `backend/services/enrollment-service/internal/service/enrollment_service.go`
- Mevcut: period yoksa → serbest. Yeni: period yoksa → REDDET
- Mevcut: admin bypass yok. Yeni: admin bypass yok (değişiklik yok, zaten doğru)
- Period içinde: sadece öğrenci kayıt yapabilir (`RequireStudent()`)
- Period dışında: herkes reddedilir (admin dahil)
- **hard_deadline kontrolüne gerek yok** — period zaten yeterli kilittir

**2. Admin enrollment endpoint'i YOK**
- Admin ders kaydına müdahale edemez. Bu öğrencinin kendi sorumluluğudur.
- Mevcut admin endpoint'leri varsa kaldırılmalı veya read-only olmalı.

**Dosyalar:**
- `backend/services/enrollment-service/internal/service/enrollment_service.go` (güncelleme — sıkı period kontrolü)
- `backend/services/enrollment-service/cmd/main.go` (route temizliği, admin write endpoint varsa kaldır)

---

## FAZ 6 — Backend: Dönem Oluşturma + Period'ları Dağıtma

### Neden?
Wizard'ın Step 1'de dönem + period'ları tek seferde oluşturması için.
Ama TEK atomik endpoint DEĞİL — ayrı adımlar, wizard frontend'de orkestre eder.

### Mimari Karar: Neden tek endpoint değil?
Gelecekte bölüm başkanı rolü eklendiğinde:
- Adım 1 (dönem + period) → admin yapar
- Adım 2 (ders ekleme) → bölüm başkanları yapar
- Adım 3 (dönem başlatma) → admin yapar

Tek atomik endpoint olsa bu ayrıştırma mümkün olmazdı.

### Değişiklikler

**1. `POST /api/catalog/admin/semesters` endpoint'ini genişlet (GÜNCELLEME)**

Mevcut request:
```json
{ "name": "2025-2026-Spring", "hard_deadline": "2026-06-30T23:59:59Z" }
```

Yeni request (periods opsiyonel ekleniyor):
```json
{
  "name": "2025-2026-Spring",
  "hard_deadline": "2026-06-30T23:59:59Z",
  "periods": {
    "enrollment": { "start": "2026-02-10T00:00:00Z", "end": "2026-02-25T23:59:59Z" },
    "grading":    { "start": "2026-02-10T00:00:00Z", "end": "2026-06-25T23:59:59Z" },
    "attendance": { "start": "2026-02-24T00:00:00Z", "end": "2026-06-15T23:59:59Z" },
    "catalog":    { "start": "2026-01-15T00:00:00Z", "end": "2026-02-20T23:59:59Z" }
  }
}
```

Eğer `periods` gönderilirse:
1. Semester'ı oluştur (planned)
2. Validation: tüm period.end ≤ hard_deadline
3. Catalog period → kendi DB'sine yaz
4. Diğer servislere internal HTTP ile period oluştur:
   - `POST /api/enrollment/internal/periods`
   - `POST /api/grades/internal/periods`
   - `POST /api/attendance/internal/periods`

Eğer `periods` gönderilmezse → mevcut davranış (sadece semester oluştur).
**Geriye uyumlu.**

**2. Internal period endpoint'leri (her serviste YENİ)**
- `POST /api/{service}/internal/periods` — catalog service'in çağıracağı endpoint
- Sadece `X-Internal-Secret` header ile erişilebilir
- Period oluşturur ve döner

**3. Wizard frontend akışı (3 ayrı API çağrısı):**
```
Step 1-2: POST /semesters  (dönem + periods)  → semester_id döner
Step 3:   POST /semesters/:semester/courses   (her ders için, loop)
Step 4:   PUT /semesters/:id/activate         (dönemi başlat)
```

Her adım bağımsız — biri başarısız olursa kullanıcı düzeltip tekrar deneyebilir.
Planned dönem silinebilir (temizlik için).

**4. Dönemlik ders açılışı kilidi: aktif dönemde semester_course değişikliğini engelle**

Mevcut `CreateSemesterCourse`, `UpdateSemesterCourse`, `DeleteSemesterCourse` endpoint'lerine kontrol ekle:

```
if semester.status == "active" || semester.status == "completed" {
    → 403 Forbidden: "Semester course offerings are frozen after activation"
}
```

Bu kontrol `semester_service.go`'da, mevcut `IsSemesterActive()` kontrolünün YERİNE gelir.
Şu anki mantık: "sadece active dönemde ders eklenebilir" → Yeni mantık: "sadece planned dönemde dönemlik ders eklenebilir/silinebilir/değiştirilebilir".

**NOT:** Bu kısıtlama sadece `semester_courses` tablosu içindir!
`course_catalog` tablosu (ders tanım kataloğu) her zaman düzenlenebilir — dönemden bağımsızdır.

**Dosyalar:**
- `backend/services/course-catalog-service/internal/handler/semester_status_handler.go` (güncelleme)
- `backend/services/course-catalog-service/internal/dto/semester_dto.go` (güncelleme — periods eklenir)
- `backend/services/course-catalog-service/internal/service/semester_service.go` (güncelleme — ders yapısı kilidi)
- `backend/services/enrollment-service/internal/handler/internal_handler.go` (yeni)
- `backend/services/grades-service/internal/handler/internal_handler.go` (yeni)
- `backend/services/attendance-service/internal/handler/internal_handler.go` (yeni)

---

## FAZ 7 — Meal Service: Kapalı Gün Kontrolü (Zaten Mevcut)

### Durum
Kapalı gün kontrolü backend'de zaten çalışıyor:
- `closed_days` tablosu var, admin CRUD endpoint'leri var
- `validateReservationDate()` → `IsDateClosed()` kontrol ediyor
- Kapalı günde randevu almaya çalışan öğrenci `CAFETERIA_CLOSED` hatası alıyor

### Yapılacak (minimal)
- Frontend'de kapalı gün yönetim UI'ı zaten var (ClosedDaysTab)
- Backend zaten enforce ediyor
- **Ek bir iş gerekmiyor** — mevcut implementasyon yeterli

---

## FAZ 8 — Frontend: Dönem Açma Wizard'ı

### Yeni sayfa: `/admin/system/semesters/new`

**Step 1: Dönem Bilgileri**
- Dönem adı (auto-suggest: sonraki dönem adını hesapla)
- Hard deadline (tarih + saat seçici)

**Step 2: Servis Tarih Aralıkları**
- 4 satır, her biri bir servis:
  | Servis | Başlangıç | Bitiş |
  |--------|-----------|-------|
  | Ders Kayıt (enrollment) | tarih seç | tarih seç |
  | Not Giriş (grading) | tarih seç | tarih seç |
  | Yoklama (attendance) | tarih seç | tarih seç |
  | Ders Açma (catalog) | tarih seç | tarih seç |
- Validation: hiçbir period.end > hard_deadline olamaz

**Step 3: Ders Listesi**
- Ders ara → hoca seç → sınıf → kapasite → program → ekle
- Eklenen dersler tablosu (düzenle/sil)
- **Component olarak ayrı tutulacak** — gelecekte bölüm başkanı dashboard'unda da kullanılabilir

**Step 4: Önizleme + Onayla**
- Dönem özeti kartı
- Period'lar özeti
- Ders tablosu (hoca, sınıf, saat)
- "Dönemi Aç" butonu → 3 API çağrısı sırayla:
  1. `POST /semesters` (dönem + periods)
  2. `POST /semesters/:semester/courses` (her ders için loop)
  3. `PUT /semesters/:id/activate`

**Dosyalar:**
- `frontend/src/pages/admin/system/semesters/new/index.tsx` (yeni — wizard container)
- `frontend/src/pages/admin/system/semesters/new/components/step-semester-info.tsx` (yeni)
- `frontend/src/pages/admin/system/semesters/new/components/step-periods.tsx` (yeni)
- `frontend/src/pages/admin/system/semesters/new/components/step-courses.tsx` (yeni)
- `frontend/src/pages/admin/system/semesters/new/components/step-preview.tsx` (yeni)
- `frontend/src/components/semester/course-form.tsx` (yeni — reusable ders ekleme formu)
- `frontend/src/lib/services/system-service.ts` (güncelleme)
- `frontend/src/lib/types.ts` (güncelleme)

---

## FAZ 9 — Frontend: Read-Only Modu (Completed Dönem)

### Değişiklikler

- Not sayfalarında: completed dönem → düzenleme butonları gizle
- Yoklama sayfalarında: completed dönem → düzenleme butonları gizle
- Period dışındaysa (ama hard_deadline geçmemişse): "Period dışında, sadece admin düzeltme yapabilir" uyarısı
- Hard_deadline geçmişse: "Bu dönem tamamlanmıştır, veriler değiştirilemez" banner'ı

---

## Uygulama Sırası ve Bağımlılıklar

```
FAZ 1 (shared rules: CanOperateInSemester + CanEnrollInSemester)
  ↓
FAZ 2 (catalog internal API — hard_deadline bilgisini dışa aç)
  ↓
FAZ 3, 4 (attendance + grades enforcement) — paralel yapılabilir
FAZ 5 (enrollment sıkı period kilidi) — paralel yapılabilir
  ↓
FAZ 6 (dönem + period dağıtımı + ders yapısı kilidi) — FAZ 3,4,5'e bağlı
  ↓
FAZ 7 (meal kapalı gün) — zaten mevcut, ek iş yok
  ↓
FAZ 8 (frontend wizard) — FAZ 6'ya bağlı
  ↓
FAZ 9 (frontend read-only) — FAZ 3,4,5'e bağlı
```

## Gelecekte Bölüm Başkanı Rolü Eklemek İçin:

Sadece şu değişiklikler yeterli olacak:

1. **Auth service**: `department_head` rolü ekle
2. **RBAC middleware**: `RequireRole("admin", "department_head")` — ders ekleme endpoint'ine
3. **Semester service**: `planning` ara durumu ekle (planned → planning → active → completed)
4. **Ders ekleme**: Department filtreleme ekle (`X-User-Department` header → sadece kendi bölümünün derslerini görebilir)
5. **Frontend**: Bölüm başkanı dashboard'u — mevcut `course-form.tsx` component'ini kullan
6. **Frontend**: Admin "Dönem Durumu" sayfası — hangi bölümler derslerini tamamladı, hangisi bekliyor

**Hiçbir mimari değişiklik gerekmeyecek** — sadece yeni rol + filtre + UI.

---

## Kod İçi Yorum Kuralları

Uygulama sırasında aşağıdaki noktalara **kod içinde yorum** bırakılacak. Gelecekte "neden böyle yapıldı?" sorusuna cevap vermek için.

### Backend yorumları:

**1. `shared/rules/semester_rules.go` — CanOperateInSemester()**
```go
// Three-layer semester enforcement for GRADES and ATTENDANCE only:
// Layer 1: hard_deadline passed → REJECT for everyone (including admin)
// Layer 2: caller is admin → ALLOW (admin can fix data until hard_deadline)
// Layer 3: within service period → ALLOW/REJECT for teacher/student
//
// Why this order matters:
// - hard_deadline is the absolute lock. After it, even admin cannot modify data.
//   This ensures completed semester data integrity for auditing/accreditation.
// - Admin bypass before period check allows manual corrections (e.g. wrong grade,
//   missed attendance) without waiting for a new period window.
//
// NOTE: Enrollment uses a DIFFERENT model — strict period lock, no admin bypass.
// See CanEnrollInSemester() for enrollment-specific rules.
```

**2. `shared/rules/grading_rules.go` — CanEditGrade()**
```go
// IMPORTANT: hard_deadline check MUST come before admin bypass.
// Previous behavior allowed admin to bypass everything — this was a security gap
// where admin could modify grades even after semester completion.
// See: docs/semester-wizard-plan.md "Yetki Modeli (3 Katmanlı)"
```

**3. Grades ve Attendance servislerinin enforcement noktası**
```go
// Semester enforcement: checks hard_deadline + admin bypass + period window.
// Uses CanOperateInSemester() — the three-layer model.
// Admin can override period but NOT hard_deadline.
// See: docs/semester-wizard-plan.md
```

**3b. Enrollment service'in enforcement noktası**
```go
// Enrollment uses STRICT period lock — different from grades/attendance.
// Period inside: only students can enroll. Period outside: NOBODY can modify (admin included).
// No hard_deadline check needed — period is the only lock.
// Why no admin override? Enrollment is the student's own responsibility.
// Admin should not add/remove courses on behalf of students.
// See: docs/semester-wizard-plan.md "Ders Kayıt (Enrollment) İçin: Sıkı Period Kilidi"
```

**4. Catalog service — semester creation endpoint (periods dağıtımı)**
```go
// Semester setup distributes periods to each service via internal HTTP calls.
// This is intentionally NOT a single atomic endpoint — the steps are separated so that
// in the future, a "department_head" role can handle course creation (step 2)
// while admin handles semester creation (step 1) and activation (step 3).
// See: docs/semester-wizard-plan.md "Gelecek Uyumluluk: Bölüm Başkanı Rolü"
```

**5. Attendance/Grades/Enrollment — internal period endpoint**
```go
// Internal endpoint: called by catalog-service during semester setup to create
// this service's period. Protected by X-Internal-Secret header.
// Not exposed to external clients.
```

**6. Admin attendance correction endpoints**
```go
// Admin can manually correct attendance records when period has ended
// but hard_deadline has not passed. This covers cases like:
// - System errors during attendance scanning
// - Student disputes about recorded attendance
// After hard_deadline, even these endpoints will reject modifications.
```

**7. Catalog service — semester_course ekleme/silme/güncelleme endpoint'leri**
```go
// IMPORTANT: "semester_courses" (courses offered this semester) vs "course_catalog" (all courses ever defined).
// - course_catalog: can be modified anytime, independent of semesters. Adding a course to the catalog
//   has no operational effect — it just expands the pool of courses available for future semester offerings.
// - semester_courses: FROZEN once semester is activated. No add/remove/modify — not even admin.
//
// This is intentional: the activation step IS the review/approval gate.
// In a large university with ~100 departments, department heads add semester_courses during
// "planning" phase, admin reviews everything, then activates. Any mistakes must be
// caught BEFORE activation. After activation, students enroll based on the published
// course list — changing it would break enrollments, schedules, and trust.
//
// See: docs/semester-wizard-plan.md "İki Katmanlı Değişmezlik Modeli"
```

**8. Catalog service — activate endpoint**
```go
// Activating a semester is a one-way, irreversible action that freezes the course structure.
// This endpoint should be called only after admin has reviewed all course offerings.
// After activation:
//   - No courses can be added, removed, or modified
//   - Only operational data (enrollment, grades, attendance) flows via period windows
//   - Course structure remains frozen until hard_deadline (and beyond)
//
// INVARIANT: Only one semester can be active at any given time.
// Before activating, we check if another semester is already active.
// If yes, the request is rejected with a clear error message.
// This rule exists because the entire system (enrollment, grades, attendance)
// operates on a single active semester — having two active semesters would
// cause ambiguity in which semester students enroll in, teachers grade for, etc.
// The database also enforces this via idx_semesters_single_active partial unique index
// as a safety net against race conditions.
// See: docs/semester-wizard-plan.md "Tek Aktif Dönem Kuralı"
```

### Frontend yorumları:

**9. Wizard sayfası — API çağrı sırası**
```typescript
// Wizard executes 3 separate API calls (not one atomic call):
// 1. POST /semesters — create semester + distribute periods
// 2. POST /semesters/:s/courses — add each course (loop)
// 3. PUT /semesters/:id/activate — activate semester
//
// Why separate calls instead of one atomic endpoint?
// Future extensibility: when department_head role is added,
// step 2 will be done by department heads (not admin).
// See: docs/semester-wizard-plan.md "Gelecek Uyumluluk"
```

**10. Course form component (`components/semester/course-form.tsx`)**
```typescript
// Reusable course creation form — extracted as a standalone component so it can be used in:
// 1. Semester wizard (admin creates courses for all departments)
// 2. Future: department head dashboard (each head creates courses for their own department)
// Keep this component context-agnostic — it should not assume WHO is adding the course.
```

**11. Read-only mode banner**
```typescript
// Two different read-only states:
// 1. Period ended but hard_deadline not passed → "Admin can still make corrections"
// 2. Hard_deadline passed → "Semester completed, no modifications possible"
// These are intentionally different messages because the implications differ.
```

## Toplam Etki

- **Güvenlik**: Bitmiş dönem verileri asla değiştirilemez (hard_deadline sonrası mutlak kilit)
- **Esneklik**: Admin, period dışında bile düzeltme yapabilir (hard_deadline'e kadar)
- **UX**: Tek wizard ile dönem açma (dakikalar vs saatler)
- **Tutarlılık**: Tüm servisler aynı 3 katmanlı modeli kullanır
- **Ölçeklenebilirlik**: Bölüm başkanı rolü mimari değişiklik olmadan eklenebilir
