# Meal Closed Days - Wizard Entegrasyonu

Dönem oluşturma wizard'ının 2. adımına (Servis Tarihleri) yemekhane kapalı günleri ekleme ve catalog-service -> meal-service arası iletim.

## Problem

Wizard step 2'de 4 servisin (katalog, kayıt, not, yoklama) aktif aralıkları belirleniyor ama yemekhanenin kapalı günleri bu akışta yok. Admin, dönem oluşturduktan sonra ayrıca meal-service'e gidip tek tek kapalı gün eklemek zorunda.

## Karar

Kapalı günleri dönem wizard'ına entegre et. Catalog service aracı rolünde — veriyi meal-service'e iletir, kendi DB'sinde tutmaz.

## Backend Degisiklikleri

### Catalog Service

**Config** (`config/config.go`):
- `MealService ServiceURLConfig` alanı eklenir
- `MEAL_SERVICE_URL` env var, default `http://localhost:8008`

**Handler** (`semester_status_handler.go`):
- `ServiceURLs` struct'ına `Meal string` alanı eklenir
- `createSemesterRequest`'e yeni alan:
  ```go
  type closedDayEntry struct {
      Date   string `json:"date" binding:"required"`   // YYYY-MM-DD
      Reason string `json:"reason" binding:"required"`
  }
  
  type createSemesterRequest struct {
      Name         string           `json:"name" binding:"required"`
      HardDeadline time.Time        `json:"hard_deadline" binding:"required"`
      Periods      *semesterPeriods `json:"periods,omitempty"`
      ClosedDays   []closedDayEntry `json:"closed_days,omitempty"`
  }
  ```
- `distributePeriods` (veya yeni `distributeClosedDays`) fonksiyonunda: meal-service'e `POST /api/meal/admin/closed-days/batch` çağrısı yapılır
- Hata durumunda `period_errors` listesine eklenir (mevcut pattern)

**main.go**:
- `ServiceURLs{..., Meal: cfg.MealService.BaseURL}` eklenir

### Meal Service

**Handler** (`closed_days_handler.go`):
- Yeni endpoint: `POST /admin/closed-days/batch`
- Request body: `{ "closed_days": [{ "date": "YYYY-MM-DD", "reason": "..." }, ...] }`
- Response: `{ "created": [...], "skipped": [...] }` — zaten var olan tarihler skip edilir, hata vermez
- Mevcut `CreateClosedDay` repository fonksiyonunu tekrar kullanır

**Routes** (`RegisterRoutes`):
- `closedDays.POST("/batch", h.BatchCreateClosedDays)` eklenir

## Frontend Degisiklikleri

### Types (`lib/types.ts`)

```typescript
export interface CreateSemesterRequest {
  name: string;
  hard_deadline: string;
  periods?: SemesterPeriods;
  closed_days?: Array<{ date: string; reason: string }>;
}
```

### Wizard Step 2 (`semesters/new/index.tsx`)

**State**:
```typescript
const [closedDays, setClosedDays] = useState<Array<{ date: string; reason: string }>>([]);
```

**UI (StepPeriods icinde)**: Mevcut 4 servis kartinin altina yeni kart:
- Kart baslik: "Yemekhane Kapali Gunler"
- Aciklama: "Yemekhanenin kapali olacagi ozel gunler (resmi tatiller vb.)"
- Tarih input (type="date") + sebep input (text) + "Ekle" butonu
- Eklenen gunlerin listesi: tarih, gun adi (Pazartesi, Sali vb.), sebep, sil butonu
- Gun adi: `format(new Date(date + 'T00:00:00'), 'EEEE', { locale: tr })`

**Validasyon**:
- Tarih bos olamaz, sebep bos olamaz
- Ayni tarih iki kez eklenemez
- Tarih hard deadline'i asamaz
- Kapalı gün eklenmesi zorunlu DEĞİL (bos liste gecerli)

**handleCreateSemester**:
- `closedDays` array'ini `createSemester` payload'ına ekler

### Step 4 Onizleme

Eger `closedDays.length > 0` ise, periyot tablosunun altinda "Yemekhane Kapali Gunler" tablosu gosterilir:
| Tarih | Gun | Sebep |

### system-service.ts

Degisiklik yok — `createSemester` zaten generic `CreateSemesterRequest` gonderiyor, type guncellenmesi yeterli.

## Veri Akisi

```
Frontend (wizard step 2)
  |
  | POST /api/catalog/admin/semesters
  | body: { name, hard_deadline, periods, closed_days }
  |
  v
Catalog Service (semester_status_handler.go)
  |
  | 1. Semester olustur (mevcut)
  | 2. Period'lari dagit (mevcut)
  | 3. closed_days varsa -> meal-service'e ilet (yeni)
  |
  | POST http://meal-service:8008/api/meal/admin/closed-days/batch
  | body: { "closed_days": [...] }
  |
  v
Meal Service (closed_days_handler.go)
  |
  | Her tarihi closed_days tablosuna yaz
  | Cakisan tarihleri skip et
  |
  v
closed_days tablosu (meal DB)
  |
  | validateReservationDate() kontrol eder (mevcut)
```

## Kapsam Disi

- Periyodik kapanma (her pazar vb.) — sadece spesifik tarihler
- Kapalı günlerin düzenlenmesi — sadece ekleme/silme (mevcut meal API ile)
- Dönem sonrası kapalı gün yönetimi — mevcut `/system/semesters` period-tabs sayfasında `ClosedDaysTab` zaten var
