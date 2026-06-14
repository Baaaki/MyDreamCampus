# MyDreamCampus — AI Asistanı Talimatları

Universite yonetim sistemi. Full-stack monorepo: Go mikroservisler + React+Vite web + React Native (Expo) mobil.

> Bu dosya **AI'a** talimattir. Kullanici dokumanlari icin bkz. `README.md`.

---

## 1. Cakisma Hiyerarsisi

Cakisma durumunda **yukaridan asagiya** dogru oncelik (1 en yuksek):

1. **Kullanici prompt'u** — en yuksek oncelik
2. **Ilgili `skills.md`** — `backend/skills.md`, `frontend/skills.md`, `mobile/skills.md`
3. **Bu dosya (CLAUDE.md)** — proje genel kurallari
4. **Memory** — gecmis konusmalardan sabitler

---

## 2. Zorunlu Okuma Kosullari

Gorev baslamadan **mutlaka oku**:

| Eger suraya dokunacaksan… | Once oku |
|---|---|
| `backend/services/**` | `backend/skills.md` |
| `backend/shared/**` | `backend/skills.md` |
| `frontend/src/**` | `frontend/skills.md` |
| `mobile/app/**`, `mobile/services/**`, `mobile/hooks/**` | `mobile/skills.md` |
| Migration / SQL / sqlc | `backend/skills.md` "Migration" bolumu |

Birden fazla katmanda degisiklik varsa **hepsini** oku.

---

## 3. Konusma Dili & Kod Dili

- **Konusma dili (kullaniciyla)**: Turkce
- **Kod, degisken, dosya adi, commit mesaji**: Ingilizce
- **Kullaniciya hata mesaji (UI text)**: Turkce
- **Log mesaji**: Ingilizce

