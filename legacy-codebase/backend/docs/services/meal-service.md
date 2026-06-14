# Meal Service

## Sorumluluk
Yemekhane yönetimi, tekil ve toplu rezervasyon yönetimi, QR kod ile kullanım doğrulama, aylık menü planı yayınlama

---

## İş Kuralları

### Rezervasyon Kuralları
- **Sadece öğrenciler** rezervasyon yapabilir (Teacher ve Admin rezervasyon yapamaz)
- Öğrenci pazartesi-cuma arası istediği günde, öğünde, menüde ve yemekhanede rezervasyon yapabilir
- Aynı gün ve aynı öğün için sadece bir rezervasyon yapılabilir (farklı yemekhaneler dahil)
- İptal edilen veya expire olan slot'a tekrar rezervasyon yapılabilir
- Pending durumda olan slot'a yeni rezervasyon yapılamaz

### Rezervasyon Zaman Penceresi
Tüm rezervasyon işlemleri (oluşturma ve iptal) yalnızca belirli bir zaman penceresi içinde yapılabilir:

| Özellik | Değer |
|---------|-------|
| Pencere Başlangıcı | Pazartesi 08:00 (UTC+3) |
| Pencere Bitişi | Cuma 13:00 (UTC+3) |
| Rezervasyon Geçerlilik Tarihi | Bir sonraki haftanın Pazartesi-Cuma günleri |

**Örnek**: 2 Aralık Pazartesi 08:00 ile 6 Aralık Cuma 13:00 arasında yapılan rezervasyonlar, 9-13 Aralık haftası için geçerlidir. 6 Aralık Cuma 13:01'den itibaren 9-13 Aralık haftası için işlem yapılamaz.

### Yemekhane Validasyonları
- Akşam yemeği rezervasyonu için yemekhane `serves_dinner = true` olmalı
- Vegan menü rezervasyonu için yemekhane `has_vegan_menu = true` olmalı

### Fiyatlandırma
- Öğün fiyatı: 15.00 TRY (sabit)

### Ödeme Akışı (Saga Pattern - Choreography)
1. Kullanıcı rezervasyon isteği yapar
2. Validasyonlar geçerse rezervasyon `pending` olarak DB'ye yazılır
3. Payment Service'e ödeme isteği gönderilir
4. Kullanıcıya payment_url döner
5. Ödeme tamamlanınca `payment.completed` event'i gelir → Rezervasyon `confirmed` olur
6. Ödeme başarısız olursa veya timeout (15 dk) olursa → Rezervasyon `expired` olur

### Payment Reference ID Convention
Payment Service ile iletişimde `reference_id` alanı kullanılır. Tekil ve toplu rezervasyonları ayırt etmek için **prefix convention** uygulanır:

| Rezervasyon Tipi | Prefix | Örnek reference_id |
|------------------|--------|-------------------|
| Tekil | `res_` | `res_550e8400-e29b-41d4-a716-446655440000` |
| Toplu (Batch) | `bat_` | `bat_660e8400-e29b-41d4-a716-446655440000` |

**Ödeme başlatırken**:
- Tekil rezervasyon: `reference_id = "res_" + reservation.id`
- Toplu rezervasyon: `reference_id = "bat_" + batch_id`

**payment.completed event'i geldiğinde**:
- Prefix `res_` ise → `WHERE id = uuid` ile tek rezervasyon güncellenir
- Prefix `bat_` ise → `WHERE batch_id = uuid` ile tüm batch rezervasyonları güncellenir

### Rezervasyon Yaşam Döngüsü

```
[pending] ──payment.completed──→ [confirmed] ──user cancels──→ [cancelled]
    │                                │
    │                                └──qr scan──→ [is_used = true]
    │
    └──timeout/payment.failed──→ [expired]
```

| Status | Açıklama |
|--------|----------|
| `pending` | Ödeme bekleniyor (15 dakika timeout) |
| `confirmed` | Ödeme tamamlandı, aktif rezervasyon |
| `cancelled` | Kullanıcı tarafından iptal edildi, refund tamamlandı |
| `expired` | Ödeme yapılmadı veya başarısız oldu |

### Toplu Rezervasyon Kuralları (All-or-Nothing)
Toplu rezervasyon isteğinde **tüm rezervasyonlar** validasyonlardan geçmelidir. Herhangi birinde hata varsa **hiçbir rezervasyon oluşturulmaz**.

- Çakışan rezervasyon varsa: Hangi tarih/öğün için daha önce rezervasyon alındığı belirtilir
- Validasyon hatası varsa: Tüm hatalar liste halinde döndürülür
- Başarılı durumda: Tüm rezervasyonlar tek transaction'da oluşturulur, tek ödeme URL'i döner

### Refund Akışı (Synchronous)
İptal işleminde refund **senkron** olarak gerçekleştirilir:

