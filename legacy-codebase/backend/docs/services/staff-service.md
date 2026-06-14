# Staff Service (HIGH Priority)

## Sorumluluk
İdari personel yönetimi (Admin ve Teacher - öğrenci dışındaki tüm kullanıcılar)

**Source of Truth**: Bu servis öğrenci dışındaki kullanıcılar için canonical data source'dur.

---

## İletişim

### Inbound (REST)
- Admin kullanıcıları staff CRUD işlemleri yapabilir
- `POST /api/v1/staff` - Yeni staff ekleme (sadece teacher, admin API'den eklenemez)
- `GET /api/v1/staff/:id` - Staff detayı
- `PUT /api/v1/staff/:id` - Staff güncelleme
- `DELETE /api/v1/staff/:id` - Staff silme (soft delete)
- `GET /api/v1/staff` - Staff listesi (pagination, filtering)

### Outbound (RabbitMQ)
Staff ekleme/güncelleme/silme olaylarında event yayınlar:
- `staff.created` - Yeni staff ekleme
- `staff.updated` - Staff bilgisi güncelleme
- `staff.deactivated` - Staff deaktivasyon (soft delete)

### Event Consumers
- **Auth Service**: Staff bilgilerini senkronize eder (login için gerekli)
- **Student Service**: `staff.deactivated` eventi dinler → orphaned öğrencilerin advisor_id = NULL yapılır

---

## Database Schema

### Staff Table

```sql
CREATE TABLE staff (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(50) NOT NULL,              -- teacher, admin
    department VARCHAR(100),                -- Sadece teacher için (full string: "Computer Science")
    phone VARCHAR(50),
    office_location VARCHAR(255),
    is_active BOOLEAN DEFAULT true,         -- Active staff flag (soft delete: false)
    deleted_at TIMESTAMP DEFAULT NULL,      -- Soft delete timestamp
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Unique constraint only for active staff (soft delete support)
CREATE UNIQUE INDEX idx_staff_email_unique
    ON staff(email) WHERE is_active = true;

CREATE INDEX idx_staff_role ON staff(role) WHERE is_active = true;
CREATE INDEX idx_staff_department ON staff(department) WHERE is_active = true;
CREATE INDEX idx_staff_is_active ON staff(is_active);
```

### Outbox Events Table

```sql
-- Transactional Outbox Pattern (atomicity guarantee)
CREATE TYPE outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,        -- 'staff.created', 'staff.updated', 'staff.deactivated'
    routing_key VARCHAR(100) NOT NULL,       -- RabbitMQ routing key
    payload JSONB NOT NULL,                  -- Event data (JSON)
    status outbox_status_enum DEFAULT 'pending',
    retry_count SMALLINT DEFAULT 0,
    max_retries SMALLINT DEFAULT 3,
    created_at TIMESTAMP DEFAULT NOW(),
    processed_at TIMESTAMP,
    error_message TEXT                       -- Son hata mesajı (debug için)
);

CREATE INDEX idx_outbox_events_pending ON outbox_events(status, created_at) WHERE status = 'pending';
CREATE INDEX idx_outbox_events_retry ON outbox_events(status, retry_count) WHERE status = 'failed';
```

**Outbox Pattern Flow**:
```
1. Business Logic + Outbox INSERT (same transaction)
   ↓
2. Background Worker polls pending events (5s interval)
   ↓
3. Publish to RabbitMQ
   ↓
4. Mark as 'processed' or 'failed' (with retry)
```

**Why Outbox Pattern**:
- ✅ **Atomicity**: DB write + event write aynı transaction'da (ACID guarantee)
- ✅ **At-least-once delivery**: Event kaybı imkansız (DB'de persist edilir)
- ✅ **Resilience**: RabbitMQ down olsa bile event DB'de saklanır
- ✅ **Retry mechanism**: Failed events otomatik retry edilir

---

## API Endpoints

### 🔒 POST /api/v1/staff

Yeni personel kaydı (sadece teacher, admin API'den eklenemez)

**Role Requirement**: Admin only

**Request**:
```json
{
  "email": "teacher@university.edu.tr",
  "first_name": "Ayşe",
  "last_name": "Demir",
  "role": "teacher",
  "department": "Computer Science",
  "phone": "+90 555 123 4567",
  "office_location": "A Blok 204"
}
```

**Response** (201):
```json
{
  "id": "uuid",
  "email": "teacher@university.edu.tr",
  "first_name": "Ayşe",
  "last_name": "Demir",
  "role": "teacher",
  "department": "Computer Science",
  "phone": "+90 555 123 4567",
  "office_location": "A Blok 204",
  "status": "active",
  "created_at": "2025-11-11T10:00:00Z"
}
```

**RabbitMQ Event Published**:
```json
{
  "event_id": "evt_staff_created_uuid",
  "event_type": "staff.created",
  "timestamp": "2025-11-11T10:00:00Z",
  "data": {
    "id": "uuid",
    "email": "teacher@university.edu.tr",
    "first_name": "Ayşe",
    "last_name": "Demir",
    "role": "teacher",
    "department": "Computer Science"
  }
}
```

**Business Logic**:
1. Email unique kontrolü
2. Role validation (sadece "teacher" izinli - "admin" API'den eklenemez)
3. **Atomic transaction**:
   - Staff DB'ye kaydet
   - Outbox event DB'ye kaydet (same transaction - ACID guarantee)
4. Background worker outbox'tan event'i RabbitMQ'ya publish eder (5s poll interval)
5. Auth Service event'i dinler ve login için hazırlar

**Event Consumers**: Auth Service (user creation with initial password)

---

### 🔓 GET /api/v1/staff/:id

Personel detayı görüntüleme

**Role Requirement**:
- **Teacher**: Sadece kendi bilgisi (`user_id == staff_id`)
- **Admin**: Tüm personel

**Response** (200):
```json
{
  "id": "uuid",
  "email": "teacher@university.edu.tr",
  "first_name": "Ayşe",
  "last_name": "Demir",
  "role": "teacher",
  "department": "Computer Science",
  "phone": "+90 555 123 4567",
  "office_location": "A Blok 204",
  "status": "active",
  "created_at": "2025-11-11T10:00:00Z",
  "updated_at": "2025-11-11T10:00:00Z"
}
```

**Business Logic**:
- Authorization check based on role
- Teacher: only own data (`user_id == staff_id`)
- Admin: all staff data

---

### 🔒 PUT /api/v1/staff/:id

Personel bilgisi güncelleme

**Role Requirement**: Admin only

**Request**:
```json
{
  "department": "Software Engineering",
  "phone": "+90 555 999 8888",
  "office_location": "B Blok 305"
}
```

**Response** (200):
```json
{
  "id": "uuid",
  "email": "teacher@university.edu.tr",
  "department": "Software Engineering",
  "phone": "+90 555 999 8888",
  "office_location": "B Blok 305",
  "updated_at": "2025-11-11T11:00:00Z"
}
```

**RabbitMQ Event Published**:
```json
{
  "event_id": "evt_staff_updated_uuid",
  "event_type": "staff.updated",
  "timestamp": "2025-11-11T11:00:00Z",
  "data": {
    "id": "uuid",
    "changed_fields": {
      "department": "Software Engineering",
      "phone": "+90 555 999 8888",
      "office_location": "B Blok 305"
    }
  }
}
```

**Business Logic**:
1. Staff bilgilerini güncelle
2. DB update + outbox event insert (same transaction)

**Event Consumers**: Auth Service (email/department değişikliği için)

---

### 🔒 DELETE /api/v1/staff/:id

Personel silme (soft delete)

**Role Requirement**: Admin only

**Response** (200):
```json
{
  "message": "Staff deleted successfully",
  "id": "uuid"
}
```

**RabbitMQ Event Published**:
```json
{
  "event_id": "evt_staff_deactivated_uuid",
  "event_type": "staff.deactivated",
  "timestamp": "2025-11-11T12:00:00Z",
  "data": {
    "id": "uuid"
  }
}
```

**Business Logic**:
1. Soft delete: `is_active = false`, `deleted_at = NOW()`
2. DB update + outbox event insert (same transaction)
3. Auth Service event'i dinler ve kullanıcıyı devre dışı bırakır
4. Student Service event'i dinler ve orphaned öğrencilerin advisor_id = NULL yapar

**Event Consumers**: Auth Service (user deactivation)

---

### 🔒 GET /api/v1/staff

Personel listesi (pagination, filtering)

**Role Requirement**: Admin only

**Query Parameters**:
- `page` (default: 1)
- `limit` (default: 20, max: 100)
- `role` (filter by role: teacher, admin)
- `department` (filter by department)
- `is_active` (filter by active status, default: true)

**Response** (200):
```json
{
  "data": [
    {
      "id": "uuid",
      "email": "teacher@university.edu.tr",
      "first_name": "Ayşe",
      "last_name": "Demir",
      "role": "teacher",
      "department": "Computer Science",
      "status": "active"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

**Business Logic**:
- Pagination with filtering
- Only active staff by default (`is_active = true`)

---

### 🔓 GET /api/v1/staff/instructors

Bölüme göre aktif öğretim görevlisi listesi (Course Catalog Service entegrasyonu için)

**Role Requirement**: Admin, Teacher (service-to-service)

**Query Parameters**:
- `department` (required): Bölüm adı (e.g., "Computer Science")
- `is_active` (optional, default: true): Active staff filter

**Response** (200):
```json
{
  "data": [
    {
      "id": "uuid",
      "first_name": "Ayşe",
      "last_name": "Demir",
      "department": "Computer Science"
    },
    {
      "id": "uuid",
      "first_name": "Mehmet",
      "last_name": "Öz",
      "department": "Computer Science"
    }
  ]
}
```

**Business Logic**:
```sql
SELECT id, first_name, last_name, department
FROM staff
WHERE role = 'teacher'
  AND department = $1
  AND is_active = $2;  -- default: true
```

**Usage**:
- Course Catalog Service: Instructor dropdown için öğretmen listesi
- Course Catalog Service: Ders açılırken instructor_id validation
- Student Service: Advisor validation

---

## RabbitMQ Configuration

### Exchange & Routing Keys

```
Exchange: "staff.events" (type: topic)

Routing Keys:
- staff.created
- staff.updated
- staff.deactivated
```

### Event Schemas

#### staff.created
Published when: New staff created (POST /staff)

#### staff.updated
Published when: Staff data updated (PUT /staff/:id)

#### staff.deactivated
Published when: Staff soft deleted / deactivated (DELETE /staff/:id)

---

## Error Codes

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 400 | INVALID_INPUT | Validation hatası |
| 400 | INVALID_ROLE | Geçersiz role (sadece teacher izinli) |
| 403 | FORBIDDEN | Sadece admin işlem yapabilir |
| 404 | STAFF_NOT_FOUND | Personel bulunamadı |
| 409 | EMAIL_EXISTS | Email zaten kayıtlı |
| 409 | CANNOT_CREATE_ADMIN | Admin uygulama üzerinden eklenemez |
| 500 | INTERNAL_ERROR | Server hatası |

---

## Admin Kullanıcısı Ekleme

**ÖNEMLI**: İlk admin kullanıcısı **manuel olarak** DB'ye eklenir (migration veya SQL script):

```sql
-- Staff Service DB
INSERT INTO staff (id, email, first_name, last_name, role, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'admin@university.edu.tr',
    'System',
    'Administrator',
    'admin',
    'active'
);
```

**Auth Service**'de de aynı kullanıcı manuel eklenmelidir (bkz: auth-service.md).

---

## Role-Based Access Control (RBAC)

| Endpoint | Teacher | Admin |
|----------|---------|-------|
| `POST /staff` | ❌ | ✅ |
| `GET /staff/:id` | ✅ (own) | ✅ (all) |
| `PUT /staff/:id` | ❌ | ✅ |
| `DELETE /staff/:id` | ❌ | ✅ |
| `GET /staff` | ❌ | ✅ |
| `GET /staff/instructors` | ✅ | ✅ |

**Key Points**:
- **Teacher**: Can only view their own staff record
- **Admin**: Full access to all staff operations
- **Admin Creation**: Cannot be done via API, only through direct DB insertion

---

## Related Services

- **Auth Service**: Event consumer (staff.created, staff.updated, staff.deactivated)
- **Student Service**:
  - Teacher advisor validation için staff bilgilerini kullanır (REST)
  - `staff.deactivated` eventi dinler → orphaned öğrencilerin advisor_id = NULL yapılır
- **Notification Service**: Event consumer (staff.created, staff.deactivated)
  > **Not**: Notification Service tüm servislere en son eklenecektir. Şu an event'ler sadece yukarıdaki servisler tarafından consume edilmektedir.

---

**Version**: 3.0.0 (Added is_active, deleted_at, partial unique index)
**Last Updated**: 2025-12-12
**Priority**: High (Auth Service dependency)