**YAPMA:** Turkce degisken adi (`kullaniciAdi` ❌, `username` ✅).
**YAPMA:** Ingilizce kullaniciya hata mesaji (`"User not found"` ❌, `"Kullanici bulunamadi"` ✅ — UI'da).

---

## 4. Paket Yoneticisi (kritik — yanlis kullanma)

| Dizin | Komut | YAPMA | Neden |
|---|---|---|---|
| `frontend/` | `bun add`, `bun run`, `bun tsc`, `bunx --bun <x>` | `npm`, `npx`, `yarn` | `bun.lock` source-of-truth; `package-lock.json` yok, npm bagimliliklari farkli cozuyor |
| `mobile/` | `npm install`, `npm run`, `npx expo`, `npx jest` | `bun` (eskiden vardi, kaldirildi) | Expo prebuild scriptleri npm assumption ile yazilmis, `package-lock.json` source |
| `backend/` | `go mod`, `make sqlc`, `make migrate-up` | dogrudan `goose`, `sqlc generate` | Makefile env ve `sqlc.yaml`/`dbstring` cozumlemesi yapiyor; ciplak komut config bulamaz |

---

## 5. Docker / sudo Kurali

- Docker komutlari `sudo` gerektiriyor (kullanici `docker` grubunda degil).
- **Sandbox `sudo` calistirmaz** — komutu **kullaniciya kopyala-yapistir** olarak goster, kendin calistirma.

```bash
# Bunu sen calistirma — kullaniciya goster:
sudo docker compose -f backend/infrastructure/docker-compose.yml up -d
sudo docker exec mydreamcampus-postgres-auth psql -U postgres -d mydreamcampus_auth -c "SELECT email FROM users;"
sudo docker logs -f mydreamcampus-postgres-auth
```

---

## 6. Onaysiz Yapilabilecekler vs. Sorulacaklar

### SORMA, dogrudan uygula:
- HTTP status code secimi (REST standardi: 200/201/204/400/401/403/404/409/422/500)
- Sifre hash (Argon2id), JWT (HS256) — sabit
- Migration **yazma** (dosya olusturma)
- sqlc query yazma + `make sqlc` calistirma
- DTO/Repository/Service/Handler iskeleti (skills.md sablonlarini izle)
- Test isimlendirme: `TestXxx_Scenario_ExpectedResult`
- Commit atma (atomic, feature bittiginde)
- Hata mesaji standardi (`shared/errors.AppError`)

### SOR, dogrudan UYGULAMA:
- **Yeni kutuphane** ekleme (ornek: validator icin go-playground vs ozzo)
- **Yeni servis** olusturma (port, scope, event semasi)
- **Yeni event** semasi veya mevcut event payload degisikligi (geriye uyumsuzluk)
- **Migration CALISTIRMA** (`make migrate-up`) — yazma degil, calistirma sor
- **Sema breaking change** (kolon silme, NOT NULL ekleme, type degisikligi)
- **Frontend route silme** veya yeniden adlandirma
- Onemli refactor (3+ dosya etkileyen, davranis degisikligi)
- `go.mod` / `package.json` dependency guncelleme (patch haric)

---

## 7. Git Commit Formati

```
<type>(<scope>): <description>
```

| Type | Ne zaman |
|---|---|
| `feat` | Yeni ozellik |
| `fix` | Bug fix |
| `chore` | Build, infra, tooling |
| `refactor` | Davranis degismeden yeniden yazim |
| `docs` | Dokumantasyon |
| `test` | Sadece test ekleme |

**Scope:** `auth`, `staff`, `student`, `catalog`, `enrollment`, `attendance`, `grades`, `meal`, `payment`, `shared`, `frontend`, `mobile`, `infra`

**Ornekler:**
```
feat(auth): add login and register endpoints
fix(shared): resolve logger initialization bug
chore(infra): update traefik configuration
feat(frontend): add student dashboard page
feat(mobile): implement attendance screen
```

**Kural:** Her ozellik tamamlanınca **HEMEN** commit. Atomic — bir commit bir mantiksal degisiklik.

**YAPMA:**
- `feat: stuff` (scope yok, aciklama yok)
- `update files` (type yok)
- 10 dosya tek commit'te birden cok ozellik
- `--amend` ile push edilmis commit'i degistirme
- `--no-verify` (hook bypass)

---

## 8. Is Bittiginde Checklist (Gorev Kapatmadan Once)

```
Backend feature:
- [ ] Migration yazildi + `make migrate-up` test edildi (kullanici calistirir)
- [ ] sqlc query yazildi + `make sqlc` calisti
- [ ] Repository / Service / Handler / DTO yazildi
- [ ] Route `cmd/main.go`'ya baglandi
- [ ] Event publish ediliyorsa, consumer service'lerin DTO'lari guncellendi
- [ ] Service-level test yazildi (kritik path'ler — happy + 1 error)
- [ ] `go build ./...` hatasiz
- [ ] Atomic commit atildi

Frontend feature:
- [ ] API service fonksiyonu yazildi (`src/lib/services/`)
- [ ] TanStack Query hook'u var (loading/error state)
- [ ] Type tanimi `src/lib/types.ts`'de
- [ ] Sayfa `src/routes.tsx`'e baglandi
- [ ] `bun tsc --noEmit` hatasiz
- [ ] Tarayicida acilip golden path test edildi
- [ ] Atomic commit atildi

Mobile feature:
- [ ] Service fonksiyonu (`mobile/services/`)
- [ ] Hook (`mobile/hooks/`) — TanStack Query
- [ ] Ekran `mobile/app/` altinda (Expo Router)
- [ ] `npx tsc --noEmit` hatasiz
- [ ] iOS veya Android'de manuel test edildi (loading/error/empty)
- [ ] Atomic commit atildi
```

---

## 9. Failure Mode'lar

### Test basarisiz olursa
**YAPMA:** Test'i `t.Skip()` veya `.skip()` ile atla.
**YAP:** Hatayi oku, fix et veya kullaniciya rapor et. Commit atma.

### Migration basarisiz olursa
**YAPMA:** Tablo manuel `DROP` etme, migration tablosunu `DELETE` etme.
**YAP:** Kullaniciya hatayi goster, `migrate-down` oner.

### Type error (frontend/mobile)
**YAPMA:** `as any`, `@ts-ignore`, `// @ts-expect-error` kullanma.
**YAP:** Type tanimini duzelt. `as unknown as X` dokum gerekiyorsa kullaniciya sor.

### sqlc generate hata verirse
**YAPMA:** Generated dosyalari manuel duzenleme.
**YAP:** Query SQL'ini duzelt, tekrar `make sqlc`.

### Lint/format hata
**YAP:** Otomatik duzeltilebilenleri duzelt (`gofmt`, `bun tsc`). Logic degisimi gerekirse kullaniciya sor.

---

## 10. Tone & Output

- Cevaplar **kisa** olsun. Diff varsa diff'i konus, kodu tekrar yazma.
- Turkce aciklamalarda **emoji kullanma**.
- Log/yorum yaziminda **NE** degil **NEDEN** acikla. "fetches user" ❌ — "timing-safe: dummy verify against enumeration" ✅.
- Her yerde yorum **ekleme**. Sadece sart oldugunda (gizli kisit, edge case, workaround).
- Kullaniciya rapor verirken: once sonuc, sonra detay. Tersi degil.

---

## 11. Subagent Kullanimi

**Ne zaman kullan:**
- 3+ servisi tarayan arastirma
- Tum kod tabaninda pattern arama
- Karsilastirma analizi (servis A vs servis B'deki yaklasim)

**Ne zaman KULLANMA:**
- Tek dosya okuma
- Bilinen yoldaki dosyayi okuma
- 1-2 grep yeterli olan arama

Kullanici "agent kullan" derse `Agent` tool'u ile `Explore` veya `general-purpose` subagent'i cagir.

---

## 12. Mimari Kararlar (Sabit, Tartisilmaz)

Bu kararlar verilmis — yeniden sorma:

| Konu | Karar |
|---|---|
| Inter-service iletisim | **Tercih: RabbitMQ event-driven** (notify, side-effect, eventual consistency). **Sync HTTP kabul:** read-only lookup, anlik validasyon, fan-out orkestrasyon — `internal/...` route + `X-Internal-Secret` header zorunlu. **Client -> servis** HTTP (Traefik uzerinden). JWT'yi her servis kendi `JWTAuth` middleware'i ile dogrular — auth servisine HTTP RPC yok. |
| Database | PostgreSQL 18+, her servise ayri DB |
| ORM/Query | sqlc + pgx/v5 (raw SQL yok, GORM yok) |
| Migration | goose |
| HTTP framework (Go) | Gin v1.11 |
| Auth | JWT HS256 + Argon2id + Redis blacklist |
| Frontend routing | react-router v7 (Next.js YOK) |
| Mobile routing | Expo Router v6 (file-based) |
| Frontend HTTP | ky |
| Mobile HTTP | axios |
| State (web+mobile) | TanStack Query (server state), Context (UI state) |
| Logging | Zap (backend), console (frontend, debug icin) |
| Outbox pattern | Tum event publish'lerde zorunlu |
| API gateway | Traefik v3 (port 80) |

---

## 13. Servis Portlari (referans)

| Servis | HTTP | DB |
|---|---|---|
| auth | 8001 | 5432 |
| staff | 8002 | 5433 |
| student | 8003 | 5434 |
| course-catalog | 8004 | 5435 |
| enrollment | 8005 | 5436 |
| attendance | 8006 | 5437 |
| grades | 8007 | 5438 |
| meal | 8008 | 5439 |
| payment | 50051 (gRPC) | 5440 |

Frontend dev: `3000`. Traefik gateway: `80`. Tum `/api/*` Traefik uzerinden.

---

## 14. Dokunma — Generated / Korunan Dosyalar

Bu yollardaki dosyalari **manuel duzenleme**. Kaynak dosyayi guncelle ve generator'i tekrar calistir.

| Yol | Kaynak | Regenerate |
|---|---|---|
| `backend/services/*/internal/db/*.go` | `sql/queries/*.sql` | `make sqlc` |
| `backend/services/*/sql/migrations/*.sql` (uygulanmis) | — | Yeni migration ekle, eskisini degistirme |
| `frontend/src/lib/api-types.ts` | Backend OpenAPI | `bun run gen:api-types` |
| `frontend/src/components/ui/*` | shadcn CLI | `bunx --bun shadcn@latest add <c>` |
| `mobile/types/api-types.ts` | Backend OpenAPI | `npm run gen:api-types` |
| `*.lock`, `*.lockb`, `go.sum`, `bun.lock`, `package-lock.json` | Paket yoneticisi | Komutu calistir, manuel dokunma |

---

## 15. Detayli Rehberler

- Backend: [`backend/skills.md`](backend/skills.md)
- Frontend: [`frontend/skills.md`](frontend/skills.md)
- Mobile: [`mobile/skills.md`](mobile/skills.md)
- Moduler monolith migration plan: [`architecture/README.md`](architecture/README.md) — index. AI yeni oturumda once `architecture/00-ai-rules.md` okur.

## 16. Dis Referanslar

- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [sqlc Docs](https://docs.sqlc.dev/en/latest/)
- [pgx Docs](https://pkg.go.dev/github.com/jackc/pgx/v5)
- [goose Docs](https://pressly.github.io/goose/)
- [Traefik v3 Docs](https://doc.traefik.io/traefik/)
- [Expo Router Docs](https://docs.expo.dev/router/introduction/)
- [React Router v7 Docs](https://reactrouter.com/)
- [TanStack Query Docs](https://tanstack.com/query/latest)