1. Kullanıcı iptal isteği yapar
2. Validasyonlar geçerse Payment Service'e sync refund request gönderilir
3. Refund başarılı → Rezervasyon `cancelled` yapılır, outbox'a event eklenir
4. Refund başarısız → İşlem iptal edilir, hata döner (rezervasyon değişmez)

### Veri Saklama Politikası
| Status | Saklama Süresi | Gerekçe |
|--------|----------------|---------|
| `expired` | 7 gün sonra silinir | Ödeme yapılmamış, finansal değeri yok |
| `cancelled` | Süresiz | Finansal kayıt, refund takibi için gerekli |
| `confirmed` | Süresiz | Aktif veya kullanılmış kayıtlar |

---

## QR Kod Sistemi

Her yemekhane için günlük ve öğün bazlı QR kod üretilir. Öğrenci yemekhanede bu QR kodu okutarak rezervasyonunu kullanır.

### QR Kod Yapısı
```
{cafeteria_id}:{date}:{meal_time}:{signature}
```

**Signature oluşturma**:
```
payload = {cafeteria_id}:{date}:{meal_time}
signature = hmac_sha256(payload, QR_SECRET)
```

`QR_SECRET`: Tüm sistem için tek, environment variable olarak tanımlı. Payload zaten yemekhane, tarih ve öğün bazında unique olduğu için tek secret yeterli.

### QR Kullanım Zaman Aralıkları
QR kod okutma işlemi sadece belirlenen saat aralıklarında geçerlidir. Bu kontrol **application seviyesinde** yapılır (veritabanında tutulmaz).

| Öğün | Başlangıç | Bitiş |
|------|-----------|-------|
| Öğle (lunch) | 11:00 | 13:00 |
| Akşam (dinner) | 16:00 | 19:00 |

Zaman dilimi: **UTC+3 (Europe/Istanbul)**

### Kullanım Akışı
1. Öğrenci yemekhanedeki QR kodu telefonuyla okur
2. Frontend QR içeriğini backend'e POST eder
3. Backend signature'ı doğrular (aynı HMAC'i hesaplayıp karşılaştırır)
4. **Tarih kontrolü**: QR'daki tarih bugünün tarihi ile eşleşiyor mu?
5. **Zaman aralığı kontrolü**: Şu anki saat, öğüne uygun aralıkta mı?
6. JWT'den student_id alır
7. İlgili rezervasyon bulunur ve `is_used = true` yapılır

---

## İletişim

### Inbound (RabbitMQ - Event Consumers)

| Event | Kaynak | İşlem |
|-------|--------|-------|
| `student.created` | Student Service | Öğrenci bilgisini local cache'e ekler |
| `student.updated` | Student Service | Öğrenci bilgilerini günceller |
| `student.deactivated` | Student Service | Öğrenciyi ve tüm rezervasyonlarını siler |
| `payment.completed` | Payment Service | Rezervasyon status'unu `confirmed` yapar |
| `payment.failed` | Payment Service | Rezervasyon status'unu `expired` yapar |

### Outbound (REST - Synchronous)

| Hedef | İşlem |
|-------|-------|
| Payment Service | Ödeme başlatma (reference_id + amount) |
| Payment Service | Refund işlemi (sync) |

### Outbound (RabbitMQ - Asynchronous via Outbox)

| Event | Hedef | Tetikleyici |
|-------|-------|-------------|
| `meal.reservation.created` | Notification Service *(daha sonra eklenecek)* | Rezervasyon onaylandığında |
| `meal.reservation.cancelled` | Notification Service *(daha sonra eklenecek)* | Rezervasyon iptal edildiğinde |

---

## Database Schema

