# Payment Service ⚠️ CRITICAL

## Sorumluluk
Ödeme işleme (harç, yemekhane), payment gateway entegrasyonu, transaction kayıtları, refund

**Kritik Servis**: Financial transactions, idempotency zorunlu

---

## İletişim

### Inbound (REST - Synchronous)
- **Meal Service** → Yemek rezervasyonu için ödeme başlatma
- **Enrollment Service** → Harç ödemesi için ödeme başlatma (future)

### Outbound (RabbitMQ - Asynchronous)
- `payment.completed` → Ödeme başarılı (Meal, Notification)
- `payment.failed` → Ödeme başarısız (Meal, Notification)

---

## Database Schema

```sql
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    student_id UUID NOT NULL,
    amount DECIMAL(10,2) NOT NULL,
    currency VARCHAR(3) DEFAULT 'TRY',
    payment_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    idempotency_key VARCHAR(255) UNIQUE NOT NULL,
    gateway_transaction_id VARCHAR(255),
    gateway_response TEXT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE refunds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id),
    amount DECIMAL(10,2) NOT NULL,
    reason TEXT,
    gateway_refund_id VARCHAR(255),
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_payments_student ON payments(student_id);
CREATE INDEX idx_payments_status ON payments(status);
CREATE INDEX idx_payments_idempotency ON payments(idempotency_key);
CREATE INDEX idx_refunds_payment ON refunds(payment_id);
```

---

## API Endpoints

### POST /api/v1/payments/initiate
Ödeme başlatma

**Request**:
```json
{
  "student_id": "uuid",
  "amount": 15.50,
  "currency": "TRY",
  "payment_type": "meal",
  "idempotency_key": "meal_reservation_uuid_20251109",
  "callback_url": "http://meal-service:8087/api/v1/meals/payment-callback"
}
```

**Response** (201):
```json
{
  "payment_id": "uuid",
  "status": "pending",
  "gateway_url": "https://sandbox-api.iyzipay.com/payment/...",
  "expires_at": "2025-11-09T10:15:00Z"
}
```

**Business Logic**:
1. **Idempotency check**: Aynı idempotency_key ile ödeme var mı? Varsa aynı response döndür
2. Payment kaydı oluştur
3. Iyzico gateway'e call
4. Gateway response kaydet
5. Payment URL döndür

**Security**: Idempotency key unique constraint, HMAC callback verification

---

### POST /api/v1/payments/callback
Payment gateway callback (webhook)

**Request** (from Iyzico):
```json
{
  "transaction_id": "gateway_tx_id",
  "status": "success",
  "payment_id": "uuid",
  "amount": 15.50,
  "signature": "hmac_signature"
}
```

**Response** (200):
```json
{
  "message": "Payment processed successfully"
}
```

**RabbitMQ Event Published**: `payment.completed` or `payment.failed`

**Business Logic**:
1. **Signature verification**: HMAC doğrula
2. Payment status güncelle
3. Gateway transaction ID kaydet
4. Event yayınla

**Event Consumers**: Meal Service (reservation confirm), Notification Service

---

### GET /api/v1/payments/:id
Ödeme detayı görüntüleme

**Response** (200):
```json
{
  "id": "uuid",
  "student_id": "uuid",
  "amount": 15.50,
  "currency": "TRY",
  "payment_type": "meal",
  "status": "completed",
  "gateway_transaction_id": "gateway_tx_id",
  "created_at": "2025-11-09T10:00:00Z",
  "updated_at": "2025-11-09T10:01:00Z"
}
```

---

### POST /api/v1/payments/:id/refund
İade işlemi

**Role Requirement**: Admin

**Request**:
```json
{
  "amount": 15.50,
  "reason": "Customer request"
}
```

**Response** (201):
```json
{
  "refund_id": "uuid",
  "payment_id": "uuid",
  "amount": 15.50,
  "status": "pending",
  "created_at": "2025-11-09T11:00:00Z"
}
```

**Business Logic**:
1. Payment completed status'ünde mi kontrol et
2. Refund kaydı oluştur
3. Iyzico refund API call
4. Refund status kaydet

---

### GET /api/v1/payments/student/:studentId
Öğrencinin ödeme geçmişi

**Response** (200):
```json
{
  "student_id": "uuid",
  "payments": [
    {
      "id": "uuid",
      "amount": 15.50,
      "payment_type": "meal",
      "status": "completed",
      "created_at": "2025-11-09T10:00:00Z"
    }
  ],
  "total_spent": 245.50
}
```

---

## RabbitMQ Configuration

### Exchange & Routing Keys
```
Publishing Exchange:
- "payment.events" (type: topic)

Routing Keys (Publishing):
- payment.completed
- payment.failed
```

### Event Schemas

#### payment.completed
Published when: Ödeme başarılı

```json
{
  "event_type": "payment.completed",
  "timestamp": "2025-11-09T10:01:00Z",
  "data": {
    "payment_id": "uuid",
    "student_id": "uuid",
    "amount": 15.50,
    "currency": "TRY",
    "payment_type": "meal",
    "gateway_transaction_id": "gateway_tx_id",
    "idempotency_key": "meal_abc123_20251109100000"
  }
}
```

#### payment.failed
Published when: Ödeme başarısız

```json
{
  "event_type": "payment.failed",
  "timestamp": "2025-11-09T10:01:00Z",
  "data": {
    "payment_id": "uuid",
    "student_id": "uuid",
    "amount": 15.50,
    "error_message": "Insufficient funds",
    "idempotency_key": "meal_abc123_20251109100000"
  }
}
```

---

## Error Codes

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 400 | INVALID_AMOUNT | Tutar negatif veya 0 |
| 400 | INVALID_IDEMPOTENCY_KEY | Idempotency key formatı yanlış |
| 402 | PAYMENT_FAILED | Gateway ödemeyi reddetti |
| 409 | DUPLICATE_PAYMENT | Aynı idempotency key ile ödeme var |
| 500 | GATEWAY_ERROR | Payment gateway hatası |
| 500 | INTERNAL_ERROR | Server hatası |

---

## Related Services

- **Meal Service**: Ödeme başlatma (REST inbound), event consumer (`payment.completed`)
- **Enrollment Service**: Ödeme başlatma (future)
- **Notification Service**: Event consumer (`payment.completed`, `payment.failed`)

---

**Version**: 3.0.0 (Simplified documentation)
**Last Updated**: 2025-11-18
