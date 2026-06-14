# MyDreamCampus — Bitirme ve Production Yol Haritasi

Bu dosya projenin **acik kaynak olarak yayimlanmasi** ve **production'da calismasi** icin
yapilmasi gerekenleri listeler. Mimari ve altyapi mekanizmalari (outbox, DLQ, JWT,
TanStack Query, Expo Router vb.) **zaten tamamlanmistir** — burada sadece eksik akislar,
cilalanmasi gereken yerler ve deploy hazirligi vardir.

---

## BOLUM 1 — Bitirme Listesi (feature completion)

Projenin "calisir ve tam" kabul edilmesi icin kapatilmasi gereken eksik akislar.

### Backend

- [ ] **Password reset / forgot password akisi**
  - `ForcePasswordChange` flag'i JWT'de tanimli ama handler yok.
  - Email token'li reset akisi + `/auth/forgot-password` ve `/auth/reset-password` endpoint'leri.
  - Email gonderimi icin servis sec (SMTP, Resend, vb.) — mock ile baslanabilir.

- [x] **Event consumer context propagation** ✅ 2026-04-25
  - `attendance` ve `grades` consumer'larinda `context.Background()` yerine
    `Start(ctx)` root context'i closure ile yakalandi.

- [x] **Health endpoint dependency check** ✅ 2026-04-25
  - `shared/handler/health.go` ile `LivenessHandler` (proses up) ve
    `ReadinessHandler` (DB/Redis/RabbitMQ ping). Tum servislerde `/health`
    ve `/ready` ayri.

- [x] **Graceful shutdown timeout** ✅ 2026-04-25
  - Tum 8 servis 30 saniye. `SHUTDOWN_TIMEOUT_SECONDS` default'u da 30s.

- [ ] **Student import goroutine context**
  - `student-service/.../import_service.go` CSV import icin `context.Background()` ile
    goroutine aciyor. Request iptal olursa import devam eder.
  - Service-level context veya dedicated import worker context'i kullan.
  - Not: Aslinda CSV import'unun request scope'unu asmasi istenen davranis.
    Sadece graceful shutdown sinyaline bagli olmasi yeterli.

- [x] **INTERNAL_SERVICE_SECRET default kaldirilmali** ✅ 2026-04-25
  - `shared/middleware/auth.go` `StripInternalHeaders()` init'inde env yoksa panic.
  - `shared/config/helpers.go` viper default'u kaldirildi.

### Frontend

- [x] **Auto-refresh token akisi** ✅ 2026-04-25
  - `lib/api-client.ts` `afterResponse` hook 401'de single-flight `/auth/refresh`
    cagirisi yapiyor, sonra original request `X-Refresh-Retry` header'i ile bir
    kez yeniden deneniyor. Refresh fail = logout + login redirect.

- [ ] **Form validation altyapisi**
  - `react-hook-form` + `zod` kur.
  - Paylasilan `<FormField>` wrapper'i yaz.
  - 30+ form'u zamanla migrate et — yeni form'lar dogrudan yeni pattern ile.
  - Ilk once auth (login, register, password reset) ve student/staff create form'larini
    gec.

- [x] **Route-level code splitting** ✅ 2026-04-25
  - `routes.tsx` admin/teacher/student sayfalari `lazy()` + `<Suspense>`.
  - Auth sayfalari (login vb.) eager kaldi — cold-start landing.
  - Bundle: 982KB tek -> 448KB ana + 50+ lazy chunk.

- [x] **Global error boundary** ✅ 2026-04-25
  - `components/error-boundary.tsx` class component, `main.tsx` root'a wrap.
  - Fallback UI + "Tekrar dene" / "Ana sayfa" CTA. Dev modunda hata mesaji gosteriyor.

- [x] **404 sayfasi** ✅ (zaten mevcuttu)
  - `pages/not-found.tsx` rol-bazli home redirect ile var. `routes.tsx` catch-all
    bunu kullaniyor. Roadmap yazilirken atlanmis.

### Mobile

- [x] **Refresh token entegrasyonu** ✅ 2026-04-25
  - Backend `LoginResponse` ve `RefreshResponse`'a `refresh_token` body alani eklendi
    (mobile'in cookie jar'i yok). `/auth/refresh` body fallback kabul ediyor.
  - Mobile `services/api.ts` axios interceptor 401'de single-flight refresh + retry.

- [x] **401 auto-navigate to login** ✅ 2026-04-25
  - Refresh fail durumunda SecureStore temizleniyor + `onUnauthorized` callback
    AuthContext'e user=null verir, mevcut `AuthGuard` (`_layout.tsx`) `(auth)/login`'e
    redirect ediyor. Ayrica `router.push` cagrisi gereksiz.