```sql
-- Yemekhaneler
CREATE TABLE cafeterias (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    location VARCHAR(255) NOT NULL,
    has_vegan_menu BOOLEAN NOT NULL DEFAULT false,
    serves_dinner BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Öğrenci cache (Student Service'ten event ile senkronize)
CREATE TABLE students_cache (
    id UUID PRIMARY KEY,
    student_number VARCHAR(50) NOT NULL UNIQUE,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,  -- Öğrenci aktif mi? (student.deactivated ile false olur)
    synced_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_students_cache_is_active ON students_cache(is_active) WHERE is_active = true;

-- Rezervasyon öğün tipi
CREATE TYPE meal_time_enum AS ENUM ('lunch', 'dinner');

-- Rezervasyon menü tipi
CREATE TYPE menu_type_enum AS ENUM ('normal', 'vegan');

-- Rezervasyon durumu
CREATE TYPE reservation_status_enum AS ENUM ('pending', 'confirmed', 'cancelled', 'expired');

-- Rezervasyonlar
CREATE TABLE reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id UUID NULL,
    student_id UUID NOT NULL REFERENCES students_cache(id) ON DELETE CASCADE,
    cafeteria_id UUID NOT NULL REFERENCES cafeterias(id),
    reservation_date DATE NOT NULL,
    meal_time meal_time_enum NOT NULL,
    menu_type menu_type_enum NOT NULL DEFAULT 'normal',
    status reservation_status_enum NOT NULL DEFAULT 'pending',
    is_used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMPTZ NULL,
    expires_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Aylık menü planı
CREATE TABLE monthly_menus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    year SMALLINT NOT NULL,
    month SMALLINT NOT NULL CHECK (month BETWEEN 1 AND 12),
    menu_data JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_year_month UNIQUE(year, month)
);

-- Outbox pattern
CREATE TYPE outbox_status_enum AS ENUM ('pending', 'published', 'failed');

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(100) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status outbox_status_enum NOT NULL DEFAULT 'pending',
    retry_count SMALLINT NOT NULL DEFAULT 0,
    max_retries SMALLINT NOT NULL DEFAULT 5,
    next_retry_at TIMESTAMPTZ NULL,
    last_error TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ NULL
);

-- ============ INDEXES ============

-- Rezervasyon sorgulama: Öğrencinin belirli gün/öğün rezervasyonu var mı?
CREATE INDEX idx_reservations_student_date_meal 
ON reservations(student_id, reservation_date, meal_time);

-- Aktif rezervasyonlar için unique constraint (aynı gün/öğüne birden fazla aktif rezervasyon engellenir)
CREATE UNIQUE INDEX idx_unique_active_reservation 
ON reservations(student_id, reservation_date, meal_time) 
WHERE status IN ('pending', 'confirmed');

-- QR doğrulama sorgusu: Belirli yemekhane/gün/öğün/öğrenci için confirmed rezervasyon
CREATE INDEX idx_reservations_qr_validation 
ON reservations(cafeteria_id, reservation_date, meal_time, student_id) 
WHERE status = 'confirmed' AND is_used = false;

-- Batch lookup (toplu rezervasyonlarda payment callback için)
CREATE INDEX idx_reservations_batch 
ON reservations(batch_id) 
WHERE batch_id IS NOT NULL;

-- Expiry job için: Pending ve süresi dolmuş rezervasyonlar
CREATE INDEX idx_reservations_pending_expires 
ON reservations(expires_at) 
WHERE status = 'pending';

-- Cleanup job için: Expired ve 7 günden eski rezervasyonlar
CREATE INDEX idx_reservations_expired_cleanup 
ON reservations(expires_at) 
WHERE status = 'expired';

-- Outbox polling: Pending ve retry zamanı gelmiş eventler
CREATE INDEX idx_outbox_pending_retry 
ON outbox_events(next_retry_at) 
WHERE status = 'pending';

-- Outbox failed events (monitoring için)
CREATE INDEX idx_outbox_failed 
ON outbox_events(created_at) 
WHERE status = 'failed';
```

---

## Outbox Retry Stratejisi

### Retry Politikası

| Özellik | Değer |
|---------|-------|
| Maksimum Retry | 5 |
| Backoff Stratejisi | Exponential (2^n dakika) |
| İlk Retry | 2 dakika sonra |
| Son Retry | 32 dakika sonra (toplam ~62 dakika) |

### Retry Zamanlaması

| Retry # | Bekleme Süresi | Toplam Geçen Süre |
|---------|----------------|-------------------|
| 1 | 2 dakika | 2 dakika |
| 2 | 4 dakika | 6 dakika |
| 3 | 8 dakika | 14 dakika |
| 4 | 16 dakika | 30 dakika |
| 5 | 32 dakika | 62 dakika |

### Retry Akışı

```
[pending] ──publish başarılı──→ [published]
    │
    └──publish başarısız──→ retry_count++, next_retry_at = NOW() + 2^retry_count dakika
                                │
                                └──retry_count >= max_retries──→ [failed]
```

### Failed Event İşleme
- `status = 'failed'` olan eventler otomatik retry edilmez
- Monitoring/alerting sistemi bu eventleri izler
- Manuel müdahale sonrası `status = 'pending'`, `retry_count = 0` yapılarak tekrar denenebilir

### Outbox Poller Konfigürasyonu

| Özellik | Değer |
|---------|-------|
| Poll Interval | 5 saniye |
| Batch Size | 50 |
| Query | `WHERE status = 'pending' AND (next_retry_at IS NULL OR next_retry_at <= NOW())` |

---

## Background Jobs

### 1. Reservation Expiry Job
Ödeme yapılmayan pending rezervasyonları expire eder.

| Özellik | Değer |
|---------|-------|
| Çalışma sıklığı | Her 1 dakikada bir |
| Batch size | 100 |

```sql
WITH expired_batch AS (
    SELECT id FROM reservations
    WHERE status = 'pending' AND expires_at < NOW()
    LIMIT 100
    FOR UPDATE SKIP LOCKED
)
UPDATE reservations 
SET status = 'expired', updated_at = NOW()
WHERE id IN (SELECT id FROM expired_batch);
```

