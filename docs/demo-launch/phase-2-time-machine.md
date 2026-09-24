# Faz 2 — Tüm servislerde zaman makinesi

## Başlamadan
- Oku: `new-backend/skills.md`, `frontend/skills.md`.
- README §4'teki **K1** (30 dk otomatik kapanma + şerit) ve **K2** (±2 yıl
  sınır) cevaplanmış olmalı. Cevapsızsa kullanıcıya sor.

## Bağlam
- `new-backend/shared/platform/clock/clock.go` süreç içi çalışıyor ve saati
  **donduruyor**; saat akmıyor. Donmuş saatte yoklama QR'ı hiç yenilenmiyor ve
  süresiz geçerli kalıyor.
- Uçlar yalnız catalog'da (`services/catalog-service/internal/module.go` ≈147)
  ve staff'ta (`services/staff-service/internal/module.go` ≈116-119) var.
  Diğer servislerin saati hiç değişmiyor.
- Frontend sayfası tamamen sahte: `frontend/src/lib/services/system-service.ts`
  ≈84-115 (`MOCK_DELAY`, backend'e istek atmadan hep "başarılı" döner).
- Go'da ~50 `clock.Now()` var. SQL'de iş kuralı karşılaştırması yapan birkaç
  `NOW()` var; bunlar Go ile DB'nin farklı saatlere bakmasına yol açar.

## Hedef davranış
- **Kapalı:** `clock.Now() == time.Now()`.
- **Açık:** `clock.Now() = time.Now() + offset`. Saat akmaya devam eder ve tüm
  servislerde aynı ofset kullanılır (sapma ≤ 1 sn).
- **Güvenlik ve altyapı saatleri her zaman gerçek:** JWT, oturum süreleri,
  kilitler, rate limit, idempotency, outbox yeniden denemeleri, Redis TTL'leri.
  Aksi halde saat ileri alınınca herkes oturumdan düşer.
- K1 = evet ise: 30 dk sonra kendiliğinden kapanır; açıkken her API yanıtında
  başlık olur, frontend şerit gösterir.
- K2 = evet ise: |ofset| ≤ 2 yıl; aşan istek 400 alır.
- Ayarı admin rolündeki herkes (demo admin dahil) yapabilir. Gece geri dönüşü
  (Faz 7) Redis'i temizlediği için saat de gerçeğe döner.

## Görevler

- [x] **2.1 clock paketini ofset modeline çevir**
  - `shared/platform/clock/clock.go` API'si:
    - `SetOffset(offset time.Duration, until time.Time)`
    - `Reset()`
    - `Now()`
    - `State()` → active, offset, simüle an, until
  - Dondurmalı `Set(t)` kaldırılır. `until` geçtiyse `Now()` gerçek saate
    döner (her çağrıda ucuz kontrol).
  - Testler.
  - **Commit:** `refactor(shared): switch the simulated clock to an offset model`
  > Not (24.09): Testler kesin anlara (örn. tam 03:00) dondurmaya ihtiyaç
  > duyduğu için dondurma yalnız test yardımcısında kaldı:
  > `clock/clocktest.Freeze(t, at)`. Taban saati `clock/internal/source`
  > tutuyor; üretim kodu onu değiştiremez. `State()` bir `clock.Snapshot`
  > döner. `time_handler.go` 2.5'e kadar yeni API'ye geçici olarak uyarlandı.

- [ ] **2.2 Redis ile servisler arası senkron**
  - Yeni paket `shared/platform/clocksync`:
    - Redis anahtarı `clock:state` (JSON: `offset_seconds`, `until`, `set_at`,
      `set_by`), pub/sub kanalı `clock:changed`.
    - `Start(ctx, client)`: açılışta anahtarı oku ve clock'a uygula; kanala
      abone ol; kaçan mesajlar ve yeniden bağlanma için her 10 sn'de bir
      anahtarı yeniden oku.
    - `Publish(ctx, state)` / `Clear(ctx)`: anahtarı yaz (TTL = until) ve
      kanala yayınla.
  - `shared/bootstrap/bootstrap.go`: Redis varsa `clocksync.Start(rt.Ctx, ...)`.
    Redis yoksa gerçek saat kullanılır (fail-safe). notification servisi
    bootstrap kullanmıyor ve saate ihtiyacı yok.
  - `platform/redis` wrapper'ına gerekirse pub/sub yöntemleri ekle.
  - Test: Redis'i interface arkasına alıp sahte implementasyonla test et.
    **miniredis gibi yeni bir kütüphane eklemek için kullanıcıya sor.**
  - **Commit:** `feat(shared): sync the simulated clock across services via Redis`

- [ ] **2.3 Gerçek saatte kalması gerekenler**
  - Şunlarda `clock.Now()` → `time.Now()`:
    - `shared/platform/utils/jwt.go:77`, `:112`
    - `services/auth-service/internal/service/auth_service.go`:
      `generateAccessToken` (≈817), `generateRefreshToken` (≈839), oturum
      `expiresAt` (≈151, ≈469, ≈565), şifre sıfırlama süresi (≈611)
    - `shared/platform/rabbitmq/publisher.go:59` (envelope Timestamp)
  - Diğer tüm `clock.Now()` kullanımlarını gözden geçir
    (`grep -rn "clock.Now()" new-backend --include=*.go | grep -v _test`):
    - token, oturum, rate limit, idempotency, outbox, Redis TTL → `time.Now()`
    - iş kuralı (dönem, son tarih, rezervasyon, yoklama, not, ödeme süresi)
      → `clock.Now()` kalır
    - Karar veremediğin yerleri görevin altına not et.
  - **Kabul:** Test: ofset +1 yıl iken login olunuyor ve token doğrulanıyor.
  - **Commit:** `fix(shared): keep security timestamps on the real clock`
    (+ servis bazında gerekirse)

- [ ] **2.4 SQL'deki iş saati karşılaştırmaları**
  - Aşağıdaki sorgularda `NOW()` → sqlc parametresi (`sqlc.arg(now)`); Go
    tarafı `clock.Now()` geçer:
    - `services/attendance-service/internal/sql/queries/attendance_sessions.sql:16`
      (`expires_at > NOW()`) ve `:44` (`expires_at < NOW()`)
    - `services/catalog-service/internal/sql/queries/semesters.sql:25`
      (`hard_deadline < NOW()`)
    - `services/meal-service/internal/sql/queries/reservations.sql:93` ve `:104`
      (pending ve expired zaman aşımı)
  - Gerçek saatte kalanlar: `created_at` / `updated_at` varsayılanları,
    `synced_at`, outbox `next_retry_at`, auth `sessions.sql:37`.
  - Her serviste `make sqlc`.
  - **Commit:** servis başına, örn.
    `fix(attendance): compare session expiry against the service clock`

- [ ] **2.5 Uçlar**
  - **Durum:** her servis
    `GET /api/<prefix>/admin/time/status` (JWT + admin) sunar. Prefix'ler:
    `auth, staff, students, catalog, enrollment, attendance, grades, meals, payments`.
    Tercihen merkezi mount: `shared/httpserver/server.go` `RegisterModules`.
  - **Ayar (yalnız catalog):**
    - `POST /api/catalog/admin/time/simulate` gövde `{ "time": "<RFC3339>" }`
      → ofset = time − now. K2 sınırı uygulanır; K1 ise until = now + 30 dk.
    - `POST /api/catalog/admin/time/reset`
    - İkisi de `clocksync.Publish` / `Clear` çağırır.
  - `shared/platform/handler/time_handler.go`'yu yeniden yaz. İngilizce hata
    mesajlarını (≈41) Türkçeleştir.
  - staff'taki `timeAdmin` bloğunu kaldır; catalog'daki mount'u yeni düzene
    taşı.
  - simulate/reset işlemlerini catalog audit log'una yaz.
  - Handler testleri.
  - **Commit:** `feat(catalog): expose cluster-wide time machine endpoints`

- [ ] **2.6 Görünürlük (K1)**
  - Shared middleware: saat simüle ise her yanıta `X-Simulated-Time` ve
    `X-Simulated-Until` başlıklarını ekle. Bunları
    `shared/platform/middleware/cors.go`'daki `Access-Control-Expose-Headers`'a
    ekle.
  - Frontend: ky `afterResponse` bu başlıkları bir context'e yazsın. Layout'ta
    şerit: "Zaman makinesi aktif: {tarih saat}, {n} dk sonra gerçek saate döner".
  - Mobil: `mobile/services/api.ts` interceptor'ü ve ekran üstünde küçük şerit.
    Zaman yetmezse not düş.
  - **Commit:** `feat(frontend): show a banner while the time machine is active`
    (+ mobil)

- [ ] **2.7 Zaman makinesi sayfası**
  - `frontend/src/lib/services/system-service.ts` (≈27-115):
    - `getAllTimeStatuses` → her servisin status ucu (`*ApiSafe` istemcileri)
    - `simulateTimeAll` → catalog simulate'e tek çağrı
    - `resetTimeAll` → catalog reset
    - `MOCK_DELAY` ve sahte dönüşleri sil.
    - `SERVICES` haritasında meal'in `timePath: "time"` değerini `"admin/time"`
      yap; payment'ı ekle.
  - `frontend/src/pages/admin/system/time/`: kalan süreyi, servis bazında
    saati ve servisler arası sapma uyarısını göster.
  - **Commit:** `feat(frontend): wire the time machine page to the backend`

- [ ] **2.8 e2e**
  - `.github/workflows/ci.yml` `backend-e2e` job'u (≈114-255) golden path'ine
    ekle: simulate → iki farklı servisin status'u aynı simüle saati gösterir
    → reset → gerçek saat.
  - **Commit:** `test(infra): cover the time machine in the e2e job`

## Faz sonu
- README §5 yeşil olmalı.
- Push edilirse CI e2e yeşil olmalı (push için izin iste).
- README §1 madde 6.
