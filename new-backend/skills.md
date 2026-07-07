# Backend — Go Moduler Monolith (AI Talimati)

Tek binary monolith (`new-backend/monolith`) + ayri notification servisi. 9 modul: auth, staff, student, course_catalog, enrollment, attendance, grades, meal, payment. `new-backend/**` icinde calisirken bu dosya zorunlu okumadir.

> **Onemli:** Bu artik mikroservis DEGIL. Eski mikroservis kodu `v0-microservices` git tag'inde arsivli — main'de legacy kod yok.

---

## 1. Sert Kurallar (asla ihlal etme)

- **Calisma dizini**: Tum make komutlari `new-backend/monolith/` icinden. Ciplak `goose`/`sqlc generate` YAPMA — Makefile env cozumluyor.
- **DB**: Tek PostgreSQL, modul basina ayri schema (tablo adi `auth_users` gibi prefix'li) + ayri goose version tablosu (`goose_db_version_<module>`).
- **Query**: sqlc + pgx/v5 — raw SQL string YAPMA, GORM YAPMA. `internal/modules/*/db/` generated — elle DUZENLEME.
- **Migration**: goose. Uygulanmis migration'i degistirme, yeni dosya ekle. Calistirma (`migrate-up`) kullanici onayi ister.
- **Event publish**: Outbox pattern zorunlu — service transaction icinde outbox tablosuna yaz, publisher'i dogrudan cagirma.
- **Moduller arasi cagri**: In-process client interface (ornek: `enrollment/service.NewInProcessStudentClient`) — modul modulu HTTP ile CAGIRMAZ. Side-effect/notify icin RabbitMQ event.
- **sqlc rename**: Her modulun `sqlc.yaml`'inda schema prefix'i Go adindan dusuren `rename:` blogu var (`auth_user` → `User`). Yeni tablo eklerken rename satirini da ekle.
- **Yeni modul / yeni event semasi**: once kullaniciya sor (CLAUDE.md §6).

---

## 2. Dizin Yapisi (sabit)

```
new-backend/
  go.work                    # monolith + services/notification + shared
  monolith/
    cmd/main.go              # tum modul wiring + outbox worker'lar + downstream binding'ler
    internal/http/server.go  # Module interface: Name() + RegisterRoutes(rg)
    internal/eventbus/       # OutboxWorker, exchange topolojisi, DownstreamBinding
    internal/platform/       # ortak: errors, middleware, logger, database, redis, rabbitmq, handler
    internal/modules/<m>/    # module.go + dto/ repository/ service/ handler/ errors/ worker/
                             #   db/ (generated)  sql/{migrations,queries}/  sqlc.yaml
  services/notification/     # AYRI binary — RabbitMQ consumer (consumer/delivery/templates), kendi sqlc+goose
  shared/events/             # event envelope tipleri
  infrastructure/            # docker-compose.yml, seed, Caddy
```

Kanonik ornekler: **auth** (tam katman seti + consumer), **staff** (outbox dahil en sade modul).

---

## 3. Make Komutlari (`new-backend/monolith/` icinden)

```bash
make build | run | test | test-cov | lint | tidy
make sqlc-<module>            # tek modul icin generate (ornek: make sqlc-auth)
make sqlc-all
make migrate-up-<module>      # KULLANICI ONAYI ile calistir
make migrate-down-<module> | migrate-status-<module> | migrate-up-all
make migrate-create-<module> name=create_x_table
```

`DB_URL` `.env`'den gelir. Modul listesi Makefile `MODULES` degiskeninde.

---

## 4. Yeni Endpoint Workflow (sira zorunlu)

1. Migration (gerekiyorsa): `make migrate-create-<m> name=...` — tablo adi schema-prefix'li
2. Query: `internal/modules/<m>/sql/queries/*.sql` → `make sqlc-<m>` (+ sqlc.yaml rename)
3. Repository → Service → DTO → Handler → modul `errors/` sabiti
4. Route: modulun `module.go` → `RegisterRoutes` (middleware zinciri orada kurulur)
5. `make test` + `go build ./...` hatasiz → atomic commit

---

## 5. Yeni Modul Kaydi (once kullaniciya sor)

1. `internal/modules/<m>/module.go`: `Name()` + `RegisterRoutes(rg)` implement et (`internal/http/server.go` Module interface). Opsiyonel: `Bootstrap(ctx)`, `PublicRoutesProvider`.
2. `cmd/main.go`: `New(...)` → `Bootstrap` → `RegisterModules(...)` zincirine ekle.
3. Event publish ediyorsa: `eventbus.NewOutboxWorker("<m>", "<m>.events", module.OutboxStore(), ...)` goroutine'i main.go'ya.
4. Makefile `MODULES` listesine ekle.

---

## 6. Event / Outbox / Consumer

- **Publish**: Service, is transaction'i ICINDE outbox tablosuna yazar (`staff/repository/outbox_repository.go` + `outbox_store.go` pattern'i). OutboxWorker arka planda RabbitMQ'ya basar.
- **Exchange adi**: `<module>.events` — **routing key**: `<entity>.<action>` (ornek: `staff.created`, `grade.finalize.requested`).
- **Consume**: Modul `worker/` altinda EventConsumer; queue binding'i `cmd/main.go` `downstreamBindings` listesine eklenir (consumer offline'ken mesaj kaybolmasin diye pre-declared).
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

- Test dosyalari modul paketlerinin YANINDA (`service/`, `handler/`, `dto/`, `worker/`) — ayri test agaci yok.
- Isimlendirme: `TestXxx_Scenario_ExpectedResult`. Calistirma: `make test`.
- Basarisiz testi `t.Skip()` ile atlama — fix et veya rapor et, commit atma.

---

## 10. Failure Mode Tablosu

| Durum | YAP | YAPMA |
|---|---|---|
| sqlc generate hata | Query SQL'i duzelt, tekrar `make sqlc-<m>` | `db/` dosyalarini elle duzenleme |
| Migration hata | Kullaniciya goster, `migrate-down-<m>` oner | Tablo `DROP`, goose tablosu `DELETE` |
| RabbitMQ/Redis baglantisi yok (dev) | Kullaniciya compose komutunu goster (sudo gerekir) | Publish/blacklist adimini bypass etme |
| Legacy kodda (v0-microservices tag) bug fark ettin | Not et, `new-backend`'e dokunan kismi bildir | Tag icindeki kodu duzeltmeye calisma |