### 2. Expired Reservation Cleanup Job
7 günden eski expired rezervasyonları siler.

| Özellik | Değer |
|---------|-------|
| Çalışma sıklığı | Günde bir kez (03:00 UTC+3) |
| Batch size | 500 |

```sql
WITH cleanup_batch AS (
    SELECT id FROM reservations
    WHERE status = 'expired' AND expires_at < NOW() - INTERVAL '7 days'
    LIMIT 500
    FOR UPDATE SKIP LOCKED
)
DELETE FROM reservations 
WHERE id IN (SELECT id FROM cleanup_batch);
```

### 3. Outbox Poller Job
Pending outbox eventlerini RabbitMQ'ya publish eder.

| Özellik | Değer |
|---------|-------|
| Çalışma sıklığı | Her 5 saniyede bir |
| Batch size | 50 |

```sql
SELECT * FROM outbox_events 
WHERE status = 'pending' 
  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
ORDER BY created_at ASC
LIMIT 50
FOR UPDATE SKIP LOCKED;
```

### 4. Failed Outbox Alert Job
Failed outbox eventlerini monitoring sistemine bildirir.

| Özellik | Değer |
|---------|-------|
| Çalışma sıklığı | Her 15 dakikada bir |
| Alert Threshold | 1 (herhangi bir failed event varsa alert) |

---

## API Endpoints

### Ortak Header'lar

| Header | Zorunlu | Açıklama |
|--------|---------|----------|
| `Authorization` | Evet | Bearer JWT token |

### Ortak Response Format

**Başarılı Response**:
```json
{
  "success": true,
  "data": { ... }
}
```

**Hata Response**:
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message"
  }
}
```

---

### GET /api/v1/meals/cafeterias
Aktif yemekhane listesi

**Yetki**: Authenticated (Student, Teacher, Admin)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "cafeterias": [
      {
        "id": "uuid",
        "name": "Merkez Yemekhane",
        "location": "Ana Kampüs",
        "has_vegan_menu": true,
        "serves_dinner": true
      }
    ]
  }
}
```

---

### POST /api/v1/meals/cafeterias
Yemekhane oluşturma

**Yetki**: Admin

**Request**:
```json
{
  "name": "Yeni Yemekhane",
  "location": "Doğu Kampüs",
  "has_vegan_menu": true,
  "serves_dinner": true
}
```

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Yeni Yemekhane"
  }
}
```

---

### PUT /api/v1/meals/cafeterias/:cafeteria_id
Yemekhane güncelleme

**Yetki**: Admin

**Request**:
```json
{
  "name": "Merkez Yemekhane (Yenilenmiş)",
  "location": "Ana Kampüs A Blok",
  "has_vegan_menu": true,
  "serves_dinner": false
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "name": "Merkez Yemekhane (Yenilenmiş)",
    "location": "Ana Kampüs A Blok",
    "has_vegan_menu": true,
    "serves_dinner": false,
    "is_active": true
  }
}
```

---

### DELETE /api/v1/meals/cafeterias/:cafeteria_id
Yemekhane silme (soft delete)

**Yetki**: Admin

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "message": "Cafeteria deactivated"
  }
}
```

---

### POST /api/v1/meals/reservations
Tekil rezervasyon oluşturma

**Yetki**: Student

**Request**:
```json
{
  "cafeteria_id": "uuid",
  "date": "2025-12-15",
  "meal_time": "dinner",
  "menu_type": "vegan"
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "reservation_id": "uuid",
    "payment_url": "https://sandbox-api.iyzipay.com/payment/...",
    "amount": 15.00,
    "currency": "TRY",
    "expires_at": "2025-12-08T10:15:00+03:00",
    "reservation": {
      "date": "2025-12-15",
      "meal_time": "dinner",
      "menu_type": "vegan",
      "cafeteria_name": "Merkez Yemekhane"
    }
  }
}
```

**İş Mantığı**:
1. Rol kontrolü (sadece Student)
2. **is_active kontrolü**: students_cache'den öğrenci aktif mi? → ❌ 403 STUDENT_DEACTIVATED
3. Zaman penceresi kontrolü (Pazartesi 08:00 - Cuma 13:00)
4. Tarih kontrolü (bir sonraki hafta Pazartesi-Cuma)
5. Yemekhane kontrolü (var mı, aktif mi)
6. Akşam yemeği kontrolü (`serves_dinner`)
7. Vegan menü kontrolü (`has_vegan_menu`)
8. Aktif rezervasyon kontrolü (aynı gün/öğün için pending veya confirmed kayıt varsa → 409 hatası)
9. Rezervasyonu DB'ye yaz (`status: pending`, `expires_at: now + 15 min`)
10. Payment Service'e REST call
11. Payment URL döndür

---

### POST /api/v1/meals/reservations/batch
Haftalık toplu rezervasyon oluşturma (All-or-Nothing)

