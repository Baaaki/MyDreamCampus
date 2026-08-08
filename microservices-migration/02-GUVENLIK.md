# Güvenlik — OWASP Top 10 (2021) Karşılığı

> Faz dosyası değil, **kontrol listesi**. Faz 1, 3, 4, 5, 6'da açmadan geçme.
> Sonunda faz bazlı kapı tablosu var.

Bu dosya "OWASP'a uygunluk sertifikası" değil. Mevcut kod tabanı zaten
Argon2id, JWT HS256, CSRF, rate limit, security headers, `gosec` +
`govulncheck` ile geliyor. Burada listelenen şey farklı: **monolithi 10 servise
bölmenin yarattığı yeni risk yüzeyi.** Monolith'te var olmayan, bölünce ortaya
çıkan şeyler.

---

## A01 — Broken Access Control

### Yeni risk: internal endpoint'ler ağdan erişilebilir

Monolith'te modüller arası çağrı bir Go fonksiyon çağrısıydı. Artık HTTP.

| Kural | Nerede | Doğrulama |
|---|---|---|
| `/internal/*` route'ları `/api` **dışında**, kök altında | Faz 4, adım 5 | `curl localhost/internal/... → 404` |
| Caddy `/internal`'i **proxy'lemez** | Faz 5 | Caddyfile'da `/internal` geçmemeli |
| Her internal route `InternalAuth` middleware'i taşır | Faz 3, adım 2 | Derinlemesine savunma — Caddy kuralı unutulursa ikinci hat |
| `INTERNAL_SERVICE_SECRET` boşsa servis **başlamaz** | Faz 4, `main.go` | Boş secret = `InternalAuth` bypass demek |

### Yeni risk: RBAC artık 9 yerde

Monolith'te tek middleware zinciri vardı. Şimdi her servis kendi zincirini
kuruyor — biri `RequireRole`'ü unutursa o servis korumasız kalır.

```bash
# Her serviste JWTAuth ve rol kontrolü var mı
for d in new-backend/services/*-service; do
  printf "%-28s JWTAuth:%s RequireRole:%s\n" "$(basename $d)" \
    "$(grep -rc "JWTAuth" $d/internal --include=module.go)" \
    "$(grep -rc "RequireRole\|RequireAdmin\|RequireTeacher\|RequireStudent" $d/internal --include=module.go)"
done
```

Sıfır çıkan servis varsa (payment hariç — public mock) incele.

---

## A02 — Cryptographic Failures

Bölünme secret **sayısını** değil, secret'ın **bulunduğu yer sayısını** artırıyor:

| Secret | Monolith | Mikroservis |
|---|---|---|
| `JWT_SECRET` | 1 container | **9 container** |
| `INTERNAL_SERVICE_SECRET` | 1 | **9** |
| `QR_SECRET` | 1 | 2 (attendance, meal) |
| DB parolası | 1 | 9 kullanıcı (ortak parola, ayrı CONNECT yetkisi) |

Kurallar:
- Secret **image'a gömülmez** — sadece compose `environment` üzerinden env.
- Secret **git'e girmez** — `.env` gitignore'da, `.env.example` sadece
  `CHANGE_ME` içerir.
- **`definitions.json` tuzağı:** RabbitMQ'nun `/api/definitions` çıktısı
  `users` bölümünde **parola hash'lerini içerir**. Faz 4'te bu dosya
  oluşturulurken `users`, `permissions`, `policies` blokları **silinir** —
  sadece `exchanges`, `queues`, `bindings` kalır. Hash'li de olsa parola
  git'e girmemeli.

**Servisler arası TLS yok** — trafik compose bridge network'ünde, host dışına
çıkmıyor. Tek makinede kabul edilebilir. Servisler ileride farklı makinelere
dağılırsa bu varsayım geçersiz olur; o zaman mTLS gerekir. Kararı belgele,
sessizce varsayma.

---

## A03 — Injection

sqlc + pgx her yerde parametreli sorgu üretiyor — bu taraf temiz.

**Migrasyonun kendisi bir injection yüzeyi yaratıyor:** Faz 2'de
`SimplePeriodRepository` schema adını SQL'e `fmt.Sprintf` ile gömüyor. Schema
adı config'ten gelen sabit, kullanıcı girdisi değil — ama savunma yine de
whitelist'le yapılmalı (Faz 2, adım 4'te `allowedPeriodSchemas` map'i).

**Kural:** Migrasyon boyunca `fmt.Sprintf` ile kurulan **yeni** SQL yazma.
Tek istisna yukarıdaki, o da whitelist'li.

---

## A04 — Insecure Design

**Soğuk projeksiyon fail-open (Faz 2).** Tüketicinin `academic_periods`
tablosu boşken `platform/rules` dönem kontrolünü atlıyor — yani dönem kilidi
sessizce devre dışı. Bu mevcut davranış, migrasyonun getirdiği bir açık değil,
ama bölünme sonrası **daha kolay tetiklenir** (yeni servis, boş tablo).

Faz 7'de doğrula: dönem penceresi dışında ders seçimi denemesi **reddedilmeli**.
Kabul ediliyorsa projeksiyon boştur → Faz 2'nin republish adımını çalıştır.

---

## A05 — Security Misconfiguration

