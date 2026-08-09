# Backend — Go Mikroservisler (AI Talimati)

10 ayri binary (`new-backend/services/`): auth, staff, student, catalog, enrollment, attendance, grades, meal, payment + notification. `new-backend/**` icinde calisirken bu dosya zorunlu okumadir.

> **Not:** Bu dosya Faz 4 sonrasi kismen guncellendi; tam yeniden yazim Faz 8'de. Celiskide `microservices-migration/00-BASLANGIC.md` gecerlidir.

---

## 1. Sert Kurallar (asla ihlal etme)

- **Calisma dizini**: Make komutlari servisin kendi kokunden (`services/<x>-service/`). Ciplak `goose`/`sqlc generate` YAPMA — Makefile DB_URL ve goose version tablosunu cozumluyor.
- **DB**: Tek PostgreSQL konteyneri, servis basina ayri DB + ayri DB kullanicisi. Schema adlari korunuyor (`catalog` DB'sinde `course_catalog` schema'si) + ayri goose version tablosu (`goose_db_version_<module>`).
- **Query**: sqlc + pgx/v5 — raw SQL string YAPMA, GORM YAPMA. `services/*/internal/db/` generated — elle DUZENLEME.
- **Migration**: goose. Uygulanmis migration'i degistirme, yeni dosya ekle. Calistirma (`migrate-up`) kullanici onayi ister.
- **Event publish**: Outbox pattern zorunlu — service transaction icinde outbox tablosuna yaz, publisher'i dogrudan cagirma.
- **Servisler arasi cagri**: Sync okuma/dogrulama icin **internal REST**
  (`shared/client`, `X-Internal-Secret`). In-process client adapter'lari
  kaldirildi. Side-effect/notify icin RabbitMQ event (degismedi).
- **Yeni kuyruk**: hem tuketen servisin `cmd/main.go` `DeclareQueues` listesine
  hem `infrastructure/rabbitmq/definitions.json`'a ekle. Sadece birine eklemek,
  tuketici hic ayaga kalkmadiysa mesajin sessizce kaybolmasi demek.
- **sqlc rename**: Her modulun `sqlc.yaml`'inda schema prefix'i Go adindan dusuren `rename:` blogu var (`auth_user` → `User`). Yeni tablo eklerken rename satirini da ekle.
- **Yeni modul / yeni event semasi**: once kullaniciya sor (CLAUDE.md §6).

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
  infrastructure/            # docker-compose.yml, rabbitmq/definitions.json, seed, Caddy