**Yetki**: Student

**Request**:
```json
{
  "reservations": [
    {
      "cafeteria_id": "uuid",
      "date": "2025-12-15",
      "meal_time": "lunch",
      "menu_type": "normal"
    },
    {
      "cafeteria_id": "uuid",
      "date": "2025-12-15",
      "meal_time": "dinner",
      "menu_type": "vegan"
    },
    {
      "cafeteria_id": "uuid",
      "date": "2025-12-16",
      "meal_time": "lunch",
      "menu_type": "normal"
    }
  ]
}
```

**Response** `200 OK` (Başarılı):
```json
{
  "success": true,
  "data": {
    "batch_id": "uuid",
    "payment_url": "https://sandbox-api.iyzipay.com/payment/...",
    "total_amount": 45.00,
    "currency": "TRY",
    "expires_at": "2025-12-08T10:15:00+03:00",
    "reservations": [
      {
        "reservation_id": "uuid",
        "date": "2025-12-15",
        "meal_time": "lunch",
        "menu_type": "normal",
        "cafeteria_name": "Merkez Yemekhane"
      },
      {
        "reservation_id": "uuid",
        "date": "2025-12-15",
        "meal_time": "dinner",
        "menu_type": "vegan",
        "cafeteria_name": "Merkez Yemekhane"
      },
      {
        "reservation_id": "uuid",
        "date": "2025-12-16",
        "meal_time": "lunch",
        "menu_type": "normal",
        "cafeteria_name": "Doğu Yemekhane"
      }
    ]
  }
}
```

**Response** `409 Conflict` (Çakışan Rezervasyon):
```json
{
  "success": false,
  "error": {
    "code": "RESERVATION_CONFLICTS",
    "message": "Bazı tarih/öğünler için zaten rezervasyonunuz bulunmaktadır",
    "conflicts": [
      {
        "date": "2025-12-15",
        "meal_time": "lunch",
        "existing_reservation_id": "uuid",
        "cafeteria_name": "Merkez Yemekhane",
        "status": "confirmed"
      }
    ]
  }
}
```

**Response** `400 Bad Request` (Validasyon Hataları):
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERRORS",
    "message": "Bazı rezervasyonlarda validasyon hatası var",
    "errors": [
      {
        "index": 1,
        "date": "2025-12-15",
        "meal_time": "dinner",
        "code": "CAFETERIA_NO_DINNER",
        "message": "Bu yemekhane akşam yemeği vermiyor"
      },
      {
        "index": 2,
        "date": "2025-12-16",
        "meal_time": "lunch",
        "code": "CAFETERIA_NO_VEGAN",
        "message": "Bu yemekhanede vegan menü yok"
      }
    ]
  }
}
```

**İş Mantığı**:
1. Rol kontrolü (sadece Student)
2. **is_active kontrolü**: students_cache'den öğrenci aktif mi? → ❌ 403 STUDENT_DEACTIVATED
3. Zaman penceresi kontrolü
4. **Tüm rezervasyonlar için validasyon** (tek geçişte topla):
   - Tarih kontrolü
   - Yemekhane kontrolü
   - Akşam yemeği kontrolü
   - Vegan menü kontrolü
   - Aktif rezervasyon kontrolü (çakışma)
5. **Herhangi bir hata varsa**: Hiçbir rezervasyon oluşturulmaz, tüm hatalar döner
6. **Tüm validasyonlar geçerse**:
   - batch_id üret (UUID)
   - Tüm rezervasyonları **tek transaction'da** aynı batch_id ile DB'ye yaz
   - Toplam tutarı hesapla (rezervasyon sayısı × 15.00)
   - Payment Service'e tek request (`reference_id: batch_id`)
   - Payment URL döndür

---

### GET /api/v1/meals/reservations/my
Kullanıcının rezervasyonları

**Yetki**: Student

**Query Parameters**:
| Parametre | Tip | Zorunlu | Açıklama |
|-----------|-----|---------|----------|
| from_date | date | Hayır | Başlangıç tarihi (YYYY-MM-DD) |
| to_date | date | Hayır | Bitiş tarihi (YYYY-MM-DD) |
| status | string | Hayır | pending, confirmed, cancelled, expired |

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "reservations": [
      {
        "id": "uuid",
        "date": "2025-12-15",
        "meal_time": "lunch",
        "menu_type": "normal",
        "cafeteria": {
          "id": "uuid",
          "name": "Merkez Yemekhane",
          "location": "Ana Kampüs"
        },
        "status": "confirmed",
        "is_used": false,
        "created_at": "2025-12-08T10:00:00+03:00"
      }
    ],
    "summary": {
      "total": 20,
      "confirmed": 15,
      "pending": 2,
      "used": 3,
      "cancelled": 0
    }
  }
}
```

---

### DELETE /api/v1/meals/reservations/:reservation_id
Tekil rezervasyon iptali (Sync Refund)

