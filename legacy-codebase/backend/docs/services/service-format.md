# [Service Name] Service (⚠️ CRITICAL / Priority Indicator if applicable)

## Sorumluluk
[2-3 cümle ile servisin temel sorumluluğunu açıklayın]

**[Özel Not]**: [Eğer varsa kritik bilgiler - örn: "Source of Truth", "Read-Heavy Service", "Event-Driven", "Minimalist Yaklaşım"]

## İletişim

### Inbound (RabbitMQ)
- `event.name` → [Event açıklaması]
- `another.event` → [Event açıklaması]

### Inbound (REST - Synchronous)
- **[External Service Name]** → [Ne için kullanıldığı]
  - `GET /external/endpoint` - [Açıklama]

### Outbound (RabbitMQ - Asynchronous)
- `service.event.published` → [Event açıklaması]

### Outbound (REST - Synchronous)
- **[External Service Name]** → [Ne için kullanıldığı]

## Teknoloji Stack
[Eğer servise özel teknolojiler varsa]
- JWT token generation
- Redis caching
- Argon2id password hashing
- QR kod generation
- Payment gateway integration
- vb.

---

## Database Schema

### 1. [Ana Tablo Adı]
[Tablonun amacını açıklayın]

```sql
CREATE TABLE table_name (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    field_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    deleted_at TIMESTAMP DEFAULT NULL,      -- Soft delete (NULL = active)
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Unique constraints (soft delete ile)
CREATE UNIQUE INDEX idx_table_field_unique
    ON table_name(field_name) WHERE deleted_at IS NULL;

-- Performance indexes
CREATE INDEX idx_table_status ON table_name(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_table_deleted_at ON table_name(deleted_at);
```

**Field Açıklamaları**:
- `field_name`: [Açıklama]
- `status`: [Olası değerler ve açıklamaları]

**Index Stratejisi (Performance Optimization)**:

| Index | Neden Gerekli | Kullanım Senaryosu |
|-------|---------------|-------------------|
| `field_name` | [Açıklama] | [Query örneği] |
| `status` | [Açıklama] | [Query örneği] |

**Why this approach**:
- **Performance**: [Açıklama]
- **Scalability**: [Açıklama]
- **Data Integrity**: [Açıklama]

---

### 2. [Outbox Events Tablosu - Eğer kullanılıyorsa]

**ÖNEMLI**: Event publishing atomicity garantisi için Transactional Outbox Pattern kullanılır.