```

Kanonik ornekler: **auth** (tam katman seti + consumer), **staff** (outbox dahil en sade servis).

---

## 3. Make Komutlari (`new-backend/services/<x>-service/` icinden)

```bash
make build | run | test
make sqlc                     # bu servis icin generate
make migrate-up               # KULLANICI ONAYI ile calistir
make migrate-down | migrate-status
```

`DB_URL` servisin Makefile'inda; parola `SERVICE_DB_PASSWORD` env'inden gelir.
Goose version tablosu **modul adiyla** kalir (`goose_db_version_course_catalog`),
DB adiyla degil — Faz 1'de uygulanan migration gecmisiyle uyumlu olsun diye.

---

## 4. Yeni Endpoint Workflow (sira zorunlu)

1. Migration (gerekiyorsa): `internal/sql/migrations/` altina yeni goose dosyasi — tablo adi schema-prefix'li
2. Query: `internal/sql/queries/*.sql` → `make sqlc` (+ sqlc.yaml rename)
3. Repository → Service → DTO → Handler → servis `errors/` sabiti
4. Route: servisin `internal/module.go` → `RegisterRoutes` (middleware zinciri orada kurulur).
   `/internal/*` route'lari `RegisterPublicRoutes`'a, kok altina — Caddy sadece `/api`'yi proxy'liyor.
   Para/kota etkileyen POST'lara `platformMiddleware.Idempotency()` ekle.
5. `make test` + `go build ./...` hatasiz → atomic commit

---

## 5. Yeni Servis Kaydi (once kullaniciya sor)

1. `internal/module.go`: `Name()` + `RegisterRoutes(rg)` implement et (`shared/httpserver` Module interface). Opsiyonel: `Bootstrap(ctx)`, `PublicRoutesProvider`.
2. `cmd/main.go`: `bootstrap.Init(...)` → `DeclareQueues` → `New(...)` → `Bootstrap` → `rt.Run(module)`.
3. Event publish ediyorsa: `rt.StartOutbox("<x>.events", module.OutboxStore())` + `rt.StartRetention(module.RetentionStore())`.
4. `go.work`'e, `infrastructure/` compose'a ve `infrastructure/migrate/Dockerfile`'a ekle.

---

## 6. Event / Outbox / Consumer

- **Publish**: Service, is transaction'i ICINDE outbox tablosuna yazar (`staff/repository/outbox_repository.go` + `outbox_store.go` pattern'i). OutboxWorker arka planda RabbitMQ'ya basar.
- **Exchange adi**: `<module>.events` — **routing key**: `<entity>.<action>` (ornek: `staff.created`, `grade.finalize.requested`).
- **Consume**: Servisin `internal/worker/` altinda EventConsumer; `consumer.ConsumeEnvelope(ctx, queue, handler)` kullan — DLQ butcesini ve correlation id'yi o sariyor. Queue binding'i servisin `cmd/main.go` `DeclareQueues` listesine + `definitions.json`'a eklenir.
- **Idempotency**: `processed_events` tablosu — ayni event iki kere islenmez.
- Event payload degisikligi = geriye uyumsuzluk → kullaniciya sor. Consumer'lar: notification servisi + diger modullerin worker'lari.

---

## 7. Auth & Middleware (platform/middleware)

- `JWTAuth()` — normal; `JWTAuth(WithFailClosed())` — Redis erisiemezse 503 (sifre degisimi gibi kritik yollar).
- `CSRFProtection()`, `RequireRole(...)`, `RequireAdmin()`, `RequireTeacherOrAdmin()`, `RequireStudent()`.
- Rate limit: `EndpointRateLimit("login"|"refresh"|"password")` brute-force yollarda FailClosed; `UserRateLimit()`, `IPRateLimit()` global.
- Ornek zincir: `auth/module.go` RegisterRoutes.

---

## 8. Hata Standardi

- `platform/errors.AppError`: `New/Wrap/WrapWithMessage`, kontrol: `IsNotFound/IsValidation/IsUnauthorized/IsForbidden/IsConflict`.
- Modul basina `errors/` paketi sabit hata tanimlari tutar; HTTP'ye cevirme handler katmaninda.
- Kullaniciya donen mesaj Turkce, log Ingilizce (CLAUDE.md §3).

---

## 9. Test

- Test dosyalari paketlerin YANINDA (`service/`, `handler/`, `dto/`, `worker/`) — ayri test agaci yok.
- Isimlendirme: `TestXxx_Scenario_ExpectedResult`. Calistirma: servis kokunde `make test`, hepsi icin kok dizinde `make test-backend`.
- Basarisiz testi `t.Skip()` ile atlama — fix et veya rapor et, commit atma.

---

## 10. Failure Mode Tablosu

| Durum | YAP | YAPMA |
|---|---|---|
| sqlc generate hata | Query SQL'i duzelt, tekrar `make sqlc` | `internal/db/` dosyalarini elle duzenleme |
| Migration hata | Kullaniciya goster, `make migrate-down` oner | Tablo `DROP`, goose tablosu `DELETE` |
| RabbitMQ/Redis baglantisi yok (dev) | Kullaniciya compose komutunu goster (sudo gerekir) | Publish/blacklist adimini bypass etme |
| Legacy kodda (v0-microservices tag) bug fark ettin | Not et, `new-backend`'e dokunan kismi bildir | Tag icindeki kodu duzeltmeye calisma |
