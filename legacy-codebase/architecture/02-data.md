> **AI: önce [00-ai-rules.md](00-ai-rules.md) oku** (çalışma kuralları + glossary). Index: [README.md](README.md).

## 4. Database Stratejisi: Schema-per-Module

### 4.1 Yapı

Tek PostgreSQL instance, içinde modül başına bir schema:

```
mydreamcampus (database)
├── auth                  schema
│   ├── users
│   ├── sessions
│   └── refresh_tokens
├── staff
├── student
├── course_catalog
├── enrollment
├── attendance
├── grades
├── meal
├── payment
└── public                # outbox_events burada (shared infrastructure)
```

### 4.2 Kurallar

| Kural | Açıklama |
|---|---|
| Cross-schema FK | **YASAK** — sadece ID referansı (`student_id UUID NOT NULL`, FK yok) |
| Cross-schema SELECT/JOIN | **YASAK** — kod review'da reject |
| Migration | Modül başına ayrı migration klasörü (`internal/modules/<modul>/sql/migrations/`) |
| sqlc | Modül başına ayrı `sqlc.yaml` + ayrı schema scope |
| Connection | Modül başına `search_path` set edilir: `SET search_path TO <modul>, public` |
| Outbox | `<modul>.outbox_events` — her modül kendi schema'sında (mevcut servis kalıbı, Bölüm 5.2) |

### 4.3 Enforcement (Faz 0 kararı — Bölüm 13)

Her modüle ayrı PostgreSQL role + sadece kendi schema'sına `GRANT`. Yanlış kod yazılsa bile DB engellesin. Faz 0'da kurulup kurulmayacağı **Bölüm 13'te açık soru**dur — varsayılan: convention + PR review ile başla, role-based enforcement Faz 1 sonrası eklenir.

```sql
CREATE ROLE enrollment_module LOGIN PASSWORD '...';
GRANT USAGE ON SCHEMA enrollment TO enrollment_module;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA enrollment TO enrollment_module;
GRANT USAGE ON SCHEMA public TO enrollment_module;          -- outbox icin
GRANT SELECT, INSERT ON public.outbox_events TO enrollment_module;
```

### 4.4 Modül ayırma kolaylığı

Modülü servis olarak ayırma günü:
```bash
pg_dump --schema=meal mydreamcampus > meal_schema.sql
psql -h new-meal-db < meal_schema.sql
# Modul kodunu yeni servise tasi, monolith'ten sil. Bitti.
```
Cross-schema FK olmadığı için schema dump tek başına self-contained.

---

## 8. Cross-Module Veri Stratejileri

Üç strateji, hangisi ne zaman:

### Strateji 1 — ID + Public Service Lookup (varsayılan, ~%80)

```sql
CREATE TABLE enrollment.enrollments (
    id UUID PRIMARY KEY,
    student_id UUID NOT NULL,    -- sadece ID, FK yok
    course_id UUID NOT NULL,
    semester TEXT NOT NULL
);
```

```go
// enrollment modulu icinde:
enrollments := repo.ListByCourse(courseID)
studentIDs := extractIDs(enrollments)
students := studentService.GetByIDs(ctx, studentIDs)  // batch — N+1 onler
return join(enrollments, students)                    // memory'de birlestir
```

**Servise ayrılma günü:** `studentService.GetByIDs` HTTP RPC'ye dönüşür. Aynı arayüz, farklı transport.

### Strateji 2 — Snapshot / Historical Duplicate (zorunlu)

Tarihsel doğruluk gereken yerlerde — domain'in dayatması, tartışma yok:

```sql
CREATE TABLE payment.transactions (
    id UUID PRIMARY KEY,
    student_id UUID NOT NULL,
    amount NUMERIC(10,2) NOT NULL,
    student_name_at_payment TEXT NOT NULL,    -- snapshot
    student_email_at_payment TEXT NOT NULL,   -- snapshot
    paid_at TIMESTAMPTZ NOT NULL
);
```

Öğrenci ismini değiştirse bile makbuz/fatura eski isimle kalır — feature, bug değil.

### Strateji 3 — Read Model / Projection (kontrollü kullanım)

Event'lerle senkronize edilen lokal kopya. Strateji 1 monolith içinde Go fonksiyon çağrısı olduğu için ucuz, ancak iki somut durumda Strateji 3 zorunlu olur:

