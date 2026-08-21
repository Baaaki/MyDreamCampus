# Backend — Go Mikroservisler (AI Talimati)

10 ayri binary (`new-backend/services/`): auth, staff, student, catalog,
enrollment, attendance, grades, meal, payment + notification. Her biri **ayri
Go modulu**, ayri konteyner, ayri veritabani. Ortak kod tek bir `shared/`
modulunde. `new-backend/**` icinde calisirken bu dosya zorunlu okumadir.

Servis/port/DB/route/event tablolari ve internal endpoint kontratlari:
[`../microservices-migration/01-REFERANS-MIMARI.md`](../microservices-migration/01-REFERANS-MIMARI.md).

---

## 1. Sert Kurallar (asla ihlal etme)

- **Calisma dizini**: Make komutlari servisin kendi kokunden
  (`services/<x>-service/`). Ciplak `goose` / `sqlc generate` YAPMA — Makefile
  DB_URL'i ve goose version tablosunu cozumluyor, ciplak komut yanlis tabloya yazar.
- **DB**: Tek PostgreSQL konteyneri, **servis basina ayri DB + ayri DB
  kullanicisi** (`auth` DB'si, `auth_svc` rolu). Bir servis baska servisin
  DB'sine baglanamaz — CONNECT yetkisi verilmemis. Cross-DB JOIN diye bir sey
  yok; baska servisin verisi ya internal REST ile okunur ya event ile
  projekte edilir.
- **Schema adlari**: DB icindeki schema adi korunuyor — tek istisna catalog,
  `catalog` DB'sinin icinde `course_catalog` schema'si. Goose version tablosu
  da schema adiyla (`goose_db_version_course_catalog`), DB adiyla degil.
- **Query**: sqlc + pgx/v5 — raw SQL string YAPMA, GORM YAPMA.
  `services/*/internal/db/` generated, elle DUZENLEME.
- **Migration**: goose. Uygulanmis migration'i degistirme, yeni dosya ekle.
  Calistirma (`make migrate-up`) kullanici onayi ister — yazma istemez.
- **Event publish**: Outbox pattern zorunlu — service transaction'i ICINDE
  outbox tablosuna yaz, publisher'i dogrudan cagirma.
- **Servisler arasi cagri**: Sync okuma/dogrulama icin **internal REST**
  (`shared/client`, `X-Internal-Secret`). Side-effect/notify icin RabbitMQ event.
- **Yeni kuyruk**: hem tuketen servisin `cmd/main.go` `DeclareQueues` listesine
  **hem** `infrastructure/rabbitmq/definitions.json`'a ekle. Sadece birine
  eklemek, tuketici hic ayaga kalkmadiysa mesajin sessizce kaybolmasi demek —
  definitions.json boot'ta yukleniyor, DeclareQueues ise servis ayaga kalkinca.
- **sqlc rename**: Her servisin `sqlc.yaml`'inda schema prefix'ini Go adindan
  dusuren `rename:` blogu var (`auth_user` → `User`). Yeni tablo eklerken
  rename satirini da ekle.
- **Yeni servis / yeni event semasi**: once kullaniciya sor (CLAUDE.md §6).

---

## 2. Dizin Yapisi (sabit)

```
new-backend/
  go.work                    # shared + 10 servis
  services/<x>-service/      # her biri AYRI Go modulu
    cmd/main.go              # bootstrap.Init -> DeclareQueues -> module -> StartOutbox/StartRetention -> Run
    go.mod  Dockerfile  Makefile  sqlc.yaml
    internal/                # module.go + dto/ repository/ service/ handler/ errors/ worker/
                             #   db/ (generated)  sql/{migrations,queries}/
  shared/                    # AYRI Go modulu, hepsi bunu import eder
    bootstrap/               # ortak acilis sirasi (config, logger, DB, Redis, Rabbit, HTTP)
    httpserver/              # Module interface: Name() + RegisterRoutes(rg)
    eventbus/                # OutboxWorker, RetentionWorker, exchange topolojisi
    client/                  # internal REST transport + circuit breaker
    contracts/               # servis sinirini gecen tipler
    events/  platform/       # envelope sabitleri; errors, middleware, logger, db, redis, rabbitmq
  infrastructure/            # docker-compose*.yml, rabbitmq/definitions.json, postgres/, migrate/, seed/
```

Kanonik ornekler: **auth** (tam katman seti + consumer), **staff** (outbox
dahil en sade servis).

`shared/` her servise **derlenerek** giriyor — orada yapilan bir degisiklik 10
imaji birden bayatlatir. Servise ozgu bir sey oraya konmaz.

---

## 3. Make Komutlari (`new-backend/services/<x>-service/` icinden)

```bash
make build | run | test
make sqlc                     # bu servis icin generate
make migrate-up               # KULLANICI ONAYI ile calistir
make migrate-down | migrate-status
```

Repo kokunden (tum stack):

```bash
make deploy          # 16 konteyner build + up
make deploy-grades   # TEK servisi rebuild + restart, digerlerine dokunmadan
make logs-grades     # tek servisin logu
make test-backend    # shared + 10 servis, -race
```

`DB_URL` servisin Makefile'inda; parola `SERVICE_DB_PASSWORD` env'inden gelir.

---

## 4. Yeni Endpoint Workflow (sira zorunlu)

1. Migration (gerekiyorsa): `internal/sql/migrations/` altina yeni goose
   dosyasi — tablo adi schema-prefix'li
2. Query: `internal/sql/queries/*.sql` → `make sqlc` (+ sqlc.yaml rename)
3. Repository → Service → DTO → Handler → servis `errors/` sabiti
4. Route: servisin `internal/module.go` → `RegisterRoutes` (middleware zinciri
   orada kurulur). `/internal/*` route'lari `RegisterPublicRoutes`'a, kok
   altina — Caddy sadece `/api`'yi proxy'liyor, `/internal` disaridan
   erisilemez olmali.
   Para/kota etkileyen POST'lara `platformMiddleware.Idempotency()` ekle.
5. Baska servisin verisi gerekiyorsa: **once** o servisin `/internal/*`
   kontratina bak (`01-REFERANS-MIMARI.md` §2). Yoksa once orada endpoint ac,
   sonra cagiran tarafta client yaz (§8).
6. `make test` + `go build ./...` hatasiz → atomic commit

---

## 5. Yeni Servis Kaydi (once kullaniciya sor)

1. `internal/module.go`: `Name()` + `RegisterRoutes(rg)` implement et
   (`shared/httpserver` Module interface). Opsiyonel: `Bootstrap(ctx)`,
   `PublicRoutesProvider`.
2. `cmd/main.go`: `bootstrap.Init(...)` → `DeclareQueues` → `New(...)` →
   `Bootstrap` → `rt.Run(module)`. `bootstrap.Options.Service` alanini doldur —
   log satirlarindaki `service` alani oradan geliyor, bos birakilirsa o
   servisin loglari Loki'de adsiz kalir.
3. Event publish ediyorsa: `rt.StartOutbox("<x>.events", module.OutboxStore())`
   + `rt.StartRetention(module.RetentionStore())`.
4. Kayit yerleri — **hepsi**: `go.work`, `infrastructure/docker-compose.yml`,
   `infrastructure/migrate/Dockerfile`, `infrastructure/postgres/init-databases.sh`
   (DB + rol), `frontend/Caddyfile` (path prefix), `.github/workflows/ci.yml`
   matrisi, `.github/workflows/cd.yml` + `dependabot.yml`.

---

## 6. Event / Outbox / Consumer

- **Publish**: Service, is transaction'i ICINDE outbox tablosuna yazar
  (`staff/repository/outbox_repository.go` + `outbox_store.go` pattern'i).
  OutboxWorker arka planda RabbitMQ'ya basar.
- **Exchange adi**: `<servis>.events` — **routing key**: `<entity>.<action>`
  (ornek: `staff.created`, `grade.finalize.requested`).
- **Consume**: Servisin `internal/worker/` altinda EventConsumer;
  `consumer.ConsumeEnvelope(ctx, queue, handler)` kullan — DLQ butcesini ve
  correlation id'yi o sariyor.
- **Kuyruk sahipligi**: Kuyrugu **tuketen** servis sahibidir; declare eden de o
  olur. Publish eden servis karsi tarafin kuyrugunu declare etmez.
  Yeni kuyruk = `DeclareQueues` + `definitions.json`, ikisi birden (§1).
- **Idempotency**: `processed_events` tablosu — ayni event iki kere islenmez.
- Event payload degisikligi = geriye uyumsuzluk → kullaniciya sor.
  Consumer'lar: notification servisi + diger servislerin worker'lari.

---

## 7. Auth & Middleware (platform/middleware)

- `JWTAuth()` — normal; `JWTAuth(WithFailClosed())` — Redis erisilemezse 503
  (sifre degisimi gibi kritik yollar).
- JWT'yi **her servis kendi dogrular**, auth'a RPC yok. Blacklist kontrolu
  `bootstrap.Init` icindeki `SetBlacklistChecker` ile kuruluyor — kendi
  `main.go`'sunda bootstrap'i atlayan bir servis logout'u sessizce gormezden gelir.
- `CSRFProtection()`, `RequireRole(...)`, `RequireAdmin()`,
  `RequireTeacherOrAdmin()`, `RequireStudent()`.
- Rate limit: `EndpointRateLimit("login"|"refresh"|"password")` brute-force
  yollarda FailClosed; `UserRateLimit()`, `IPRateLimit()` global.
- Ornek zincir: `auth/module.go` RegisterRoutes.

---

## 8. Dayaniklilik (servisler arasi cagrilar)

Servis sinirini gecen her sync cagri `shared/client` uzerinden gider.

- **Circuit breaker**: hedef servis basina bir breaker (`sony/gobreaker`).
  Ust uste 5 hatada acilir, 30 sn sonra half-open.
- **Neyin hata sayildigi** (`client/base.go` → `send`): breaker'i sadece
  **hedefin ayakta olmadigini** gosteren durumlar acar — baglanti hatasi,
  timeout, 5xx. **4xx acmaz**, `send` onu nil error ile geri verir. Olmayan bir
  ogrenci icin donen 404 hedefin saglikli oldugunun kanitidir; onu hata sayan
  breaker tek bir hatali id ile tum entegrasyonu keser. Yeni bir transport
  yolu eklerken bu ayrimi bozma.
- **Idempotency**: `platformMiddleware.Idempotency()` **sunucu** tarafinda,
  Redis'te. Para, kota veya tekrar edilemez side-effect ureten POST'lara takilir
  (rezervasyon, ders programi gonderme, danisman onayi/reddi). Ayni
  `Idempotency-Key` + ayni govde → ilk yanit; ayni key + farkli govde → 422.
  Client basligi gondermezse middleware seffaf gecer.
- **Hata map'lemesi**: HTTP client, hedefin 404'unu cagiran servisin kendi
  sentinel hatasina cevirir (staff 404 → catalog'un `ErrInstructorNotFound`'u).
  Bu kaybolursa handler yanlis status uretir ve frontend hata ekrani bozulur.
- Yeni client yazarken: interface'i **tuketen** servis tanimlar,
  `var _ XClient = (*HTTPXClient)(nil)` compile-time assertion koy, `httptest`
  birim testi yaz.

---

## 9. Hata Standardi

- `platform/errors.AppError`: `New/Wrap/WrapWithMessage`, kontrol:
  `IsNotFound/IsValidation/IsUnauthorized/IsForbidden/IsConflict`.
- Servis basina `errors/` paketi sabit hata tanimlari tutar; HTTP'ye cevirme
  **sadece** handler katmaninda.
- Hata sarmalama `fmt.Errorf("...: %w", err)`, kontrol `errors.Is/As` —
  `err.Error()` string karsilastirmasi yapma.
- Kullaniciya donen mesaj Turkce, log Ingilizce (CLAUDE.md §3).

---

## 10. Test

- Test dosyalari paketlerin YANINDA (`service/`, `handler/`, `dto/`, `worker/`)
  — ayri test agaci yok.
- Isimlendirme: `TestXxx_Scenario_ExpectedResult`. Calistirma: servis kokunde
  `make test`, hepsi icin repo kokunde `make test-backend`.
- Her HTTP client icin `httptest` birim testi zorunlu.
- Basarisiz testi `t.Skip()` ile atlama — fix et veya rapor et, commit atma.
- Servis sinirini gecen davranisi birim test yakalayamaz; onu CI'daki
  `backend-e2e` job'u (compose + golden path) yakalar.

---

## 11. Failure Mode Tablosu

| Durum | YAP | YAPMA |
|---|---|---|
| sqlc generate hata | Query SQL'i duzelt, tekrar `make sqlc` | `internal/db/` dosyalarini elle duzenleme |
| Migration hata | Kullaniciya goster, `make migrate-down` oner | Tablo `DROP`, goose tablosu `DELETE` |
| RabbitMQ/Redis baglantisi yok (dev) | Kullaniciya compose komutunu goster (sudo gerekir) | Publish/blacklist adimini bypass etme |
| Baska servisin verisi lazim | Internal REST kontratina bak, yoksa endpoint ac | O servisin DB'sine baglanmaya calisma |
| Bir servis 502 veriyor | `make logs-<servis>`, breaker acik mi bak | Breaker esigini gevsetip ustunu ortme |
| Legacy kodda (v0-microservices tag) bug fark ettin | Not et, `new-backend`'e dokunan kismi bildir | Tag icindeki kodu duzeltmeye calisma |
