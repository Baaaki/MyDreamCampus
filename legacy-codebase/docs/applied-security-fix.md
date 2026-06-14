# Applied Security Fixes

**Tarih:** 2026-03-22
**Kapsam:** Backend (Go microservices), Frontend (React), Mobile (React Native), Infrastructure (Docker/Traefik)

Kapsamli guvenlik auditi sonucunda tespit edilen aciklara yonelik uygulanan duzeltmeler asagida listelenistir.

---

## Ozet Tablo

| # | Fix | Ciddiyet | Katman | Durum |
|---|-----|----------|--------|-------|
| 1 | .env dosyalarini repo'dan cikar | Kritik | Genel | Atlanadi |
| 2 | CORS wildcard + credentials | Kritik | Backend | Tamamlandi |
| 3 | IDOR (GetMyAdvisees, ImportJob) | Kritik | Backend | Tamamlandi |
| 4 | CSV upload boyut limiti + sanitizasyon | Kritik | Backend | Tamamlandi |
| 5 | Cookie SameSite + Domain + Secure | Kritik | Backend | Tamamlandi |
| 6 | Open redirect | Kritik | Frontend | Tamamlandi |
| 7 | Redis authentication | Yuksek | Altyapi | Tamamlandi |
| 8 | localStorage -> httpOnly cookie | Kritik | Full-stack | Tamamlandi |
| 9 | CSRF token korunasi | Yuksek | Full-stack | Tamamlandi |
| 10 | Sifre politikasi guclendirme | Yuksek | Backend + Mobile | Tamamlandi |
| 11 | Redis blacklist fail-open | Orta | Backend | Atlanadi |
| 12 | Traefik dashboard auth | Yuksek | Altyapi | Tamamlandi |
| 13 | Rate limiting default aktif | Orta | Backend | Dogrulandi (zaten aktif) |
| 14 | CSP header | Orta | Frontend | Tamamlandi |
| 15 | TLS/HTTPS | Yuksek | Altyapi | Tamamlandi |
| 16 | Header spoofing korumasi | Yuksek | Backend | Tamamlandi |
| 17 | Timing-safe email enumeration | Yuksek | Backend | Tamamlandi |
| 18 | Access token JTI | Orta | Backend | Tamamlandi |
| 19 | Audit logging sistemi | Orta | Backend | Tamamlandi |
| 20 | Docker network segmentation | Orta | Altyapi | Tamamlandi |

---

## Detayli Aciklamalar

### #2 — CORS Wildcard + Credentials Duzeltmesi

**Dosya:** `backend/shared/middleware/cors.go`

**Sorun:** `Access-Control-Allow-Origin: *` ile `Access-Control-Allow-Credentials: true` birlikte kullaniliyordu. Bu CORS spesifikasyonunu ihlal eder ve cross-origin cookie/token hirsizligina yol acar.

**Cozum:**
- `CORS()`: Wildcard origin kaldirildi. Gelen `Origin` header'i `defaultAllowedOrigins` listesine (`http://localhost:3000`, `http://localhost:3002`) karsi kontrol ediliyor. Sadece eslesen origin'e izin veriliyor.
- `CORSWithOrigins()`: Ayni pattern, disaridan verilen origin listesiyle calisiyor.
- `CORSForMobile()`: Production'da sadece explicit origin listesi; development'ta localhost ve Expo scheme'leri. Bos Origin header'da wildcard set etme kaldirildi.
- Tum fonksiyonlara `Vary: Origin` header'i eklendi.

---

### #3 — IDOR Aciklari Duzeltmesi

**Dosya:** `backend/services/student-service/internal/handler/student_handler.go`

**Sorun:**
- `GetMyAdvisees`: `advisor_id` query parameter'dan aliniyordu — herhangi bir ogretmen baska ogretmenlerin danismanlari gorebiliyordu.
- `GetImportJobStatus`: Sahiplik kontrolu yoktu, herhangi bir kullanici herhangi bir import job'u gorebiliyordu.
- `BulkImport` ve `ListImportJobs`: Placeholder UUID (`00000000-...`) kullaniliyordu.

**Cozum:**
- `GetMyAdvisees`: `advisor_id` artik JWT context'ten (`c.Get("user_id")`) aliniyor.
- `GetImportJobStatus`: Kullanici sahipligi dogrulaniyor; admin degilse ve job'un sahibi degilse 403 donuyor.
- `BulkImport` ve `ListImportJobs`: Gercek user_id JWT context'ten cikartiliyor.

---

### #4 — CSV Upload Guvenlik Iyilestirmesi

**Dosyalar:**
- `backend/services/student-service/internal/handler/student_handler.go`
- `backend/services/student-service/internal/service/import_service.go`