**Strateji 3'e geçiş eşikleri (somut):**
- **(a) Yük eşiği:** Bir endpoint'in p95 latency'si 500ms'i aşarsa **VE** bu yavaşlığın >50%'sı cross-module lookup'lardan kaynaklanıyorsa (profiling ile doğrulanmış).
- **(b) Throughput eşiği:** Bir endpoint dakikada 1000+ isteğe ulaşırsa ve her istek başına 2+ cross-module lookup yapıyorsa.
- **(c) attendance modülü zorunluluğu:** attendance modülünün yoğun saatleri (Bölüm 1'deki 100K/2sa zirve) → student.full_name lookup hot path. **Faz 2'de attendance modülü taşınırken Strateji 3 doğrudan uygulanır.**

**attendance modülü için somut Strateji 3 örneği:**

```sql
-- attendance schema'sinda — student'in read model'i
CREATE TABLE attendance.students_view (
    id            UUID PRIMARY KEY,
    full_name     TEXT NOT NULL,
    student_number TEXT NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Senkronizasyon: student modülü `student.created`/`student.updated`/`student.deactivated` event'leri publish eder, attendance modülü kendi schema'sında consume edip view'unu günceller. **Bu in-process consume** (modüller arası RabbitMQ kullanılmaz — Bölüm 0.2 Glossary'de "in-process call" tanımı). Concrete kalıp Faz 2'de yazılır.

**(b) ve (c) dışında Strateji 3 kullanma.** Diğer modüller için Strateji 1 yeterlidir; CPU/IO ölçümü yapmadan read model eklemek code bloat üretir.

---

## 9. Modül-Modül Strateji Haritası

> **Bu tablo eksiksizdir.** Yeni cross-module ilişki eklemek için ÖNCE kullanıcıya sor (Bölüm 0.1).
> "→" sembolü "consume eder/okur" anlamında, listelenmemiş modüller arası **iletişim YOKTUR**.

| Modül | Veri sahipliği | Cross-module tüketicileri (eksiksiz liste) |
|---|---|---|
| **auth** | users, sessions, refresh_tokens | JWT claim yeterli — cross-module read pratikte yok |
| **staff** | staff, teacher_profiles | course_catalog→**1**, attendance→**1**, grades→**2** (öğretmen snapshot at grade time) |
| **student** | students | enrollment→**1**, attendance→**3** (read model, hot path), grades→**2**, payment→**2** |
| **course_catalog** | courses, semester_courses | enrollment→**2** (ders snapshot at enroll time), attendance→**1**, grades→**2** |
| **enrollment** | enrollments | attendance→**1** (validation: "öğrenci derse kayıtlı mı?"), grades→**1** (validation) |
| **attendance** | attendance_records | grades→**1** (katılım notu hesabı için, soğuk path) |
| **grades** | grade_records | Self-contained — başka modül grade verisi tüketmez |
| **meal** | meals, closed_days, meal_credits | **payment→event**: meal modülü `meal.credit_purchase_requested` event'i publish eder, payment modülü consume eder. **Başka cross-module iletişimi yoktur.** |
| **payment** | transactions | Self-contained — başka modül payment verisi tüketmez (meal'den event consume eder, ama payment kendi kayıtlarını tutar) |
| **notification** (servis) | delivery_log, processed_events | Cross-DB sorgusu YOK — event payload self-contained (Bölüm 5.3) |

**meal-payment izolasyonu (önemli kural — Bölüm 1):**
meal modülü **sadece** payment modülü ile, **sadece event üzerinden** iletişim kurar. meal modülü `student`, `auth` veya başka modülün public Service interface'ini çağırmaz. Yemek kredisi alımı akışı:

```
[meal.PurchaseCredit handler] → outbox: meal.credit_purchase_requested
                                  ↓ (event)
[payment.events binding]      → payment modulu consume eder, transaction yaratir
                              → outbox: payment.succeeded
                                  ↓ (event)
[notification.events binding] → notification welcome receipt email gonderir
[meal.events binding (kendi consumer'i)] → meal modulu credit balance'i gunceller
```

**attendance yük profili (önemli kural — Bölüm 1):**
attendance modülü 2 saatte 100.000 yoklama isteği zirve yükü alır. Her yoklama isteğinde öğrenci adı görüntülenir → **cross-module student lookup hot path**. Bu yüzden Strateji 3 (read model) **doğrudan ilk implementasyonda** uygulanır, profile ölçümü beklemez. `attendance.students_view` tablosu student modülünün event'leriyle senkronize tutulur (Bölüm 8 Strateji 3 örneği).

**Net üç kural:**
- **Kural A — Snapshot (Strateji 2):** Tarihsel kayıt tutan tabloda (transcript, makbuz, not, ödeme history, kayıt history) cross-module veri **her zaman snapshot**. İstisna yok.
- **Kural B — Lookup (Strateji 1):** Operasyonel/anlık veride varsayılan; in-process Go fonksiyon çağrısı.
- **Kural C — Read Model (Strateji 3):** Sadece (i) attendance.students_view (bilinen yük) (ii) Bölüm 8'deki ölçüm eşiklerinden birinin aşılması durumunda. Başka yerde kullanılmaz.

---