- [ ] **Environment config per build**
  - `app.json` yerine `app.config.ts` kullanarak dev / staging / prod ayrimi.
  - `EXPO_PUBLIC_API_URL` degerini her build icin override edilebilir yap.
  - `eas.json` dosyasi olustur, en az `development` ve `production` profilleri.

- [x] **QR tarama retry** ✅ 2026-04-25
  - `QRScannerModal.tsx` parse fail'de `handledRef` 1.5s lock + auto-reset.
  - Camera ayni QR'i her frame'de yeniden firelamayacak, user modali kapatmadan
    tekrar deneyebiliyor.

- [x] **Toast race condition** ✅ 2026-04-25
  - `setTimeout(showNext, 200)` ve closure'dan okunan `visible` kaldirildi.
  - `showingRef` (synchronous) + `useEffect(visible)` ile dismiss-driven advance.
  - Hizli tetiklemede toast kaybolmuyor.

---

## BOLUM 2 — Production Hazirligi

Projeyi gercek bir sunucuda, gercek kullanicilarla calistirmak icin.

### Altyapi ve DevOps

- [ ] **Production docker-compose dosyasi**
  - `infrastructure/docker-compose.prod.yml` — development'tan ayri.
  - `restart: unless-stopped`, resource limit'leri (`mem_limit`, `cpus`).
  - Hot-reload (air) kaldirilmis, sadece built binary.
  - Development volume mount'lari olmayacak.

- [ ] **Traefik TLS / HTTPS**
  - Let's Encrypt ACME resolver ekle (`traefik.yml`).
  - Domain al (Cloudflare, Namecheap vb.) ve DNS yonlendir.
  - HTTP -> HTTPS zorunlu redirect middleware.

- [ ] **Secret management**
  - `.env` dosyalari commit'lenmemeli — `.gitignore`'a ekle.
  - Her servis icin `.env.example` birak (bos degerlerle).
  - Production'da `docker secrets` veya harici secret manager (SOPS, Vault, Doppler).
  - `JWT_SECRET`, `ADMIN_INITIAL_PASSWORD`, DB password'leri **asla repo'ya girmeyecek**.

- [ ] **Database backup stratejisi**
  - 9 Postgres instance'i icin gunluk `pg_dump` + 7 gun retention.
  - Basit cron job veya `pgbackrest`.
  - Volume snapshot (eger cloud provider destekliyorsa).
  - Restore prosedurunu test et, README'de belgele.

- [ ] **Log aggregation**
  - Loki + Promtail config zaten `infrastructure/` altinda var, baglanmamis.
  - Docker container log'larini Promtail'e yonlendir.
  - Grafana'dan sorgulanabilir hale getir.
  - Minimum: request ID ile servisler arasi trace edebil.

- [ ] **Rate limiting production degerleri**
  - Development'ta comut hic kisitlanmamis olabilir.
  - Login: 5/dk/IP, register: 3/dk/IP, genel API: 100/dk/user gibi makul degerler.
  - Redis-backed rate limiter hali hazirda var, sadece config tun.

- [ ] **Database migration stratejisi**
  - Goose zaten var ama production'da nasil calisacak?
  - Secenek 1: Container entrypoint'te `goose up` (basit).
  - Secenek 2: Ayri migration job container'i deploy oncesi.
  - Rollback testi yap — down migration'lar calisiyor mu?

- [ ] **Health check'ler Docker compose'da**
  - Her servisin `healthcheck:` blogu dependency check'li `/health` endpoint'ine baksin.
  - `depends_on: condition: service_healthy` kullan.

### Guvenlik

- [ ] **Admin initial password zorunlu degistirme**
  - `ADMIN_INITIAL_PASSWORD` env'den, ilk girisinde `ForcePasswordChange=true`.
  - Bu zaten tasarlandi, flow'un isledigini dogrula.

- [x] **Security headers middleware** ✅ 2026-04-25
  - `shared/middleware/security_headers.go`: X-Content-Type-Options, X-Frame-Options,
    Referrer-Policy, Permissions-Policy. HSTS sadece `ENVIRONMENT=production`.
  - Tum servislerin setupRouter'inda Recovery'den sonra eklendi.

- [x] **CORS allowed origins production'a daraltilmali** ✅ 2026-04-25
  - `CORS_ALLOWED_ORIGINS` env var (comma-separated) — production'da unset = panic.
  - Dev'de localhost / exp:// fallback. `CORSForMobile` da ayni env'i okuyor.