**Sorun:** Dosya boyut limiti yoktu (DoS riski). CSV hucre icerikleri dogrulanmiyordu (formula injection riski).

**Cozum:**
- `http.MaxBytesReader` ile 10MB upload limiti eklendi.
- `sanitizeCSVCell()` fonksiyonu eklendi: `=`, `+`, `-`, `@`, `\t`, `\r` ile baslayan hucrelere `'` prefix'i eklenerek Excel/Sheets formula injection onleniyor.
- Sanitizasyon tum CSV kayitlarina `parseStudentRecord()` icinde uygulanıyor.

---

### #5 — Cookie Guvenlik Attributeleri

**Dosya:** `backend/services/auth-service/internal/handler/auth_handler.go`

**Sorun:** Refresh token cookie'lerinde `SameSite` ve `Domain` attribute'lari eksikti. Cookie path `/api/v1/auth` ile uyumsuzdu.

**Cozum:**
- Tum `SetCookie` cagrilarindan once `c.SetSameSite(http.SameSiteLaxMode)` eklendi.
- Cookie path `/api` olarak guncellendi (tum API endpoint'leri kapsamasi icin).
- `Secure` flag production ortaminda aktif.

---

### #6 — Open Redirect Duzeltmesi

**Dosya:** `frontend/src/pages/auth/login/index.tsx`

**Sorun:** Login sonrasi redirect parametresi yeterince dogrulanmiyordu. `?redirect=//evil.com` gibi URL'lerle kullanicilar kotu niyetli sitelere yonlendirilebiliyordu.

**Cozum:** Uc katmanli dogrulama eklendi:
```typescript
redirectTo.startsWith("/") && !redirectTo.startsWith("//") && !redirectTo.includes("://")
```

---

### #7 — Redis Authentication

**Dosyalar:**
- `backend/infrastructure/redis/redis.conf`
- `backend/infrastructure/.env`
- `backend/shared/config/helpers.go`
- `backend/infrastructure/docker-compose.yml`

**Sorun:** Redis sifresi yoktu (`requirepass` yorum satirindaydi) ve tum interface'lere acikti (`bind 0.0.0.0`).

**Cozum:**
- `requirepass changeme_redis_secret` aktif edildi.
- `bind 127.0.0.1` ile sadece localhost'a baglandi.
- `.env` ve shared config defaults guncellendi.
- Docker healthcheck'e `-a` flag'i eklendi.

> **Production notu:** `changeme_redis_secret` degerini guclu bir sifre ile degistirin.

---

### #8 — localStorage'dan httpOnly Cookie'ye Gecis

**Dosyalar:**
- `backend/services/auth-service/internal/handler/auth_handler.go`
- `backend/shared/middleware/auth.go`
- `frontend/src/lib/api-client.ts`
- `frontend/src/pages/auth/login/index.tsx`
- `frontend/src/components/auth-guard.tsx`
- `frontend/src/components/layout/header.tsx`
- `frontend/src/pages/auth/change-password/index.tsx`
- `frontend/src/pages/auth/sessions/index.tsx`

**Sorun:** JWT access token `localStorage`'da tutuluyordu. Herhangi bir XSS acigi token'in calinmasina yol aciyordu.

**Cozum:**

Backend:
- Login, RefreshToken ve ChangePassword handler'lari access token'i httpOnly cookie olarak set ediyor.
- Logout handler'lari access_token cookie'sini de temizliyor.
- Auth middleware hem Authorization header'i hem de `access_token` cookie'sini destekliyor (mobil uyumluluk).
- Access token JSON response'ta da kaliyor (mobil uygulama icin).

Frontend:
- `localStorage.setItem("access_token", ...)` tamamen kaldirildi.
- API client'a `credentials: 'include'` eklendi.
- Auth guard ve header component'leri sadece `user` bilgisini localStorage'dan okuyor (guvenlik icin degil, UI icin).
- Logout isleminde backend cagrilarak httpOnly cookie server tarafindan temizleniyor.

---

### #9 — CSRF (Cross-Site Request Forgery) Korumasi

**Dosyalar:**
- `backend/shared/middleware/csrf.go` (yeni dosya)
- `backend/shared/middleware/cors.go`
- `backend/services/*/cmd/main.go` (8 servis)
- `frontend/src/lib/api-client.ts`

**Sorun:** State-changing endpoint'lerde (POST/PUT/DELETE) CSRF korumasi yoktu.

**Cozum:** Double-submit cookie pattern uygulanadi:

Backend:
- `SetCSRFToken()`: Global middleware — non-httpOnly `csrf_token` cookie set eder (JS tarafindan okunabilir).
- `CSRFProtection()`: Protected route middleware — `X-CSRF-Token` header'inin `csrf_token` cookie'siyle eslestigini dogrular.
- GET/HEAD/OPTIONS istekleri atlanir.
- `Authorization` header'i olan istekler atlanir (mobil uygulama).
- CORS ayarlarina `X-CSRF-Token` header'i eklendi.
- 8 servise entegre edildi (payment haric — gRPC-only).

Frontend:
- `getCookie()` helper fonksiyonu eklendi.
- `beforeRequest` hook'unda CSRF token okunup `X-CSRF-Token` header'i olarak ekleniyor.

---

### #10 — Sifre Politikasi Guclendirme

**Dosyalar:**
- `backend/shared/utils/validation.go`
- `mobile/app/(auth)/register.tsx`

**Sorun:** Minimum sifre uzunlugu 8 karakterdi (backend), mobilde 6. Ozel karakter zorunlulugu yoktu.

**Cozum:**
- Minimum uzunluk 8'den **12**'ye yukseltildi.
- **4 karakter sinifinin tamami** zorunlu hale getirildi: buyuk harf, kucuk harf, rakam, ozel karakter.
- Mobil register ekraninda minimum uzunluk 6'dan **12**'ye yukseltildi ve hata mesaji guncellendi.

---

### #12 — Traefik Dashboard Korumasi

**Dosyalar:**
- `backend/infrastructure/traefik/traefik.yml`
- `backend/infrastructure/traefik/dynamic.yml`

**Sorun:** Dashboard `insecure: true` ile 8080 portunda sifresiz erisime acikti.

**Cozum:**
- `insecure: false` yapildi.
- `dynamic.yml`'e `dashboard-auth` basicAuth middleware eklendi.

> **Production notu:** `htpasswd -n admin` komutuyla guclu credential olusturun.

---

### #13 — Rate Limiting Default Aktif

**Dosya:** `backend/shared/config/helpers.go`

**Durum:** Dogrulama yapildi — `RATE_LIMIT_ENABLED` zaten `true` olarak default ayarliydi. Degisiklik gerekmedi.

---

### #14 — Content Security Policy (CSP) Header

**Dosya:** `frontend/index.html`

**Sorun:** CSP header yoktu, inline script injection'a karsi koruma bulunmuyordu.

**Cozum:** `<head>` icine CSP meta tag eklendi:
- `default-src 'self'`
- `script-src 'self' 'unsafe-inline' 'unsafe-eval'` (React dev icin gerekli)
- `style-src 'self' 'unsafe-inline' https://fonts.googleapis.com`
- `font-src 'self' https://fonts.gstatic.com`
- `img-src 'self' data: blob:`
- `connect-src 'self' http://localhost:* ws://localhost:*`

---

### #15 — TLS/HTTPS Yapilandirmasi

**Dosya:** `backend/infrastructure/traefik/traefik.yml`

**Sorun:** HTTPS tamamen devre disiydi, tum trafik HTTP uzerinden akiyordu.

**Cozum:**
- `websecure` entrypoint port 443'te eklendi.
- HTTP'den HTTPS'e otomatik redirect yapilandirildi.
- Let's Encrypt certificate resolver yorum satiri olarak eklendi.

> **Production notu:** TLS sertifikasi icin Let's Encrypt yapilandirmasi acilmali veya manuel sertifika yuklenmelidir.

---

### #16 — Header Spoofing Korumasi

**Dosyalar:**
- `backend/shared/middleware/auth.go`
- `backend/shared/config/helpers.go`

**Sorun:** `X-User-ID`, `X-User-Role` header'lari dogrulanmadan kabul ediliyordu. Traefik bypass edilirse kimlik taklit edilebiliyordu.

**Cozum:**
- `StripInternalHeaders()` middleware eklendi: Gelen isteklerdeki `X-Internal-Secret` header'ini `INTERNAL_SERVICE_SECRET` env variable ile karsilastirir. Eslesme yoksa `X-User-ID`, `X-User-Role`, `X-User-Email` header'larini siler.
- Config'e `INTERNAL_SERVICE_SECRET` default degeri eklendi.

> **Production notu:** `changeme_internal_secret` degerini guclu bir secret ile degistirin.

---

### #17 — Timing-Safe Email Enumeration Korumasi

**Dosya:** `backend/services/auth-service/internal/service/auth_service.go`

**Sorun:** Var olmayan kullanici icin hemen hata donuyordu, yanlis sifreli mevcut kullanici icin Argon2 hash karsilastirmasi yapiliyordu. Bu zamanlama farkiyla gecerli email adresleri tespit edilebiliyordu.

**Cozum:** Kullanici bulunamadığında dummy Argon2 hash dogrulamasi eklendi:
```go
utils.VerifyPassword("$argon2id$v=19$m=65536,t=3,p=4$dummysalt$dummyhash", req.Password)
```
Bu sayede her iki durumda da yaklasik ayni sure harcanir.

---

### #18 — Access Token JTI (JWT ID)

**Dosya:** `backend/shared/utils/jwt.go`

**Sorun:** Shared utils'teki `GenerateAccessTokenWithSecret` fonksiyonu JTI uretmiyordu, bu nedenle access token'lar bireysel olarak revoke edilemiyordu.

**Cozum:** Access token uretiminde `uuid.New().String()` ile JTI eklendi. Return signature `(string, string, error)` olarak guncellendi (token + JTI).

**Not:** Auth service'in kendi `generateAccessToken` metodu zaten JTI icerigini uretiyordu.

---

### #19 — Security Audit Logging Sistemi

**Dosyalar:**
- `backend/shared/audit/security.go` (yeni dosya)
- `backend/services/auth-service/internal/handler/auth_handler.go`
- `backend/services/auth-service/cmd/main.go`

**Sorun:** Guvenlik olaylari icin ozel audit log yoktu. Login denemeleri, sifre degisiklikleri, hesap kilitlemeleri takip edilemiyordu.

**Cozum:**

Yeni `shared/audit` paketi olusturuldu:
- 14 guvenlik olay tipi tanimlandi (`LOGIN`, `LOGIN_FAILED`, `LOGOUT`, `PASSWORD_CHANGE`, `ACCOUNT_LOCKED` vb.)
- Structured JSON ciktisi (ISO8601 timestamp) ile Loki/ELK uyumlu.
- `LogSecurityFromContext()` ve `LogSecurityFromContextWithDetails()` fonksiyonlari.

Auth handler'a 8 audit log noktasi eklendi:
- Basarili/basarisiz login
- Hesap kilitleme
- Logout / Logout All
- Token yenileme
- Sifre degistirme

---

### #20 — Docker Network Segmentation

**Dosya:** `backend/infrastructure/docker-compose.yml`

**Sorun:** Tum servisler tek bir bridge network uzerindeydi. Bir container ele gecirilirse tum servislere erisim mumkundu.

**Cozum:** 4 ayri network olusturuldu:
- `app-network` (bridge) — Servisler arasi iletisim
- `db-network` (internal) — PostgreSQL instance'lari
- `cache-network` (internal) — Redis
- `mq-network` (internal) — RabbitMQ

`internal: true` ile isaretlenen network'ler dis baglanti kabul etmez.

---

## Atlanilan Fixler

### #1 — .env Dosyalarinin Repo'dan Cikarilmasi
**Sebep:** Git history temizligi (`git filter-repo`) gerektirir ve tum credential'larin rotate edilmesini icerir. Bu islem ayri bir gorev olarak planlanmalidir.

### #11 — Redis Blacklist Fail-Open -> Fail-Closed
**Sebep:** Redis cokerse tum authentication'in durmasi riski var. Redis HA/replication yapilandirmasi olmadan fail-closed yapmak availability sorunlarina yol acar.

---

## Production Icin Gerekli Aksiyonlar

Bu fixler uygulandiktan sonra production'a gecmeden once:

1. **Credential Rotation:**
   - `REDIS_PASSWORD` — `changeme_redis_secret` degerini degistirin
   - `INTERNAL_SERVICE_SECRET` — `changeme_internal_secret` degerini degistirin
   - `JWT_SECRET` — Guclu, rastgele bir secret kullanin
   - `ADMIN_INITIAL_PASSWORD` — Guclu bir sifre belirleyin
   - Traefik dashboard basicAuth credential'ini olusturun

2. **TLS Sertifikasi:**
   - Let's Encrypt yapilandirmasini acin veya manuel sertifika yukleyin
   - `traefik.yml`'deki certificate resolver yorumlarini kaldirin

3. **`.env` Temizligi:**
   - Tum `.env` dosyalarini `.gitignore`'a ekleyin (zaten eklendi)
   - `git filter-repo` ile git history'den silin
   - Tum credential'lari rotate edin

4. **CSP Header Guncelleme:**
   - Production'da `unsafe-inline` ve `unsafe-eval`'i kaldirin
   - `connect-src`'deki `localhost` referanslarini production domain ile degistirin

5. **CORS Origin Listesi:**
   - `cors.go`'daki `defaultAllowedOrigins`'e production domain'i ekleyin

6. **Monitoring:**
   - Audit log'lari Loki/ELK'e yonlendirin
   - CSRF violation alert'leri kurun
   - Basarisiz login denemeleri icin alarm esigi belirleyin