| Kural | Faz | Doğrulama |
|---|---|---|
| Servisler host portu **publish etmez**, sadece `expose` | 6 | `docker compose ps` → sadece caddy'de host portu |
| Infra portları `127.0.0.1`'e bağlı (standalone overlay) | 6 | RabbitMQ UI dışarıdan erişilemez |
| `REVOKE CONNECT ... FROM PUBLIC` her DB'de | 1 | Faz 1 bitiş kriteri 4. madde |
| Image'lar distroless + `USER nonroot` | 4 | Dockerfile şablonu bunu taşıyor, kaldırma |
| Base image sürümü pinli (`golang:1.26`, `postgres:18`) | 4, 6 | `:latest` kullanma |

---

## A06 — Vulnerable and Outdated Components

`go.mod` sayısı 2'den **11'e** çıkıyor. Tarama kapsamı büyümezse yeni
servisler denetimsiz kalır.

- `.github/workflows/security.yml` → `gosec` ve `govulncheck` artık
  `./services/...` ve `./shared/...` üzerinde koşmalı (Faz 8, B2).
- `.github/dependabot.yml` → her servis dizini için `gomod` girdisi.
  **11 girdi**, tek girdi değil.
- Bağımlılık sürümleri servisler arası **sapmamalı** — `go.work` lokal
  geliştirmede aynı sürümü zorlar ama Docker build `GOWORK=off` ile çalışıyor.
  Faz 8'de tüm `go.mod`'ların ortak bağımlılık sürümlerini karşılaştır.

---

## A07 — Identification and Authentication Failures

### Bölünmenin yarattığı GERÇEK regresyon: rate limit çarpanı

Rate limit anahtarı servis adıyla kuruluyor
([ratelimit.go:74](new-backend/monolith/internal/platform/middleware/ratelimit.go#L74)):

```go
key = fmt.Sprintf("ratelimit:%s:ip:%s:global", rl.config.ServiceName, c.ClientIP())
```

Monolith'te `ServiceName = "monolith"` → tek kova. Her servis kendi adını
verirse **9 ayrı kova** oluşur ve saldırgan isteği 9 servise yayarak efektif
IP limitini **9 katına** çıkarır. Bu, migrasyonun sessizce getirdiği bir
zayıflama.

**Çözüm (Faz 4, `main.go` wiring):**

```go
// Global IP/user limitleri TÜM servislerde aynı Redis kovasını paylaşmalı.
// Servis adı verilirse her servis kendi kovasını açar ve saldırgan isteği
// servislere yayarak limiti servis sayısı kadar katlar.
ServiceName: "public"   // sabit — servis adı DEĞİL
```

Endpoint bazlı limitler (`login`, `refresh`, `password`) sadece
auth-service'te yaşıyor, onlar doğal olarak tek kova — dokunma.

Faz 7'de doğrula: iki farklı servise dağıtılmış istekler **ortak** limite
takılmalı.

### Token blacklist her serviste kontrol edilmeli

Logout, JTI'yi Redis blacklist'ine yazıyor. Bir servis `JWTAuth`
middleware'ini atlarsa o serviste logout işlemez. Faz 7, E bölümü 4. madde
bunu test ediyor — atlama.

---

## A08 — Software and Data Integrity Failures

- `definitions.json` git'e girecek → içinde credential **olmamalı** (A02'ye bak).
- Docker build context repo kökü; `.dockerignore` `.env` dosyalarını dışlıyor
  mu, Faz 4'te kontrol et.
- `seed` container'ı admin API üzerinden yazıyor (DB'ye doğrudan değil) —
  bu tasarımı koru, API kontratları da doğrulanmış oluyor.

---

## A09 — Security Logging and Monitoring Failures

Bölünme burayı doğrudan zayıflatıyor: bir istek 9 servise dağıldığında olay
zinciri kaybolur.

- **İzlenebilirlik zorunlu** — bkz. `03-IZLENEBILIRLIK.md`. Bu bir "nice to
  have" değil, A09'un karşılığı.
- `audit.InitSecurity(cfg.Server.Environment)` her servisin `main.go`'sunda
  korunmalı (Faz 4, adım 4).
- Audit log artık **event üzerinden** catalog'a gidiyor (Faz 3, adım 4).
  Outbox garantisi kayıp riskini kapatıyor, ama Faz 7'de `course_catalog.audit_log`
  tablosunun dolduğunu doğrula.

---

## A10 — Server-Side Request Forgery

Internal client'ların base URL'i **env'den** geliyor
(`STAFF_SERVICE_URL` vb.), istek verisinden değil.

**Kural:** İstek gövdesinden veya query'den gelen bir değeri asla internal
client'ın URL'ine koyma. Path parametresi (`/internal/staff/:id`) URL
encode'lanmalı.

---

## Faz Bazlı Kapılar

Her fazın bitiş kriterine ek olarak buradaki maddeler:

| Faz | Kapı |
|---|---|
| 1 | `REVOKE CONNECT` izolasyon testi geçiyor (A05) |
| 2 | `allowedPeriodSchemas` whitelist'i var, ham schema interpolasyonu yok (A03) |
| 3 | Her internal route `InternalAuth` taşıyor (A01); audit event akıyor (A09) |
| 4 | `ServiceName: "public"` sabit — rate limit kovası ortak (A07); `definitions.json` credential içermiyor (A02, A08); `audit.InitSecurity` her serviste (A09) |
| 5 | Caddyfile'da `/internal` yok (A01) |
| 6 | Servisler host portu publish etmiyor (A05) |
| 7 | Blacklist testi, DB izolasyon testi, internal erişilemezlik testi geçiyor |
| 8 | `gosec`/`govulncheck`/dependabot 11 modülü kapsıyor (A06) |
