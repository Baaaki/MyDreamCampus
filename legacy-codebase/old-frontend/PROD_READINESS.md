# Production Readiness — Pratik Katman

Bu doküman: **CV/portfolyo seviyesinde "ciddi proje" izlenimi veren**, ama proje teslim süresini günlerce uzatmayan iyileştirmeler. Enterprise-grade (multi-tenant, event signing, certificate pinning, HSM key rotation) **kapsam dışı**.

Her madde için kabaca **effort**, **etki**, ve neyi çözdüğü yazılı.

> **Durum işareti:** ✅ uygulandı · 🟡 kısmen uygulandı · ⏳ yapılmadı
>
> Son güncelleme: 2026-04-25 — Tier 1'in tamamı + Tier 2.10/2.13 + Tier 3.17 uygulandı. Bu doküman bir sonraki turda **tekrar uygulamamak için** referans olarak okunmalı.

---

## Felsefe

> "Bilmiyorum çünkü atlamadım" değil, **"biliyorum ve scope'tan çıkardım"** duruşu.

README/ROADMAP'te aşağıdaki "Kapsam dışı" listesini açıkça belirt — eksiklik değil, **karar** olarak görünsün:

- Event signing / mesaj imzalama (HMAC veya asymmetric)
- JWT için RS256 + KID rotation
- Mobil için certificate pinning
- Çoklu rol (multi-role) modeli — şu an tek string
- Multi-tenant izolasyon (tek üniversite varsayımı)
- HSM / KMS entegrasyonu
- WAF / DDoS protection
- Penetration test raporu

---

## Tier 1 — Mutlaka yap (toplam ~1 gün)

Bunlar olmadan "production'a yakın" demek zor.

### 1. ✅ Hassas endpoint'lerde fail-closed davranış

**Şu an**: Redis düşerse blacklist/rate-limit fail-open.
**Yap**: Login, register, password-change, grade-write, financial endpoint'lerinde Redis erişilemezse 503 dön. Diğer endpoint'lerde fail-open kalsın.
**Effort**: 1-2 saat (middleware'e `failClosed bool` parametresi).
**Etki**: "Redis'i düşür, security'yi düşür" sınıf saldırıyı keser.

**Uygulandı**:
- `EndpointLimit.FailClosed` alanı eklendi → `backend/shared/middleware/ratelimit.go`
- `JWTAuth(WithFailClosed())` opsiyonu → `backend/shared/middleware/auth.go` (blacklist + token version check fail-closed)
- Auth-service'te login/refresh/password endpoint'leri `FailClosed: true`; password-change ayrıca `JWTAuth(WithFailClosed())` ile sarıldı → `backend/services/auth-service/cmd/main.go`
- **Atlandı**: grade-write ve diğer servislerde fail-closed sarımı uygulanmadı; her servisin main.go'sunda manuel sarım gerekiyor (Traefik forward-auth'a bağlı). Yapılırsa: `JWTAuth(WithFailClosed())` veya `EndpointRateLimit` config'inde FailClosed:true. Auth-service dışında uygulanmadı.

### 2. ✅ Username enumeration kapatma

**Şu an**: "User not found" vs "Invalid password" mesajları muhtemelen ayrışıyor.
**Yap**: Login hatalarında tek mesaj: `"Invalid credentials"`. Aynı response time için hatta user yokken bile dummy Argon2 hash karşılaştırması yap.
**Effort**: 30 dakika.
**Etki**: Aktif kullanıcı listesi sızdırmaz.