```sql
-- Standard Outbox Schema (tüm servislerde aynı format kullanılır)
CREATE TYPE outbox_status_enum AS ENUM ('pending', 'processed', 'failed');

CREATE TABLE outbox_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,        -- 'resource.created', 'resource.updated'
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
- ❌ **Without Outbox**: DB commit başarılı, RabbitMQ publish fail → Data inconsistency!

---

## API Endpoints

### 🔒 `POST /api/v1/resource`
[Endpoint'in ne yaptığını açıklayın]

**Role Requirement**: [Sadece Admin / Authenticated (Student, Teacher, Admin) / Public]

**Request**:
```json
{
  "field_name": "value",
  "another_field": "value"
}
```

**Response** (201):
```json
{
  "id": "uuid",
  "field_name": "value",
  "created_at": "2025-11-11T10:00:00Z"
}
```

**Side Effect**: [Eğer varsa - örn: "`resource.created` event published to RabbitMQ"]

**Business Logic**:
1. [Adım 1 - örn: Validation]
2. [Adım 2 - örn: Database transaction]
3. [Adım 3 - örn: Event publishing]
4. [Adım 4 - örn: Cache invalidation]

**ÖNEMLI**: [Eğer kritik notlar varsa]

---

### 🔓 `GET /api/v1/resource/:id`
[Endpoint'in ne yaptığını açıklayın]

**Role Requirement**:
- **Student**: [Erişim kuralı - örn: Sadece kendi bilgisi]
- **Teacher**: [Erişim kuralı - örn: Sadece danışmanlık yaptığı kayıtlar]
- **Admin**: [Erişim kuralı - örn: Tüm kayıtlar]

**Query Parameters** (optional):
- `filter_param` (optional filter)
- `page` (default: 1)
- `limit` (default: 20, max: 100)

**Response** (200):
```json
{
  "id": "uuid",
  "field_name": "value",
  "nested_object": {
    "id": "uuid",
    "name": "value"
  }
}
```

**Authorization Logic**:
```go
func CanAccessResource(requestingUser User, resourceID uuid.UUID) bool {
    resource := getResource(resourceID)

    switch requestingUser.Role {
    case "admin":
        return true  // Admin herkesi görebilir

    case "student":
        return requestingUser.ID == resourceID  // Sadece kendi

    case "teacher":
        return resource.AdvisorID == requestingUser.ID  // Sadece danışmanlık yaptığı

    default:
        return false
    }
}
```

---

### 🌐 `GET /api/v1/public-resource`
[Public endpoint açıklaması]

**Role Requirement**: None (Public endpoint)

**Response** (200):
```json
{
  "data": []
}
```

---

## Business Logic

### [İşlem Adı] Flow
[Karmaşık business logic varsa açıklayın]

```go
func ProcessFlow(input Input) error {
    // 1. Validation
    if err := validateInput(input); err != nil {
        return err
    }

    // 2. Database transaction
    tx := db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()

    // 3. Business operations
    if err := performOperation(tx, input); err != nil {
        tx.Rollback()
        return err
    }

    // 4. Commit
    return tx.Commit().Error
}
```

**Why this approach**:
- **[Açıklama 1]**: [Reasoning]
- **[Açıklama 2]**: [Reasoning]

**Common mistakes avoided**:
- ❌ [Hata 1]: [Neden problematik]
- ❌ [Hata 2]: [Neden problematik]
- ✅ [Doğru yaklaşım]: [Neden daha iyi]

---

## RabbitMQ Events

### Published Events

#### `resource.created`
[Event'in ne zaman yayınlandığını açıklayın]

```json
{
  "event_type": "resource.created",
  "timestamp": "2025-11-11T10:00:00Z",
  "data": {
    "id": "uuid",
    "field_name": "value"
  }
}
```

**Event Consumers**:
- **[Service Name]**: [Ne için kullanır]

---

### Consumed Events

#### `external.event` Handler
[Event'i nasıl handle ettiğini açıklayın]

```go
func HandleExternalEvent(msg amqp.Delivery) {
    var event ExternalEvent
    json.Unmarshal(msg.Body, &event)

    // Process event
    processEvent(event.Data)

    msg.Ack(false)
}
```

---

### RabbitMQ Configuration

```go
// Exchange configuration
ExchangeName = "service.events"
ExchangeType = "topic"

// Routing keys
RoutingKeyCreated = "resource.created"
RoutingKeyUpdated = "resource.updated"
```

---

## Transactional Outbox Pattern (Eğer kullanılıyorsa)

### Problem Statement

**Sorun**: Standart event publishing atomik değil

```go
// ❌ PROBLEM: Two separate operations (not atomic)
db.Create(&resource)           // ✅ Success
rabbitmq.Publish(event)        // ❌ FAILS → Event lost!
// Result: Resource created in DB but consumers never notified
```

### Solution: Transactional Outbox

```go
// ✅ SOLUTION: Atomic operation
db.Transaction(func(tx *gorm.DB) error {
    // 1. Write resource (business data)
    tx.Create(&resource)

    // 2. Write event to outbox (same transaction)
    tx.Create(&outboxEvent)

    // Both succeed or both fail (ACID)
    return nil
})
// Separate background worker publishes events from outbox
```

### Background Outbox Publisher

```go
func (p *OutboxPublisher) Start(ctx context.Context) {
    ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            p.publishPendingEvents(ctx)
        case <-ctx.Done():
            return
        }
    }
}

func (p *OutboxPublisher) publishPendingEvents(ctx context.Context) {
    // Fetch unpublished events
    var events []OutboxEvent
    db.Where("published = ?", false).
        Order("created_at ASC").
        Limit(100).
        Find(&events)

    // Publish each event
    for _, event := range events {
        if err := publishToRabbitMQ(event); err != nil {
            // Retry logic
            continue
        }

        // Mark as published
        db.Model(&event).Updates(map[string]interface{}{
            "published": true,
            "published_at": time.Now(),
        })
    }
}
```

---

## Implementation Phases

### Phase 1: [Phase Name] (Faz X.X)
- [ ] [Görev 1]
- [ ] [Görev 2]
- [ ] [Görev 3]

### Phase 2: [Phase Name]
- [ ] [Görev 1]
- [ ] [Görev 2]

### Phase 3: [Phase Name]
- [ ] [Görev 1]
- [ ] [Görev 2]

---

## Error Handling

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 400 | INVALID_INPUT | Validation hatası |
| 403 | FORBIDDEN | Yetkisiz erişim |
| 404 | RESOURCE_NOT_FOUND | Kaynak bulunamadı |
| 409 | DUPLICATE_RESOURCE | Kaynak zaten var |
| 500 | INTERNAL_ERROR | Server hatası |

---

## Environment Variables

```env
# Database
DATABASE_URL=postgresql://user:pass@localhost:5432/service_db

