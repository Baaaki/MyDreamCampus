# Auth Service

## Sorumluluk
Kimlik doğrulama, JWT token yönetimi (access + refresh token), şifre yönetimi, session yönetimi

**Minimalist Yaklaşım**: Bu servis SADECE authentication işlemlerinden sorumludur. Kullanıcı kaydı Student Service ve Staff Service tarafından yapılır.

---

## İletişim

### Inbound (RabbitMQ)
- `student.created` → Student Service'ten yeni öğrenci event'i (Auth DB'ye ekler)
- `staff.created` → Staff Service'ten yeni personel event'i (Auth DB'ye ekler)
- `student.updated` → Öğrenci bilgisi güncellemesi (email, department değişikliği)
- `staff.updated` → Personel bilgisi güncellemesi (email, department değişikliği)
- `student.deactivated` → Öğrenci soft delete (is_active = false, token'lar revoke)
- `staff.deactivated` → Personel soft delete (is_active = false, token'lar revoke)

### Outbound
Yok (Auth Service sadece authentication yapar, event yayınlamaz)

---

## Database Schema

### Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,                    -- Student/Staff Service'ten gelen UUID
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,    -- Argon2id ile hashlenmiş
    role VARCHAR(50) NOT NULL,              -- student, teacher, admin
    department VARCHAR(100),                -- Full string: "Computer Science", "Medicine", etc.
    is_active BOOLEAN DEFAULT TRUE,         -- false = login engellenir
    token_version INT DEFAULT 1,            -- Token revocation için (increment = tüm tokenlar invalid)
    force_password_change BOOLEAN DEFAULT TRUE, -- İlk login'de şifre değiştirme zorunlu
    failed_login_attempts INT DEFAULT 0,    -- Rate limiting için
    locked_until TIMESTAMP NULL,            -- Account lockout zamanı
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL               -- Soft delete timestamp
);

-- Unique constraint only for active users (soft delete support)
CREATE UNIQUE INDEX idx_users_email_unique
    ON users(email) WHERE is_active = true;

CREATE INDEX idx_users_role ON users(role) WHERE is_active = true;
CREATE INDEX idx_users_department ON users(department) WHERE is_active = true;
CREATE INDEX idx_users_is_active ON users(is_active);
```

### Sessions Table (Opsiyonel - Multi-device tracking)

```sql
CREATE TABLE sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_jti VARCHAR(255) UNIQUE NOT NULL,  -- JWT ID
    device_info VARCHAR(255),               -- "Chrome on Windows", "Safari on iPhone"
    ip_address VARCHAR(45),                 -- IPv4 veya IPv6
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    last_used_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_jti ON sessions(refresh_token_jti);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
```

### Processed Events Table (Idempotency için)

```sql
CREATE TABLE processed_events (
    event_id VARCHAR(255) PRIMARY KEY,      -- Event'in unique ID'si
    event_type VARCHAR(100) NOT NULL,       -- student.created, staff.created, vb.
    processed_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_processed_events_type ON processed_events(event_type);
CREATE INDEX idx_processed_events_processed_at ON processed_events(processed_at);
```

**ÖNEMLI**:
- `id` otomatik generate edilmez, Student/Staff Service'ten gelir (admin hariç)
- `role` bilgisi event'ten alınır
- Kullanıcı ekleme endpoint'i YOK (event-driven, admin seed hariç)
- `token_version` increment edildiğinde tüm aktif tokenlar invalid olur
- `is_active = false` olan kullanıcılar login olamaz (okuldan ayrılanlar)
- `deleted_at` soft delete için kullanılır, hard delete yapılmaz
- Department bilgisi full string olarak saklanır: "Computer Science", "Electrical Engineering", "Medicine", "Law", "Business", etc.

---

## Admin Seed (Initial Setup)

Sistem ilk kurulduğunda en az bir admin kullanıcı olmalıdır. Bu kullanıcı **migration veya seed script** ile oluşturulur.

### Seed SQL

```sql
-- Admin seed (ilk kurulumda çalıştırılır)
-- Password: Admin123! (production'da değiştirilmeli)
INSERT INTO users (
    id,
    email,
    password_hash,
    role,
    department,
    is_active,
    token_version,
    force_password_change,
    failed_login_attempts,
    created_at,
    updated_at
) VALUES (
    '00000000-0000-0000-0000-000000000001',  -- Sabit UUID (Staff Service ile aynı)
    'admin@university.edu.tr',
    '$argon2id$v=19$m=65536,t=3,p=4$...',    -- Argon2id hash of 'Admin123!'
    'admin',
    NULL,                                     -- Admin'in department'ı yok
    true,
    1,
    true,                                     -- İlk login'de şifre değiştirmeli
    0,
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;
```

### Go Seed Function

```go
func (s *AuthService) SeedAdmin() error {
    // Admin zaten var mı kontrol et
    exists, err := s.repo.AdminExists()
    if err != nil {
        return err
    }
    if exists {
        log.Info("Admin user already exists, skipping seed")
        return nil
    }
    
    // Default password hash'le
    defaultPassword := os.Getenv("ADMIN_INITIAL_PASSWORD") // veya "Admin123!"
    hash, err := argon2id.Hash(defaultPassword)
    if err != nil {
        return err
    }
    
    admin := &User{
        ID:                  uuid.MustParse("00000000-0000-0000-0000-000000000001"), // Staff Service ile aynı
        Email:               os.Getenv("ADMIN_EMAIL"), // veya "admin@university.edu.tr"
        PasswordHash:        hash,
        Role:                "admin",
        Department:          nil,
        IsActive:            true,
        TokenVersion:        1,
        ForcePasswordChange: true,
    }
    
    if err := s.repo.CreateUser(admin); err != nil {
        return err
    }
    
    log.Info("Admin user seeded successfully")
    return nil
}
```

### Environment Variables (Production)

```bash
ADMIN_EMAIL=admin@university.edu.tr
ADMIN_INITIAL_PASSWORD=SecureRandomPassword123!
```

**Güvenlik Notları**:
- Production'da `ADMIN_INITIAL_PASSWORD` environment variable'dan alınmalı
- Admin ilk login'de şifresini değiştirmek zorunda (`force_password_change: true`)
- Seed sadece admin yoksa çalışır (idempotent)
- Admin UUID'si sabit tutulabilir veya random generate edilebilir

---

## Session Cleanup (Cron Job)

Expired session'ları temizlemek için periyodik cleanup gereklidir:

```sql
-- Her saat çalışacak cleanup job
DELETE FROM sessions WHERE expires_at < NOW();

-- Eski processed_events kayıtlarını temizle (30 günden eski)
DELETE FROM processed_events WHERE processed_at < NOW() - INTERVAL '30 days';
```

**Go Implementation**:
```go
func (s *AuthService) StartCleanupScheduler() {
    ticker := time.NewTicker(1 * time.Hour)
    go func() {
        for range ticker.C {
            s.repo.CleanupExpiredSessions()
            s.repo.CleanupOldProcessedEvents(30 * 24 * time.Hour)
        }
    }()
}
```

---

## Security Configuration

### Password Policy

```yaml
password_policy:
  min_length: 8
  max_length: 128
  require_uppercase: true
  require_lowercase: true
  require_digit: true
  require_special: false          # Opsiyonel, UX için zorunlu değil
  common_password_check: true     # "123456", "password" gibi yaygın şifreler engellenir
```

### Rate Limiting

```yaml
rate_limiting:
  login:
    # Email bazlı (brute force koruması)
    email_max_attempts: 5         # 5 başarısız deneme per email
    email_window_minutes: 15      # 15 dakika içinde
    email_lockout_minutes: 30     # 30 dakika hesap kilidi
    
    # IP bazlı (distributed attack koruması)
    ip_max_attempts: 20           # 20 başarısız deneme per IP
    ip_window_minutes: 15         # 15 dakika içinde
    ip_block_minutes: 60          # 1 saat IP bloğu
  
  refresh:
    max_requests: 10              # 10 refresh isteği
    window_minutes: 1             # 1 dakika içinde
  
  password_change:
    max_requests: 3               # 3 şifre değiştirme
    window_minutes: 60            # 1 saat içinde
```

**Rate Limiting Stratejisi**:
- Email bazlı limit: Tek bir hesaba yönelik brute force saldırılarını engeller
- IP bazlı limit: Credential stuffing ve distributed saldırıları engeller (daha yüksek threshold, NAT arkasındaki meşru kullanıcıları korumak için)
- Her iki limit de bağımsız çalışır, biri tetiklenirse erişim engellenir

### Token Configuration

```yaml
tokens:
  access_token:
    expiry_minutes: 15            # Kısa ömür, revocation gerekmez
    algorithm: HS256
  
  refresh_token:
    expiry_hours: 24              # 24 saat
    algorithm: HS256
    rotation: true                # Her refresh'te yeni token üretilir
```

---

## API Endpoints

### 🌐 POST /api/v1/auth/login

Kullanıcı girişi ve JWT token üretimi

**Role Requirement**: Herkese açık (unauthenticated)

**Request**:
```json
{
  "email": "student@university.edu.tr",
  "password": "SecurePass123!"
}
```

**Response** (200):
```http
Set-Cookie: refresh_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=86400
```
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900,
  "user": {
    "id": "uuid",
    "email": "student@university.edu.tr",
    "role": "student",
    "department": "Computer Science"
  },
  "force_password_change": false
}
```

**Response - Şifre Değişikliği Gerekli** (200):
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900,
  "user": {
    "id": "uuid",
    "email": "student@university.edu.tr",
    "role": "student",
    "department": "Computer Science"
  },
  "force_password_change": true,
  "message": "İlk girişinizde şifrenizi değiştirmeniz gerekmektedir."
}
```

**Cookie Attributes**:
- `HttpOnly`: JavaScript erişemez (XSS protection)
- `Secure`: Sadece HTTPS (production)
- `SameSite=Strict`: CSRF protection
- `Path=/api/v1/auth`: Sadece auth endpoint'lerine gönderilir
- `Max-Age=86400`: 24 saat (refresh token ömrü)

**Business Logic**:
1. Rate limit kontrolü (email ve IP ayrı ayrı kontrol edilir)
2. Account lockout kontrolü (`locked_until` geçmiş mi?)
3. Email ile kullanıcı bul
4. **Aktif kullanıcı kontrolü**: `is_active = false` ise login engelle
5. Argon2 ile password verify
6. Başarısızsa `failed_login_attempts` increment, gerekirse lockout uygula
7. Başarılıysa `failed_login_attempts` sıfırla
8. Access token (15 dakika) + Refresh token (24 saat) üret
9. Session tablosuna kaydet (device info, IP)
10. `force_password_change` durumunu response'a ekle

**JWT Access Token Payload**:
```json
{
  "user_id": "uuid",
  "role": "student",
  "department": "cs",
  "token_version": 1,
  "exp": 1234567890,
  "iat": 1234567890
}
```

**JWT Refresh Token Payload**:
```json
{
  "user_id": "uuid",
  "jti": "unique-token-id",
  "token_version": 1,
  "exp": 1234567890,
  "iat": 1234567890
}
```

**JWT Claims Açıklaması**:
- `user_id`: User primary key (Student/Staff Service'ten gelen UUID)
- `role`: student, teacher, admin (authorization için kullanılır)
- `department`: Kullanıcının bölümü (department-based authorization için)
- `token_version`: Token revocation kontrolü için
- `jti`: JWT ID - refresh token'ı benzersiz tanımlar (session tracking)
- `exp` (expiration): Token geçerlilik süresi (Unix timestamp)
- `iat` (issued at): Token oluşturulma zamanı (Unix timestamp)

---

### 🔓 POST /api/v1/auth/logout

Kullanıcı çıkışı ve refresh token iptal etme

**Role Requirement**: Refresh token cookie gerekli (access token opsiyonel)

**Request**:
```http
Cookie: refresh_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```
```json
{}
```

**Response** (200):
```http
Set-Cookie: refresh_token=; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=0
```
```json
{
  "message": "Successfully logged out"
}
```

**Business Logic**:
1. Cookie'den refresh token'ı al
2. Token'ı decode et (signature verify - expired olsa bile kabul et)
3. Token'dan `jti` ve `user_id` çıkar
4. Sessions tablosundan ilgili session'ı sil
5. Cookie'yi sil (Max-Age=0)

**Not**: Access token expire olmuş kullanıcılar da logout yapabilmeli. Bu nedenle sadece refresh token yeterlidir.

**Güvenlik Notu**: Logout işlemi sonrası, mevcut access token (varsa) 15 dakika boyunca teorik olarak kullanılabilir. Bu kabul edilebilir bir trade-off'tur çünkü:
- Access token zaten kısa ömürlü (15 dakika)
- Her servisin blacklist kontrolü yapması performans açısından maliyetli
- Kritik operasyonlar için `logout-all` endpoint'i mevcuttur

---

### 🔓 POST /api/v1/auth/logout-all

Tüm cihazlardan çıkış (Token Version increment)

**Role Requirement**: Authenticated (Student, Teacher, Admin)

**Request**:
```http
Authorization: Bearer <access_token>
```
```json
{}
```

**Response** (200):
```http
Set-Cookie: refresh_token=; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=0
```
```json
{
  "message": "Successfully logged out from all devices"
}
```

**Business Logic**:
1. JWT token'dan `user_id` al
2. `token_version` increment et (atomic operation)
3. Redis'e yeni version'ı yaz (distributed validation için)
4. Sessions tablosundan user'ın tüm session'larını sil
5. Current cookie'yi sil
6. User tekrar login olmak zorunda (tüm cihazlarda)

**Önemli**: Bu endpoint çağrıldığında:
- Tüm access token'lar anında invalid olur (`token_version` uyuşmaz)
- Tüm refresh token'lar anında invalid olur
- Hesap ele geçirildiyse saldırganın tüm erişimi kesilir

---

### 🌐 POST /api/v1/auth/refresh

Yeni access token alma (refresh token ile) - **Token Rotation dahil**

**Role Requirement**: Herkese açık (refresh token cookie gerekli)

**Request**:
```http
Cookie: refresh_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```
```json
{}
```

**Response** (200):
```http
Set-Cookie: refresh_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.NEW_TOKEN...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=86400
```
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Business Logic**:
1. Rate limit kontrolü
2. Cookie'den refresh token'ı al
3. Refresh token'ı decode et ve verify et
4. **Token version kontrolü**: JWT'deki `token_version` == DB'deki `token_version` mi?
5. Sessions tablosunda `jti` var mı kontrol et (logout edilmiş mi?)
6. **Refresh Token Rotation**:
   - Eski session'ı sil (eski `jti` ile)
   - Yeni refresh token üret (yeni `jti` ile)
   - Yeni session oluştur
   - Yeni refresh token'ı cookie olarak set et
7. Yeni access token üret (güncel `token_version` ve `department` ile)

**Token Rotation Güvenlik Avantajları**:
- Çalınmış refresh token sadece bir kez kullanılabilir
- Token reuse detection: Eski token kullanılmaya çalışılırsa, muhtemel token theft tespit edilir
- Her refresh'te token değiştiği için, saldırı penceresi minimize edilir

**Token Reuse Detection** (Opsiyonel Güvenlik Katmanı):
Eğer sessions tablosunda olmayan bir `jti` ile refresh denenirse ve bu `jti` daha önce kullanılmış (processed) ise, bu token theft göstergesidir. Bu durumda:
1. User'ın tüm session'larını sil
2. `token_version` increment et
3. 401 döndür ve kullanıcıyı uyar

---

### 🔓 POST /api/v1/auth/change-password

Şifre değiştirme (authenticated user)

**Role Requirement**: Authenticated (Student, Teacher, Admin)

**Request**:
```http
Cookie: refresh_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
Authorization: Bearer <access_token>
```
```json
{
  "old_password": "OldPass123!",
  "new_password": "NewSecurePass456!"
}
```

**Response** (200):
```http
Set-Cookie: refresh_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.NEW_TOKEN...; HttpOnly; Secure; SameSite=Strict; Path=/api/v1/auth; Max-Age=86400
```
```json
{
  "message": "Password changed successfully",
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_in": 900
}
```

**Business Logic**:
1. Rate limit kontrolü
2. JWT token'dan `user_id` al
3. Eski şifre doğru mu kontrol et
4. Yeni şifre policy'ye uygun mu kontrol et
5. Yeni şifre Argon2 ile hashle
6. DB'yi güncelle:
   - `password_hash` = yeni hash
   - `force_password_change` = false
   - `token_version` increment (atomic)
7. Redis'e yeni `token_version` yaz
8. Tüm session'ları sil (sessions tablosundan)
9. **Yeni session oluştur** (current device için)
10. **Yeni access token üret** (güncel `token_version` ile)
11. **Yeni refresh token üret** ve cookie olarak set et

**Güvenlik + UX Dengesi**:
- `token_version` increment edildiği için tüm eski token'lar (tüm cihazlarda) anında invalid olur
- Kullanıcıya yeni token'lar verildiği için mevcut oturumu kesintisiz devam eder
- Hesap ele geçirildiyse saldırganın erişimi anında kesilir (15 dakika beklemeye gerek yok)
- Kullanıcı tekrar login olmak zorunda kalmaz

---

### 🔓 GET /api/v1/auth/sessions

Aktif oturumları listele

**Role Requirement**: Authenticated (Student, Teacher, Admin)

**Request**:
```http
Authorization: Bearer <access_token>
```

**Response** (200):
```json
{
  "sessions": [
    {
      "id": "session-uuid-1",
      "device_info": "Chrome on Windows",
      "ip_address": "192.168.1.100",
      "created_at": "2025-11-18T10:00:00Z",
      "last_used_at": "2025-11-18T14:30:00Z",
      "is_current": true
    },
    {
      "id": "session-uuid-2",
      "device_info": "Safari on iPhone",
      "ip_address": "10.0.0.50",
      "created_at": "2025-11-17T08:00:00Z",
      "last_used_at": "2025-11-17T20:00:00Z",
      "is_current": false
    }
  ]
}
```

**Business Logic**:
1. JWT token'dan `user_id` al
2. Sessions tablosundan user'ın aktif session'larını çek
3. Current session'ı işaretle (JWT'deki `jti` ile eşleştir)

---

### 🔓 DELETE /api/v1/auth/sessions/{session_id}

Belirli bir oturumu sonlandır

**Role Requirement**: Authenticated (Student, Teacher, Admin)

**Request**:
```http
Authorization: Bearer <access_token>
```

**Response** (200):
```json
{
  "message": "Session terminated successfully"
}
```

**Response - Current Session** (400):
```json
{
  "error": "CANNOT_TERMINATE_CURRENT_SESSION",
  "message": "Aktif oturumunuzu sonlandırmak için logout endpoint'ini kullanın"
}
```

**Business Logic**:
1. JWT token'dan `user_id` al
2. Session'ın bu user'a ait olduğunu doğrula
3. Current session değilse sil

**Not**: Tek bir session silindiğinde, o session'ın access token'ı 15 dakika boyunca teorik olarak geçerli kalır. Bu trade-off kabul edilebilir (logout-all ile anında revocation mümkün).

---

## Token Validation (Distributed)

**ÖNEMLI**: Auth Service'te token validation endpoint'i YOK!

Her backend servisi (Student, Enrollment, Grades, vb.) JWT token'ı **kendi middleware'inde** validate eder. Bu distributed validation yaklaşımıdır.

### Neden Merkezi Validation Yok?

- **Performance**: Her request'te Auth Service'e gitmek → latency artışı
- **Single Point of Failure**: Auth Service down olursa tüm sistem durur
- **Scalability**: Auth Service bottleneck olur

### Nasıl Çalışıyor?

Tüm servisler **aynı JWT_SECRET** environment variable'ına sahip:

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Auth Service│     │Student Svc  │     │Enrollment   │
│ JWT_SECRET  │     │ JWT_SECRET  │     │JWT_SECRET   │
│ = "xyz123"  │     │ = "xyz123"  │     │= "xyz123"   │
└─────────────┘     └─────────────┘     └─────────────┘
```

### Middleware Implementation

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Token'ı header'dan al
        token := extractToken(c)
        
        // 2. JWT decode ve signature verify (JWT_SECRET ile)
        claims, err := jwt.Parse(token, jwtSecret)
        if err != nil {
            c.AbortWithStatusJSON(401, gin.H{"error": "INVALID_TOKEN"})
            return
        }
        
        // 3. Token version kontrolü (Redis cache)
        cachedVersion := redis.Get("user:version:" + claims.UserID)
        if cachedVersion != "" && claims.TokenVersion < cachedVersion {
            c.AbortWithStatusJSON(401, gin.H{"error": "TOKEN_REVOKED"})
            return
        }
        
        // 4. Context'e user bilgilerini set et
        c.Set("user_id", claims.UserID)
        c.Set("role", claims.Role)
        c.Set("department", claims.Department)
        c.Next()
    }
}
```

### Token Version Sync

Auth Service `token_version` değiştirdiğinde Redis'i günceller:

```go
// Auth Service - logout-all veya change-password sonrası
func (s *AuthService) InvalidateAllTokens(userID string) {
    // 1. DB'de token_version increment
    newVersion := s.repo.IncrementTokenVersion(userID)
    
    // 2. Redis'e yaz (diğer servisler okuyacak)
    redis.Set("user:version:"+userID, newVersion, 24*time.Hour)
    
    // 3. Sessions tablosunu temizle
    s.repo.DeleteAllSessions(userID)
}
```

---

## RabbitMQ Event Consumption

### Event Idempotency

Tüm event'ler idempotent şekilde işlenir. Her event'in unique bir `event_id`'si olmalıdır.

```go
func (s *AuthService) ProcessEvent(event Event) error {
    // 1. Event daha önce işlenmiş mi?
    exists, err := s.repo.IsEventProcessed(event.EventID)
    if err != nil {
        return err
    }
    if exists {
        // Zaten işlenmiş, skip et
        log.Info("Event already processed", "event_id", event.EventID)
        return nil
    }
    
    // 2. Transaction başlat
    tx, err := s.db.Begin()
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // 3. Event'i işle
    if err := s.handleEvent(tx, event); err != nil {
        return err
    }
    
    // 4. Event'i processed olarak işaretle
    if err := s.repo.MarkEventProcessed(tx, event.EventID, event.EventType); err != nil {
        return err
    }
    
    // 5. Commit
    return tx.Commit()
}
```

### student.created

**Event Schema**:
```json
{
  "event_id": "evt_123456789",
  "event_type": "student.created",
  "timestamp": "2025-11-11T10:00:00Z",
  "data": {
    "id": "uuid",
    "email": "student@university.edu.tr",
    "first_name": "Ahmet",
    "last_name": "Yılmaz",
    "department": "Computer Science"
  }
}
```

**Auth Service Action**:
1. Idempotency kontrolü (event_id daha önce işlenmiş mi?)
2. Initial password = email adresi (örn: "student@university.edu.tr")
   > **Portföy Projesi Notu**: Initial password olarak email adresi kullanılması güvenlik açısından ideal değildir. Production ortamında random password + email gönderimi (Notification Service) tercih edilmelidir. Bu projede Notification Service henüz implement edilmediği için basitlik amacıyla email adresi kullanılmaktadır. `force_password_change: true` ile ilk login'de şifre değişikliği zorunlu tutularak risk minimize edilmektedir.
3. Argon2id ile hashle
4. Auth DB'ye user ekle (ON CONFLICT DO NOTHING):
   - role: "student"
   - department: event'ten
   - force_password_change: true
   - token_version: 1
5. Event'i processed olarak işaretle

**SQL (Idempotent Insert)**:
```sql
INSERT INTO users (id, email, password_hash, role, department, force_password_change, token_version)
VALUES ($1, $2, $3, 'student', $4, true, 1)
ON CONFLICT (id) DO NOTHING;
```

**Güvenlik Notu**: İlk login'de şifre değiştirme zorunlu (`force_password_change: true`)

---

### staff.created

**Event Schema**:
```json
{
  "event_id": "evt_987654321",
  "event_type": "staff.created",
  "timestamp": "2025-11-11T10:00:00Z",
  "data": {
    "id": "uuid",
    "email": "teacher@university.edu.tr",
    "role": "teacher",
    "first_name": "Ayşe",
    "last_name": "Demir",
    "department": "Medicine"
  }
}
```

**Auth Service Action**: Student ile aynı flow (role event'ten alınır)

---

### student.updated / staff.updated

**Event Schema**:
```json
{
  "event_id": "evt_456789123",
  "event_type": "student.updated",
  "timestamp": "2025-11-11T12:00:00Z",
  "data": {
    "id": "uuid",
    "changed_fields": {
      "email": "new-email@university.edu.tr",
      "department": "Medicine"
    }
  }
}
```

**Auth Service Action**:
1. Idempotency kontrolü
2. Auth DB'deki user bilgilerini güncelle
3. **Email değişirse**: `token_version` increment (güvenlik önlemi)
4. Redis cache invalidate
5. Event'i processed olarak işaretle

---

### student.deactivated / staff.deactivated

Öğrenci veya personel okuldan ayrıldığında gönderilir.

**Event Schema (student.deactivated)**:
```json
{
  "event_id": "evt_789123456",
  "event_type": "student.deactivated",
  "timestamp": "2025-11-11T14:00:00Z",
  "data": {
    "id": "uuid",
    "student_number": "2021123456",
    "is_active": false,
    "deleted_at": "2025-11-11T14:00:00Z"
  }
}
```

**Event Schema (staff.deactivated)**:
```json
{
  "event_id": "evt_789123457",
  "event_type": "staff.deactivated",
  "timestamp": "2025-11-11T14:00:00Z",
  "data": {
    "id": "uuid"
  }
}
```

**Auth Service Action**:
1. Idempotency kontrolü
2. User'ı soft delete yap:
   - `is_active` = false
   - `deleted_at` = NOW()
3. `token_version` increment (tüm token'ları revoke et)
4. Redis'e yeni version'ı yaz
5. Tüm session'ları sil
6. Event'i processed olarak işaretle

**SQL**:
```sql
UPDATE users 
SET is_active = false, 
    deleted_at = NOW(), 
    token_version = token_version + 1,
    updated_at = NOW()
WHERE id = $1;

DELETE FROM sessions WHERE user_id = $1;
```

**Önemli**: Hard delete yapılmaz. Audit trail ve veri bütünlüğü için soft delete tercih edilir. Kullanıcı tekrar aktif edilebilir (örn: yanlışlıkla silindiyse).

---

## Error Codes

| HTTP Code | Error | Açıklama |
|-----------|-------|----------|
| 401 | INVALID_CREDENTIALS | Email/password yanlış |
| 401 | ACCOUNT_DEACTIVATED | Hesap deaktif edilmiş (okuldan ayrılmış) |
| 400 | WEAK_PASSWORD | Şifre policy'ye uymuyor |
| 400 | CANNOT_TERMINATE_CURRENT_SESSION | Aktif session silinemez |
| 401 | UNAUTHORIZED | Token geçersiz/expired |
| 401 | TOKEN_REVOKED | Token version uyuşmuyor (logout-all yapılmış) |
| 403 | FORCE_PASSWORD_CHANGE | Şifre değişikliği zorunlu |
| 429 | RATE_LIMIT_EXCEEDED | Çok fazla istek |
| 429 | ACCOUNT_LOCKED | Hesap geçici olarak kilitli |
| 500 | INTERNAL_ERROR | Server hatası |

---

## Redis Key Patterns

```
# Rate limiting - Email bazlı (login attempts)
rate:login:email:{email} = count [TTL: 15 dakika]

# Rate limiting - IP bazlı (login attempts)
rate:login:ip:{ip} = count [TTL: 15 dakika]

# Rate limiting - Refresh endpoint
rate:refresh:ip:{ip} = count [TTL: 1 dakika]

# Rate limiting - Password change
rate:password:{user_id} = count [TTL: 60 dakika]

# Token version (distributed validation için)
user:version:{user_id} = token_version [TTL: 24 saat]
```

---

## Department Format (Standardize)

**Department bilgisi tüm servislerde FULL STRING olarak saklanır.**

Örnek departmanlar:
- Computer Science
- Electrical Engineering
- Mechanical Engineering
- Civil Engineering
- Medicine
- Law
- Business
- Economics
- Psychology
- Biology

**Not**: Yeni bölümler eklenirken tüm servislerde tutarlı string kullanılmalıdır. Kısaltma/kod kullanılmaz.

---

## TODO / Future Enhancements

### 🔜 Şifremi Unuttum (Forgot Password)

Bu özellik şu an implement edilmemiştir. Gelecekte eklenebilir:

```
POST /api/v1/auth/forgot-password
- Email ile password reset token gönderimi
- Notification Service entegrasyonu gerekli

POST /api/v1/auth/reset-password
- Token ile yeni şifre belirleme
```

**Gereksinimler**:
- Notification Service (email gönderimi)
- Password reset token tablosu
- Token expiry (1 saat önerilir)

---

## Related Services

- **Student Service**: Event source (student.created, student.updated, student.deactivated)
- **Staff Service**: Event source (staff.created, staff.updated, staff.deactivated)
- **All Services**: JWT token validation (kendi middleware'lerinde, aynı JWT_SECRET ile)
- **Redis**: Token version cache, rate limiting
- **Notification Service**: Forgot password flow için gerekli (Phase 4)
  > **Not**: Notification Service tüm servislere en son eklenecektir. Forgot password ve email verification özellikleri Notification Service implement edildikten sonra aktif edilecektir.

---

## Implementation Checklist (Lisans Öğrencisi İçin)

### Phase 1: Core (Zorunlu)
- [ ] **Admin seed script** (ilk kurulumda admin oluşturma)
- [ ] Login/Logout endpoints
- [ ] Refresh token endpoint (rotation dahil)
- [ ] Password change endpoint
- [ ] Basic rate limiting (email + IP bazlı)
- [ ] Password policy validation
- [ ] force_password_change flag
- [ ] is_active kontrolü (login'de)
- [ ] Event idempotency (processed_events tablosu)

### Phase 2: Security Enhancement (Önerilen)
- [ ] token_version ile revocation
- [ ] logout-all endpoint
- [ ] Account lockout mechanism
- [ ] Department claim in JWT
- [ ] Session cleanup cron job
- [ ] student.deactivated / staff.deactivated event handling

### Phase 3: Advanced (Bonus)
- [ ] Sessions tablosu ve tracking
- [ ] GET /sessions endpoint
- [ ] DELETE /sessions/{id} endpoint
- [ ] Device info parsing (User-Agent)
- [ ] Token reuse detection

### Phase 4: Future (Opsiyonel)
- [ ] Forgot password flow (Notification Service gerekli)
- [ ] Email verification
- [ ] Two-factor authentication (2FA)
- [ ] Kullanıcı reaktivasyonu (soft delete geri alma)

---

**Version**: 5.0.0 (Department full string, partial unique index, portfolio notes)
**Last Updated**: 2025-12-12

## Changelog

### v5.0.0 (2025-12-12)
- **Department Format**: Standardize - full string kullanımı (kısaltma/kod yerine)
- **Email Unique Index**: Partial index ile soft delete desteği
- **Admin UUID**: Staff Service ile senkronize edildi
- **Portfolio Note**: Initial password = email açıklaması eklendi
- **Notification Service**: Future integration notu eklendi

### v4.3.0 (2025-12-04)
- **Admin Seed**: İlk kurulumda admin kullanıcı oluşturma mekanizması eklendi
- **Soft Delete**: `is_active` ve `deleted_at` alanları eklendi
- **Deactivation Events**: `student.deactivated` ve `staff.deactivated` event handling eklendi
- **Login Kontrolü**: Deaktif kullanıcılar login olamıyor (`ACCOUNT_DEACTIVATED` error)

### v4.2.1 (2025-12-04)
- **Password Change**: Şifre değişikliğinde yeni access token ve refresh token döndürülüyor (kullanıcı logout edilmiyor)

### v4.2.0 (2025-12-04)
- **JWT Claims**: `department` claim eklendi (department-based authorization için)
- **Error Codes**: `INVALID_CREDENTIALS` 400 → 401 olarak düzeltildi
- **Logout**: Access token zorunluluğu kaldırıldı (sadece refresh token yeterli)
- **Refresh Token Rotation**: Her refresh'te yeni token üretimi eklendi
- **Password Change**: Current session korunuyor, sadece diğer session'lar sonlandırılıyor
- **Event Idempotency**: `processed_events` tablosu ve idempotent event processing eklendi
- **Session Cleanup**: Cron job documentation eklendi
- **Rate Limiting**: Email ve IP bazlı ayrı limitler netleştirildi
- **Department Codes**: Standardize edilmiş kod listesi eklendi
- **Security Trade-offs**: Logout sonrası access token durumu açıkça dokümante edildi