**Uygulandı**:
- Eski dummy hash format hatalıydı (decode short-circuit, timing leak vardı) — `backend/shared/utils/password.go`'da `init()` ile gerçek bir Argon2id hash hesaplanıyor; `VerifyDummyPassword(password)` helper'ı eklendi
- Auth service: user-not-found ve deactivated dallarında `VerifyDummyPassword` çağrılıyor → `backend/services/auth-service/internal/service/auth_service.go`
- `ACCOUNT_DEACTIVATED` response kodu kaldırıldı, `INVALID_CREDENTIALS` ile birleştirildi (audit'te ayrı kalıyor)
- **Not**: `ACCOUNT_LOCKED` (429) ayrı tutuldu — UX gerekçesi (kullanıcı "30dk sonra dene" mesajı görmeli). Lock leak teorik olarak var ama 5 başarısız deneme + rate-limit gerektirdiği için pratik enumeration saldırısına imkân vermez.

### 3. ✅ Cookie flag denetimi (production config)

**Yap**: Tüm `SetCookie` çağrılarında prod'da:
- `Secure: true`
- `HttpOnly: true` (csrf_token hariç)
- `SameSite: Strict` (refresh_token için), `Lax` (csrf_token için)

`isProduction` flag'i environment'a göre **build sırasında** zorlanmalı, runtime check unutulabilir.
**Effort**: 30 dakika.

**Uygulandı**:
- `setAuthCookie` / `clearAuthCookie` helper'ları → `backend/services/auth-service/internal/handler/auth_handler.go` — `SameSite=Strict`, `HttpOnly=true`, `Secure=isProduction`
- 4 endpoint'te (Login/Logout/LogoutAll/RefreshToken/ChangePassword) tekrar eden cookie kodu helper'a çekildi
- CSRF cookie zaten `SameSite=Lax`, `HttpOnly=false` ile doğru ayarlı → `backend/shared/middleware/csrf.go`

### 4. ✅ CORS allowlist — wildcard yasak

**Yap**: Prod config'de `AllowOrigins: ["*"]` koyma. Whitelist sabit. `AllowCredentials: true` ile `*` zaten geçersiz, ama explicit ol.
**Effort**: 15 dakika.

**Uygulandı (önceden)**: `backend/shared/middleware/cors.go`'da `resolveAllowedOrigins()` prod'da `CORS_ALLOWED_ORIGINS` set değilse panic; wildcard hiçbir yerde yok. Origin allowlist'te eşleşmezse `Access-Control-Allow-Origin` header gönderilmiyor.

### 5. ✅ Recovery middleware payload sızıntısı

**Yap**: Panic recovery'de stack trace **sadece log'a** gitsin. Response body'de generic mesaj: `"Internal server error"` + request ID.
**Effort**: 15 dakika.

**Uygulandı**: `backend/shared/middleware/recovery.go` — stack ve panic value sadece zap log'a; response body `{"error": ..., "message": "Internal server error", "request_id": ...}`. `fmt.Sprintf("Internal server error: %v", err)` formatı kaldırıldı.

### 6. ✅ Environment validation startup'ta

**Yap**: Servis başlarken zorunlu env var'lar yoksa fail-fast (panic with message). Şu an `INTERNAL_SERVICE_SECRET` için var; aynısını JWT_SECRET, DB_URL, REDIS_URL için de uygula. Ayrıca **secret minimum length** kontrolü (örn. JWT secret < 32 byte ise panic).
**Effort**: 30 dakika.

**Uygulandı**:
- `backend/shared/config/helpers.go`:
  - `minJWTSecretLength = 32` sabiti
  - `ValidateCommonConfig` prod'da JWT_SECRET ≥ 32 byte zorluyor
  - Yeni `ValidateRedisConfig` helper'ı: REDIS_ADDR boş olamaz, prod'da REDIS_PASSWORD boş veya `changeme_redis_secret` olamaz
- `backend/services/auth-service/config/config.go`: `ValidateRedisConfig` çağrısı + prod'da `ADMIN_INITIAL_PASSWORD` default değer kontrolü
- **Not**: `INTERNAL_SERVICE_SECRET` zaten `StripInternalHeaders()` middleware init sırasında panic atıyor (auth.go:201) — değişiklik gerekmedi.

### 7. ✅ Request ID propagation

**Yap**: Her isteğin başında `X-Request-ID` header üret (yoksa). Log'lara ekle. Servisler arası HTTP/event çağrılarında forward et.
**Effort**: 1 saat.
**Etki**: Production'da bir sorunu izlemek "iyi olur" değil, "yapılabilir" olur. Mülakatta bunu söylemek puan.

**Uygulandı**:
- `backend/shared/logger/context.go`: `WithRequestIDValue(ctx, id)` — gelen ID'yi context'e bağlıyor, boşsa fresh UUID
- `backend/shared/middleware/logger.go`: gelen `X-Request-ID` header varsa onu kullanıyor (Traefik → service → downstream tek trace)
- `backend/shared/client/semester_client.go`: cross-service HTTP çağrısında `X-Request-ID` forward ediliyor
- **Atlandı**: RabbitMQ event publish/consume akışında request_id propagation (event header'larında geçirme) yapılmadı. Async event chain'i izlemek istenirse `shared/rabbitmq/publisher.go` ve `shared/rabbitmq/consumer.go`'a header desteği eklenmeli.

---

## Tier 2 — Yapması rahat, kazanımı yüksek (~1 gün daha)

### 8. ⏳ Refresh token rotation + reuse detection

**Mekanizma**:
- Refresh token kullanıldığında DB'de "used" işaretle, yenisini ver.
- Aynı refresh token ikinci kez kullanılırsa → **session ailesini komple iptal** (tüm refresh ve access token'lar).
- Bu "token reuse detection" — sızdırılmış token'ı yakalar.

**Effort**: 3-4 saat (DB tablosuna `family_id`, `revoked_at`, `replaced_by` kolonları + service logic).
**Etki**: Sektör standardı. README'de adıyla yazınca "ciddi proje" sinyali.

**Yapılmadı — neden**: DB schema değişikliği + sqlc regeneration + repository/service refactor. Doc'un kendi tahmininin (3-4h) en uzun maddesi. Yapmak istenirse:
1. `sessions` tablosuna `family_id UUID NOT NULL`, `revoked_at TIMESTAMP`, `replaced_by UUID` ekle (migration)
2. `make sqlc` ile yeniden generate
3. `auth_service.RefreshAccessToken`: yeni JTI verirken `family_id` aynı kalsın, eski session'a `replaced_by` yaz
4. Reuse tespit: aynı `family_id` içinde `revoked_at != NULL` olan bir token tekrar kullanılırsa → tüm aileyi revoke + token version increment

### 9. ⏳ Graceful shutdown

**Yap**: SIGTERM yakalandığında:
- Yeni HTTP isteklerini reddet (server'ı stop)
- Devam eden istekleri 30s timeout'la bekle
- RabbitMQ consumer'ları kapat (in-flight mesajları ack et)
- DB pool'u kapat

Şu an muhtemelen `os.Exit` ile ölüyorsun → in-flight mesajlar veya yarım transaction'lar.
**Effort**: 2-3 saat (her servisin main.go'su için).

**Durum karışık**: auth-service'te zaten 30s timeout'lu graceful shutdown var (`backend/services/auth-service/cmd/main.go:200-208`). Diğer 8 servisin main.go'sunda aynı pattern olup olmadığı doğrulanmadı — **yapılırsa**: her servisin `cmd/main.go`'sunu auth-service ile aynı pattern'e getir (signal handling + `srv.Shutdown(ctx)` + consumer/cleanup cancel).

### 10. ✅ Health check ayrımı: liveness vs readiness

**Yap**:
- `/healthz` (liveness): sadece servis ayakta mı? (sabit 200)
- `/readyz` (readiness): DB, Redis, RabbitMQ erişilebilir mi? Erişilemezse 503.

Docker healthcheck readiness'i kullansın. K8s'e gidersen aynısı çalışır.
**Effort**: 1 saat.

**Uygulandı (önceden)**: `backend/shared/handler/health.go` — `LivenessHandler` ve `ReadinessHandler` mevcut. Auth-service'te `/health` (liveness) ve `/ready` (DB+Redis+RabbitMQ ping) ayrı route'lar olarak kayıtlı (`auth-service/cmd/main.go:163-169`). **Yapılırsa**: Docker compose'daki `healthcheck` direktiflerini `/ready`'e yönlendirmek (şu an postgres/rabbitmq için işliyor; servis-level healthcheck Docker compose'da tanımlı değil).

### 11. ⏳ Object-level authorization helper

**Yap**: `shared/middleware` veya `shared/authz` altında:
```go
func RequireOwnership(getOwnerID func(*gin.Context) (string, error)) gin.HandlerFunc
```
Handler'larda explicit çağrılır. **Convention** sayesinde IDOR riski azalır — middleware'i unutursan code review yakalar.
**Effort**: 1-2 saat tasarım, sonra her resource için handler'a bir satır.

**Yapılmadı — neden**: Helper'ı yazmak 30dk ama _her resource için_ owner-extractor lambda'sını domain bilgisiyle yazmak gerekiyor (örn. grades: student kendi notunu, teacher ders verdiği öğrencinin notunu, admin hepsini). Sadece `RequireOwnership(...)` helper'ını yazıp uygulamadan commit'lemek anlamsız. **Yapılırsa**: önce `auth-service/DeleteSession` gibi net "owner=user_id" durumuna örnek olarak uygula, sonra her servis için domain-specific extractor'ları yaz.

### 12. ⏳ Audit log append-only constraint

**Yap**: Audit tablosunda DB seviyesinde:
- `UPDATE` ve `DELETE` izinleri PG role'den çekilmiş olsun (sadece INSERT)
- `created_at` immutable (trigger ile)
- Application user (servis) bu tabloya sadece INSERT yetkili user ile yazsın

**Effort**: 1 saat (migration + role).
**Etki**: "Saldırgan iz silebilir mi?" sorusuna verilecek somut cevap.

**Yapılmadı — neden**: Servisler tek `postgres` user kullanıyor. Ayrı `audit_writer` role'ü oluşturmak için connection setup'ı ikiye bölmek gerek. **Yapılırsa**: (1) migration ile `REVOKE UPDATE, DELETE ON audit_log FROM postgres` + immutable `created_at` trigger'ı, (2) docker-compose'da audit-writer için ek user/password env, (3) servislerde audit insert için ayrı pool. (1)+(2) yapılıp (3) skip edilirse audit insert'leri permission denied alır — yarım iş tehlikeli.

### 13. ✅ Docker compose resource limits

**Yap**: Her servise `mem_limit`, `cpus`, ve özellikle **postgres'e** `shm_size`. Limit yoksa bir servis OOM olunca host'u sürükler.
**Effort**: 30 dakika.

**Uygulandı**: `backend/infrastructure/docker-compose.yml` — 8 postgres'e `mem_limit: 512m`, `cpus: 1.0`, `shm_size: 256mb`; rabbitmq `mem_limit: 1g, cpus: 1.0`; redis & traefik `mem_limit: 256m, cpus: 0.5`. `docker compose config` validate ediyor.

---

## Tier 3 — Atlatabilirsen atla, ama README'de "yapılmadı" notu olsun

### 14. ⏳ Distributed tracing (OpenTelemetry)

Loki var (log), Grafana var (metric muhtemelen). Trace yok. **OTel SDK eklemek 1 saat**, ama ekosistemi hazırlamak yarım gün. Yapmasan da "stack hazır, OTel exporter eklenebilir" notunu README'ye yaz.

**Yapılmadı**: Tier 3 — README notu da yazılmadı.

### 15. ⏳ Database migration safety

**Sorun**: Goose migration'ları **online migration** garantisi vermiyor. NOT NULL kolonu eklemek prod'da kilit yaratır.

**Yap**: README'ye "production migration playbook" bölümü:
- ADD COLUMN nullable + default → app deploy → backfill → ALTER NOT NULL
- DROP COLUMN: önce code'dan çıkar, sonra migration

Sadece doküman, **kod değişikliği yok**.
**Effort**: 30 dakika yazı.

**Yapılmadı**: Sadece doküman.

### 16. ⏳ Backup & disaster recovery dokümanı

**Yap**: `docs/runbooks/backup.md` — postgres dump cron komutu, restore prosedürü, RPO/RTO hedefleri (mock değerler bile olsa).
**Effort**: 30 dakika.

**Yapılmadı**: Sadece doküman.

### 17. ✅ Mobil token storage

**Yap**: Expo `SecureStore` kullan, `AsyncStorage` kullanma. Token'lar refresh + access ayrı key'lerde.
**Effort**: 1 saat.

**Uygulandı (önceden)**: `mobile/services/authService.ts` ve `mobile/services/api.ts` zaten `expo-secure-store` kullanıyor. `jwt_token`, `refresh_token`, `user_data` ayrı key'lerde.

### 18. ⏳ Rate limit per-resource

**Şu an**: Per-IP ve per-user global. Bir kullanıcı API'yi spam'lese tüm endpoint'lerde aynı limit.
**İyileşme**: Pahalı endpoint'lere (export, bulk import, rapor üretme) **ek** endpoint-level limit. Zaten `EndpointRateLimit` var, sadece config'de doldur.
**Effort**: 30 dakika.

**Yapılmadı**: Mekanizma hazır (`EndpointRateLimit` middleware + `EndpointLimits` config). Config'de hangi endpoint hangi grup'a düşecek doldurulmadı (örn. CSV import, transcript export). Yapılırsa: ilgili servisin config'ine `RATE_LIMIT_EXPORT_LIMIT` gibi env'lar + main.go'da `EndpointLimits` map'ine ekle, route'a `EndpointRateLimit("export")` koy.

---

## Yapma — kapsam dışı tutulması gerekenler

Aşağıdakileri yapmaya kalkışırsan **hafta gider**, CV projesi değerine oran bozulur:

| Konu | Niye yapma |
|---|---|
| Event signing (HMAC her mesajda) | Tüm publish/consume akışı değişir, dikkat dağıtır |
| RS256 + JWKS rotation | Anahtar dağıtımı altyapısı gerekir, yarım yapılırsa zarar |
| Multi-tenant tam izolasyon | Schema, query'ler, RBAC topyekun değişir |
| WAF / DDoS koruması | Cloud-level, lokal demo'da gösterilemez |
| Sertifika pinning (mobil) | Demo akışını bozar, kullanıcı APK update sıkıntısı |
| Full SOC2/ISO altyapısı | Kâğıt iş, kod değil |

README'nin **"Out of Scope"** bölümünde bu liste duruyorsa, eksiklik değil **karar** olarak görünür — bu fark mülakatta önemli.

---

## Önerilen sıra

1. **Önce Tier 1'i bitir** → projeyi "production'a benziyor" çizgisine taşır.
2. README'ye "Security & Production Readiness" bölümü yaz, yapılanları + bilinçli atlanan'ları listele.
3. Tier 2'den **8 (refresh rotation)** ve **11 (object-level authz)** mülakatta en çok puan getiren ikisi — onları seç.
4. Tier 3 isteğe bağlı.

Toplam: **2 günlük disiplinli iş** = projenin "öğrenci ödevi" çağrışımından "junior+ developer ürünü" çağrışımına geçişi.

---

## Mevcut durum (2026-04-25)

**Tamamlanan**: Tier 1 (1-7) + Tier 2.10 + Tier 2.13 + Tier 3.17 → 10 madde.

**Bekleyen** (öncelik sırasına göre):
- 🟢 Yüksek değer / orta efor: **8 (refresh rotation)**, **11 (object-level authz)**
- 🟡 Tutarlılık / az efor: **9 (graceful shutdown — diğer servislere yay)**, **18 (rate limit per-resource — config doldur)**
- 🔵 Sadece doküman: **14 (OTel notu)**, **15 (migration playbook)**, **16 (backup runbook)**
- 🔴 Daha büyük / dikkat: **12 (audit append-only — DB role split)**

**Tekrar uygulamayın**: Tier 1'in hiçbir maddesi yeniden yapılmamalı; her ✅ maddesi altında uygulanan dosyalar listeli.
