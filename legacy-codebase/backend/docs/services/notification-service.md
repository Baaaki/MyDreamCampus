# Notification Service

## Sorumluluk
E-posta, SMS, push notification gönderimi

**Non-Critical Service**: Asenkron, queue'da bekleyebilir

---

## İletişim

### Inbound (RabbitMQ - Event Consumers)
Tüm önemli eventleri dinler:
- `enrollment.program_submitted` → "Ders programınız danışman onayına gönderildi"
- `enrollment.program_approved` → "Ders kaydınız onaylandı"
- `enrollment.program_rejected` → "Ders programınız reddedildi"
- `grade.submitted` → "Notunuz girildi"
- `attendance.marked` → "Yoklamanız alındı" (optional)
- `attendance.failed` → "Devamsız durumundasınız"
- `meal.reservation.batch_confirmed` → "Yemek rezervasyonunuz onaylandı"
- `payment.completed` → "Ödemeniz alındı"
- `payment.failed` → "Ödeme başarısız oldu"

### Outbound
- **SMTP Server** → E-posta gönderimi (REST/SMTP)
- **SMS Gateway** → SMS gönderimi (REST) - Opsiyonel

---

## Database Schema

```sql
-- Notification templates
CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) UNIQUE NOT NULL,
    channel VARCHAR(50) NOT NULL,
    subject VARCHAR(255),
    template TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Notification log
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    recipient_id UUID NOT NULL,
    recipient_email VARCHAR(255),
    recipient_phone VARCHAR(50),
    channel VARCHAR(50) NOT NULL,
    subject VARCHAR(255),
    message TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'pending',
    sent_at TIMESTAMP,
    error_message TEXT,
    retry_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_notifications_recipient ON notifications(recipient_id);
CREATE INDEX idx_notifications_status ON notifications(status);
CREATE INDEX idx_notifications_event ON notifications(event_type);
```

---

## Event to Notification Mapping

### enrollment.program_approved
**Template**:
```
Subject: Course Enrollment Confirmed
Body:
Dear {{student_name}},

Your course enrollment program has been approved by your advisor.

Total courses: {{course_count}}
Semester: {{semester}} {{academic_year}}

Best regards,
University Student Information System
```

---

### grade.submitted
**Template**:
```
Subject: Grade Posted for {{course_code}}
Body:
Dear {{student_name}},

Your grade for {{course_code}} - {{course_name}} has been posted.

Letter Grade: {{letter_grade}}

You can view your detailed grades at: https://debis.university.edu.tr/grades

Best regards,
University Student Information System
```

---

### attendance.failed
**Template**:
```
Subject: Attendance Failure Notice
Body:
Dear {{student_name}},

You have failed the course {{course_code}} due to insufficient attendance.

Attendance Rate: {{attendance_rate}}%
Required: 70%

This will be reflected in your transcript as FF (Devamsız).

Best regards,
University Student Information System
```

---

### meal.reservation.batch_confirmed
**Template**:
```
Subject: Meal Reservations Confirmed
Body:
Dear {{student_name}},

Your weekly meal reservations have been confirmed.

Total meals: {{meal_count}}
Total amount: {{total_amount}} TRY

Best regards,
University Student Information System
```

---

### payment.completed
**Template**:
```
Subject: Payment Received
Body:
Dear {{student_name}},

We have received your payment.

Amount: {{amount}} {{currency}}
Payment Type: {{payment_type}}
Transaction ID: {{transaction_id}}

Thank you!

Best regards,
University Student Information System
```

---

### payment.failed
**Template**:
```
Subject: Payment Failed
Body:
Dear {{student_name}},

Unfortunately, your payment could not be processed.

Amount: {{amount}} {{currency}}
Error: {{error_message}}

Please try again or contact support.

Best regards,
University Student Information System
```

---

## Business Logic

### Event Handler
1. Event'i consume et
2. Event type için template bul
3. Recipient bilgilerini al (student email from event data)
4. Template'i event data ile render et
5. Notification kaydı oluştur
6. E-posta gönder (SMTP)
7. Status güncelle (sent/failed)
8. Başarısız ise retry schedule et (exponential backoff)

### Retry Logic
- Max 3 retry
- Exponential backoff: 2^retry_count minutes (1, 2, 4 minutes)
- Retry count aşılınca status = "permanently_failed"

---

## API Endpoints (Optional - Admin)

### GET /api/v1/notifications/student/:studentId
Öğrencinin bildirim geçmişi

**Response** (200):
```json
{
  "student_id": "uuid",
  "notifications": [
    {
      "id": "uuid",
      "event_type": "enrollment.program_approved",
      "channel": "email",
      "subject": "Course Enrollment Confirmed",
      "status": "sent",
      "sent_at": "2025-11-09T10:02:00Z"
    }
  ]
}
```

---

### POST /api/v1/notifications/retry/:id
Başarısız bildirimi tekrar gönder

**Role Requirement**: Admin

**Response** (200):
```json
{
  "message": "Notification retry scheduled",
  "notification_id": "uuid"
}
```

---

## RabbitMQ Configuration

### Exchange & Routing Keys
```
Subscribed Exchanges:
- "enrollment.events" (routing keys: enrollment.*)
- "grade.events" (routing keys: grade.*)
- "attendance.events" (routing keys: attendance.*)
- "meal.events" (routing keys: meal.*)
- "payment.events" (routing keys: payment.*)

Queue binding: All events (#)
```

---

## Error Codes

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 400 | INVALID_TEMPLATE | Template syntax hatası |
| 404 | TEMPLATE_NOT_FOUND | Event için template yok |
| 500 | SMTP_ERROR | Email gönderim hatası |
| 500 | INTERNAL_ERROR | Server hatası |

---

## Related Services

- **All Services**: Event consumer (enrollment, grade, attendance, meal, payment events)

---

**Version**: 3.0.0 (Simplified documentation)
**Last Updated**: 2025-11-18