**Yetki**: Student (sadece kendi rezervasyonları)

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "reservation_id": "uuid",
    "refund_amount": 15.00,
    "currency": "TRY",
    "refund_status": "completed"
  }
}
```

**İş Mantığı**:
1. Rezervasyonun kullanıcıya ait olduğunu doğrula
2. **is_active kontrolü**: students_cache'den öğrenci aktif mi? → ❌ 403 STUDENT_DEACTIVATED
3. Zaman penceresi kontrolü
4. Status kontrolü (sadece `confirmed` iptal edilebilir)
5. is_used kontrolü (kullanılmış rezervasyon iptal edilemez)
6. **Payment Service'e sync refund request**
7. Refund başarılı → Status'u `cancelled` yap
8. Outbox'a `meal.reservation.cancelled` event'i ekle
9. Response döndür

**Refund Başarısız Durumu** `424 Failed Dependency`:
```json
{
  "success": false,
  "error": {
    "code": "REFUND_FAILED",
    "message": "Geri ödeme işlemi başarısız oldu. Lütfen daha sonra tekrar deneyin."
  }
}
```

---

### POST /api/v1/meals/reservations/use
QR kod ile rezervasyon kullanımı

**Yetki**: Student

**Request**:
```json
{
  "qr_payload": "cafeteria_uuid:2025-12-15:lunch:a1b2c3d4e5f6..."
}
```

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "message": "Reservation validated",
    "reservation_id": "uuid",
    "cafeteria_name": "Merkez Yemekhane",
    "meal_time": "lunch",
    "menu_type": "normal"
  }
}
```

**İş Mantığı**:
1. QR payload'ı parse et (`cafeteria_id:date:meal_time:signature`)
2. Signature'ı doğrula: `hmac_sha256(cafeteria_id:date:meal_time, QR_SECRET)` hesapla, karşılaştır
3. **Tarih kontrolü** (application seviyesinde):
   - QR'daki tarih bugünün tarihi ile eşleşmeli
   - Eşleşmiyorsa → `INVALID_QR_DATE` hatası
4. **Zaman aralığı kontrolü** (application seviyesinde):
   - Öğle (lunch): 11:00 - 13:00
   - Akşam (dinner): 16:00 - 19:00
   - Aralık dışındaysa → `OUTSIDE_MEAL_TIME_WINDOW` hatası
5. JWT'den student_id al
6. **is_active kontrolü**: students_cache'den öğrenci aktif mi? → ❌ 403 STUDENT_DEACTIVATED
7. Matching confirmed, unused rezervasyon bul
8. `is_used = true`, `used_at = NOW()` yap
9. Başarı mesajı dön

**Hata Durumları**:
- `INVALID_QR`: QR format hatalı veya signature geçersiz
- `INVALID_QR_DATE`: QR kod bugün için geçerli değil
- `OUTSIDE_MEAL_TIME_WINDOW`: QR okutma saati öğün zaman aralığı dışında
- `NO_RESERVATION`: Bu yemekhane/gün/öğün için rezervasyon yok
- `ALREADY_USED`: Rezervasyon zaten kullanılmış

---

### POST /api/v1/meals/menu/monthly
Aylık menü planı oluşturma/güncelleme

**Yetki**: Admin

**Request**:
```json
{
  "year": 2025,
  "month": 12,
  "menu_data": {
    "2025-12-15": {
      "lunch": {
        "normal": ["Mercimek Çorba", "Tavuk Döner", "Pilav", "Ayran"],
        "vegan": ["Sebze Çorba", "Nohut Yemeği", "Bulgur Pilavı", "Salata"]
      },
      "dinner": {
        "normal": ["Domates Çorba", "Köfte", "Makarna", "Meyve"],
        "vegan": ["Mercimek Çorba", "Barbunya Fasulye", "Pilav", "Meyve"]
      }
    }
  }
}
```

**Response** `201 Created`:
```json
{
  "success": true,
  "data": {
    "year": 2025,
    "month": 12,
    "message": "Monthly menu saved"
  }
}
```

---

### GET /api/v1/meals/menu/monthly
Aylık menü görüntüleme

**Yetki**: Public (authentication gerekmez)