# RabbitMQ
RABBITMQ_URL=amqp://guest:guest@localhost:5672/
RABBITMQ_EXCHANGE=service.events

# Redis (if applicable)
REDIS_URL=redis://localhost:6379

# JWT (Token validation için)
JWT_SECRET=your-super-secret-key-min-32-chars

# External Services
EXTERNAL_SERVICE_URL=http://external-service:8080

# Server
PORT=8080
```

---

## Testing Strategy

### Unit Tests
```go
// Test resource creation
func TestCreateResource(t *testing.T)

// Test authorization
func TestUserCanAccessOnlyOwnResource(t *testing.T)

// Test event publishing
func TestPublishResourceCreatedEvent(t *testing.T)
```

### Integration Tests
```bash
# Create resource
curl -X POST http://localhost:8080/api/v1/resource \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer {token}" \
  -d '{
    "field_name":"value"
  }'

# Get resource
curl -X GET http://localhost:8080/api/v1/resource/{id} \
  -H "Authorization: Bearer {token}"
```

---

## Performance Considerations

### Database Optimization
- **Indexing**: [Hangi alanlar]
- **Partial Indexes**: [WHERE deleted_at IS NULL gibi]
- **Compound Indexes**: [Multi-column indexes]
- **Query Timeout**: [Max query time]

### Caching Strategy
```
Key: "resource:{id}"
Value: Resource JSON
TTL: [Duration]
Invalidate on: [Events that trigger invalidation]
```

### Scalability
- [Stratejiler - örn: Connection pooling, Load balancing]

---

## Dependencies

```go
// go.mod
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/lib/pq v1.10.9
    github.com/rabbitmq/amqp091-go v1.8.1
    github.com/google/uuid v1.3.0
    github.com/golang-jwt/jwt/v5 v5.2.0
    // ... diğer dependencies
)
```

---

## Monitoring & Logging

### Metrics to Track
- [Metric 1 - örn: Request rate]
- [Metric 2 - örn: Error rate]
- [Metric 3 - örn: Event processing lag]

### Log Events
```go
// Event created
log.Info("Resource created",
    "resource_id", resourceID,
    "field_name", fieldName)

// Event published
log.Info("Event published",
    "event_type", "resource.created",
    "resource_id", resourceID)

// Error occurred
log.Error("Failed to process",
    "resource_id", resourceID,
    "error", err)
```

---

## Role-Based Access Control (RBAC) Summary

| Endpoint | Student | Teacher | Admin |
|----------|---------|---------|-------|
| `POST /resource` | ❌ | ❌ | ✅ |
| `GET /resource/:id` | ✅ (own) | ✅ (specific) | ✅ (all) |
| `PUT /resource/:id` | ❌ | ❌ | ✅ |
| `DELETE /resource/:id` | ❌ | ❌ | ✅ |

**Key Points**:
- **Student**: [Erişim kuralları]
- **Teacher**: [Erişim kuralları]
- **Admin**: [Erişim kuralları]

---

## Security Best Practices (Eğer gerekiyorsa)

### Input Validation
- [Validation rules]

### Rate Limiting
- [Rate limit kuralları]

### CORS Configuration
- [CORS settings]

---

## Production Readiness Checklist (Eğer gerekiyorsa)

### Functionality
- ✅ [Feature 1]
- ✅ [Feature 2]

### Performance
- ✅ [Optimization 1]
- ✅ [Optimization 2]

### Resilience
- ✅ [Resilience pattern 1]
- ✅ [Resilience pattern 2]

### Security
- ✅ [Security measure 1]
- ✅ [Security measure 2]

### Observability
- ✅ [Observability feature 1]
- ✅ [Observability feature 2]

---

**Related Services**: [Hangi servislerle ilişkili - örn: Auth Service (event consumer), Student Service (REST call)]
**Deployment**: [Faz bilgisi - örn: Faz 2.1]
**Priority**: [HIGH / MEDIUM / LOW - örn: CRITICAL (foundation service)]
**Version**: [Versiyon - örn: 1.0.0]
**Last Updated**: [Tarih - örn: 2025-11-13]