- [ ] **Inter-service iletisim**
  - Traefik internal network'u uzerinden — dis dunyaya kapali.
  - `INTERNAL_SERVICE_SECRET` production'da random generate.
  - Not: env yoksa panic eden kismi yapildi (Bolum 1), random generate hala manuel.

- [x] **Password policy** ✅ 2026-04-25
  - Backend: `shared/utils/password.go` `ValidatePasswordPolicy` (8+ chars, 1 upper/lower/digit).
  - Frontend: `frontend/src/lib/password-policy.ts` ve `mobile/lib/password-policy.ts` ayni
    kurali ayni mesajla replicate ediyor. ChangePassword form'larinda kullanildi.

- [ ] **Audit log yazma**
  - Admin islemleri (user sil, rol degistir) icin audit trail zaten tasarlanmis olabilir,
    dogrula ve Loki'ye ayri stream olarak gitsin.

### Test ve Kalite

- [ ] **Minimum test coverage**
  - Auth flow: login, logout, refresh, password change — backend integration test.
  - Enrollment create/drop — kritik domain logic.
  - JWT helper'lari — unit test.
  - Frontend: login page + auth guard — 5-10 Vitest test.
  - Hedef: kritik patikalar icin %50, toplam zorunlu degil.

- [ ] **CI pipeline**
  - GitHub Actions: her PR'da `go test`, `go vet`, `bun run lint`, `bun run build`.
  - Docker image build check.
  - CI yesil olmadan merge kapali.

- [ ] **Smoke test script**
  - `make smoke` veya `./scripts/smoke.sh` — docker compose ayaga kaldirdiktan sonra
    her servisin `/health` yanitini kontrol eder.

### Dokumantasyon ve Acik Kaynak Hazirligi

- [ ] **README.md revizyonu**
  - Ekran goruntuleri ([docs/screenshots/](docs/screenshots/) hazir).
  - "What is this?" — 2 paragraf, ne yapiyor, kim icin.
  - Architecture diagram (mermaid veya basit PNG).
  - Quickstart: `git clone` -> `make up` -> browser'da ac.
  - Tech stack listesi (badge'ler).

- [ ] **CONTRIBUTING.md**
  - Branch / commit / PR kurallari.
  - Kod stili (Uber Go guide referansi).
  - Local development setup adimlari.

- [ ] **LICENSE**
  - MIT veya Apache 2.0 — ikisinden biri.

- [ ] **.env.example'lar**
  - Her servis icin tam, aciklamali.
  - Gercek secret yerine `<your-jwt-secret>` gibi placeholder.

- [ ] **OpenAPI / Swagger (opsiyonel ama degerli)**
  - `swaggo/swag` ile annotation'li Go handler'lari otomatik spec uretir.
  - Frontend tip cikarimi icin kullanilabilir (orval).

- [ ] **Demo deployment (CV icin onemli)**
  - Kucuk bir VPS (Hetzner, DigitalOcean, ~5 EUR/ay) veya
  - fly.io / Railway free tier.
  - `demo.mydreamcampus.com` gibi bir subdomain + read-only demo kullanicilari.
  - README'de "Live Demo" linki.

- [ ] **Screenshot / demo video**
  - Ana ekranlar: login, admin dashboard, student dashboard, mobile app.
  - 30 saniyelik demo GIF veya video — README'nin en ustu.

---

## BOLUM 3 — Bilinc li Olarak Atlanan Konular

CV projesi scope'unda **kasitli olarak yapilmayacak**, gerekirse README'de belirt.
Bu liste "eksik" degil, "yazili trade-off"tur.

- OpenTelemetry / distributed tracing (structured log + request ID yeterli).
- Prometheus metrics + Grafana dashboard.
- Kubernetes manifest'leri (docker-compose yeterli, enterprise icin gerekir).
- E2E test suite (Playwright / Detox) — unit + integration ile yetinilecek.
- Biometric auth, push notifications (mobile).
- i18n — Turkce hardcoded, ingilizce cevirisi sonraki versiyon.
- Service mesh, circuit breaker.
- Event sourcing / CQRS full migration (outbox pattern yeterli).
- Payment gateway (mock kalacak).
- Notification service (ayri microservice olarak yapilmayacak, inline olacak).

---

## Onerilen Sira

**Hafta 1** — Bitirme listesi (Bolum 1)
- Backend eksik akislar + mobile refresh token + frontend form validation.

**Hafta 2** — Production altyapi (Bolum 2)
- Traefik TLS + secret management + production compose + backup.

**Hafta 3** — Acik kaynak hazirligi
- README + LICENSE + CONTRIBUTING + demo deployment + screenshot'lar.
- Minimum test coverage + CI pipeline.
- Temiz repo'ya ilk push.