**Query Parameters**:
| Parametre | Tip | Zorunlu | Default |
|-----------|-----|---------|---------|
| year | integer | Hayır | Current year |
| month | integer | Hayır | Current month |

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "year": 2025,
    "month": 12,
    "menu_data": { ... }
  }
}
```

---

### GET /api/v1/meals/cafeterias/:cafeteria_id/qr
Yemekhane QR kodu (Admin/Staff için)

**Yetki**: Admin

**Query Parameters**:
| Parametre | Tip | Zorunlu | Default |
|-----------|-----|---------|---------|
| date | date | Hayır | Today |
| meal_time | string | Evet | - |

**Response** `200 OK`:
```json
{
  "success": true,
  "data": {
    "cafeteria_id": "uuid",
    "cafeteria_name": "Merkez Yemekhane",
    "date": "2025-12-15",
    "meal_time": "lunch",
    "qr_payload": "uuid:2025-12-15:lunch:a1b2c3d4e5f6...",
    "valid_time_window": {
      "start": "11:00",
      "end": "13:00"
    }
  }
}
```

**Not**: `qr_payload` içindeki son kısım HMAC signature'dır. Backend `QR_SECRET` environment variable'ı ile hesaplar.

---

## RabbitMQ Configuration

### Exchange Topology

```
┌─────────────────────────────────────────────────────────────┐
│                    INBOUND EXCHANGES                        │
├─────────────────────────────────────────────────────────────┤
│  student.events (topic)                                     │
│    ├── student.created → meal.student.created.queue         │
│    ├── student.updated → meal.student.updated.queue         │
│    └── student.deactivated → meal.student.deactivated.queue │
│                                                             │
│  payment.events (topic)                                     │
│    ├── payment.completed → meal.payment.completed.queue     │
│    └── payment.failed → meal.payment.failed.queue           │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    OUTBOUND EXCHANGE                        │
├─────────────────────────────────────────────────────────────┤
│  meal.events (topic)                                        │
│    ├── meal.reservation.created                             │
│    └── meal.reservation.cancelled                           │
└─────────────────────────────────────────────────────────────┘
```

### Event Schemas

#### student.created (Consumed)
```json
{
  "event_type": "student.created",
  "event_id": "uuid",
  "timestamp": "2025-12-08T10:00:00Z",
  "data": {
    "id": "uuid",
    "student_number": "2021123456",
    "first_name": "Ali",
    "last_name": "Yılmaz"
  }
}
```

**Handler**: Upsert into students_cache

---

#### student.updated (Consumed)
```json
{
  "event_type": "student.updated",
  "event_id": "uuid",
  "timestamp": "2025-12-08T10:00:00Z",
  "data": {
    "id": "uuid",
    "student_number": "2021123456",
    "first_name": "Ali",
    "last_name": "Yılmazoğlu"
  }
}
```

**Handler**: Update students_cache (upsert for out-of-order tolerance)

---

#### student.deactivated (Consumed)
```json
{
  "event_type": "student.deactivated",
  "event_id": "uuid",
  "timestamp": "2025-12-08T10:00:00Z",
  "data": {
    "id": "uuid"
  }
}
```

**Handler**: Delete from students_cache (CASCADE deletes reservations)

---

#### payment.completed (Consumed)
```json
{
  "event_type": "payment.completed",
  "event_id": "uuid",
  "timestamp": "2025-12-08T10:01:00Z",
  "data": {
    "payment_id": "uuid",
    "reference_id": "bat_660e8400-e29b-41d4-a716-446655440000",
    "amount": 45.00,
    "currency": "TRY"
  }
}
```

**Handler**:
1. reference_id'nin prefix'ini kontrol et
2. Prefix'e göre rezervasyon(lar)ı bul:
   - `res_` prefix → `WHERE id = uuid` (tekil)
   - `bat_` prefix → `WHERE batch_id = uuid` (toplu)
3. Status = `confirmed`, expires_at = NULL
4. Her rezervasyon için outbox'a `meal.reservation.created` ekle

---

#### payment.failed (Consumed)
```json
{
  "event_type": "payment.failed",
  "event_id": "uuid",
  "timestamp": "2025-12-08T10:01:00Z",
  "data": {
    "payment_id": "uuid",
    "reference_id": "res_550e8400-e29b-41d4-a716-446655440000",
    "reason": "insufficient_funds"
  }
}
```

**Handler**:
1. reference_id'nin prefix'ini kontrol et
2. Prefix'e göre rezervasyon(lar)ı bul:
   - `res_` prefix → `WHERE id = uuid` (tekil)
   - `bat_` prefix → `WHERE batch_id = uuid` (toplu)
3. Status = `expired`

---

#### meal.reservation.created (Published)
```json
{
  "event_type": "meal.reservation.created",
  "event_id": "uuid",
  "timestamp": "2025-12-08T10:01:00Z",
  "data": {
    "reservation_id": "uuid",
    "student_id": "uuid",
    "student_number": "2021123456",
    "date": "2025-12-15",
    "meal_time": "dinner",
    "menu_type": "vegan",
    "cafeteria_id": "uuid",
    "cafeteria_name": "Merkez Yemekhane",
    "amount": 15.00,
    "currency": "TRY"
  }
}
```

---

#### meal.reservation.cancelled (Published)
```json
{
  "event_type": "meal.reservation.cancelled",
  "event_id": "uuid",
  "timestamp": "2025-12-08T10:01:00Z",
  "data": {
    "reservation_id": "uuid",
    "student_id": "uuid",
    "student_number": "2021123456",
    "date": "2025-12-15",
    "meal_time": "dinner",
    "refund_amount": 15.00,
    "currency": "TRY"
  }
}
```

---

## Error Codes

### 4xx Client Errors

| HTTP | Code | Açıklama |
|------|------|----------|
| 400 | INVALID_DATE_RANGE | Tarih bir sonraki hafta Pazartesi-Cuma arasında değil |
| 400 | INVALID_MEAL_TIME | Geçersiz öğün (lunch/dinner dışında) |
| 400 | INVALID_MENU_TYPE | Geçersiz menü tipi (normal/vegan dışında) |
| 400 | INVALID_QR | QR kod formatı hatalı veya signature geçersiz |
| 400 | INVALID_QR_DATE | QR kod bugün için geçerli değil (farklı bir güne ait) |
| 400 | OUTSIDE_MEAL_TIME_WINDOW | QR okutma saati öğün zaman aralığı dışında (öğle: 11:00-13:00, akşam: 16:00-19:00) |
| 400 | CAFETERIA_NO_DINNER | Bu yemekhane akşam yemeği vermiyor |
| 400 | CAFETERIA_NO_VEGAN | Bu yemekhanede vegan menü yok |
| 400 | CAFETERIA_NOT_ACTIVE | Bu yemekhane aktif değil |
| 400 | OUTSIDE_RESERVATION_WINDOW | Rezervasyon penceresi dışında (Pzt 08:00 - Cum 13:00) |
| 400 | RESERVATION_ALREADY_USED | Bu rezervasyon zaten kullanılmış |
| 400 | INVALID_STATUS_FOR_CANCEL | Sadece confirmed rezervasyonlar iptal edilebilir |
| 400 | VALIDATION_ERRORS | Toplu rezervasyonda validasyon hataları (detaylar error.errors içinde) |
| 403 | NOT_OWNER | Bu rezervasyon size ait değil |
| 403 | STUDENT_DEACTIVATED | Öğrenci deaktif edilmiş (is_active = false) |
| 403 | ROLE_NOT_ALLOWED | Bu işlem için yetkiniz yok (sadece öğrenciler rezervasyon yapabilir) |
| 404 | CAFETERIA_NOT_FOUND | Yemekhane bulunamadı |
| 404 | RESERVATION_NOT_FOUND | Rezervasyon bulunamadı |
| 404 | NO_RESERVATION | Bu slot için rezervasyonunuz yok |
| 409 | ACTIVE_RESERVATION_EXISTS | Bu tarih/öğün için aktif veya bekleyen rezervasyon var |
| 409 | RESERVATION_CONFLICTS | Toplu rezervasyonda çakışan rezervasyonlar var (detaylar error.conflicts içinde) |

### 5xx Server Errors

| HTTP | Code | Açıklama |
|------|------|----------|
| 424 | PAYMENT_SERVICE_ERROR | Payment Service'e ulaşılamadı veya hata döndü |
| 424 | REFUND_FAILED | Geri ödeme işlemi başarısız oldu |
| 500 | INTERNAL_ERROR | Beklenmeyen sunucu hatası |
| 503 | SERVICE_UNAVAILABLE | Servis geçici olarak kullanılamıyor |

---

## Related Services

| Servis | İletişim Tipi | Açıklama |
|--------|---------------|----------|
| Student Service | Event (RabbitMQ) | Öğrenci CRUD event'leri consume edilir |
| Payment Service | REST (Sync) | Ödeme başlatma ve refund işlemleri |
| Payment Service | Event (RabbitMQ) | Ödeme sonuç event'leri consume edilir |
| Notification Service | Event (RabbitMQ) | Rezervasyon bildirimleri publish edilir *(daha sonra eklenecek)* |

---

## Deployment Notes

### Environment Variables

```bash
# Database
DATABASE_URL=postgresql://user:pass@host:5432/meal_service

# RabbitMQ
RABBITMQ_URL=amqp://user:pass@host:5672/

# Payment Service
PAYMENT_SERVICE_URL=http://payment-service:8080

# QR Code
QR_SECRET=your-random-32-char-secret-here

# Reservation Settings
RESERVATION_TIMEOUT_MINUTES=15
MEAL_PRICE_TRY=15.00

# Meal Time Windows (application config, not DB)
LUNCH_START_HOUR=11
LUNCH_END_HOUR=13
DINNER_START_HOUR=16
DINNER_END_HOUR=19

# Outbox Settings
OUTBOX_POLL_INTERVAL_SECONDS=5
OUTBOX_BATCH_SIZE=50
OUTBOX_MAX_RETRIES=5

# Timezone
TZ=Europe/Istanbul
```

### Health Check Endpoints

```
GET /health         → Basic health
GET /health/ready   → Readiness (DB + RabbitMQ connection)
GET /health/live    → Liveness
```

---

**Version**: 12.1.0  
**Last Updated**: 2025-12-04