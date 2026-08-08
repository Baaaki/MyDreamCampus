# Faz 0 — `platform/` → `shared/platform/`

**Ön koşul:** Yok (ilk faz)
**Risk:** Düşük — tamamen mekanik taşıma, davranış değişmiyor
**Monolith durumu:** Faz sonunda **hâlâ çalışır** olmalı

---

## Amaç

Go'da `internal/` altındaki paketler o modülün dışından import **edilemez**.
`monolith/internal/platform/` ayrı Go modülleri (servisler) tarafından
paylaşılamaz. Bu yüzden `shared/platform/` altına taşınıyor.

**Doğrulanmış kolaylık:** `internal/platform/` hiçbir monolith modülüne veya
`monolith/config`'e import yapmıyor. Yani taşıma çembersel bağımlılık
yaratmaz — sadece import path'i değişir.

---

## Kapsam

**Taşınan:** `new-backend/monolith/internal/platform/` (14 paket, 61 `.go` dosyası)

```
audit/ (4)  clock/ (2)  database/ (1)  dto/ (2)  errors/ (4)  handler/ (4)
logger/ (3)  middleware/ (19)  rabbitmq/ (4)  redis/ (2)  repository/ (2)
rules/ (6)  semester/ (3)  utils/ (9)
```

**Taşınmayan:**
- `monolith/config/` — servis başına farklılaşacak, Faz 4'te bölünür
- `monolith/internal/eventbus/` — outbox worker + exchange declare; Faz 4'te
  `shared/`'a taşınacak (şimdi değil, modül `OutboxStore` tiplerine bağlı)
- `monolith/internal/http/server.go` — Faz 4'te `shared/httpserver/` olur
- `monolith/internal/modules/` — Faz 4'te servislere bölünür

---

## Adımlar

### 1. Dizini taşı

```bash
cd "new-backend"
git mv monolith/internal/platform shared/platform
```

### 2. Import path'lerini güncelle

Eski: `github.com/baaaki/mydreamcampus/monolith/internal/platform/...`
Yeni: `github.com/baaaki/mydreamcampus/shared/platform/...`

```bash
cd "new-backend"
grep -rl "monolith/internal/platform" --include="*.go" monolith shared \
  | xargs sed -i 's|github.com/baaaki/mydreamcampus/monolith/internal/platform|github.com/baaaki/mydreamcampus/shared/platform|g'
```

`shared/platform` içindeki dosyalar da birbirini eski path ile import
ediyordu — yukarıdaki komut `shared`'ı da kapsadığı için ikisi birden düzelir.

Kontrol (boş dönmeli):

```bash
grep -rn "monolith/internal/platform" --include="*.go" new-backend
```

### 3. Bağımlılıkları çöz

`shared/go.mod` şu an sadece `testify` içeriyor. Taşınan kod şunları
kullanıyor: gin, zap, pgx/v5, go-redis/v9, amqp091-go, golang-jwt/v5,
golang.org/x/crypto, google/uuid.

```bash
cd "new-backend/shared" && go mod tidy
cd "../monolith"        && go mod tidy
```

`monolith/go.mod`'daki `replace .../shared => ../shared` satırı **duruyor**,
dokunma.

### 4. Derleme kontrolü

```bash
cd "new-backend/shared"   && go build ./... && go vet ./...
cd "../monolith"          && go build ./... && go vet ./...
```

### 5. Testleri çalıştır

```bash
cd "new-backend/shared"   && go test ./...
cd "../monolith"          && go test ./...
```

Taşınan paketlerin testleri de birlikte geldi (`middleware`, `rules`,
`utils`, `errors`, `handler`, `audit`, `semester`, `clock`, `dto` test
dosyaları var). Hepsi geçmeli.

### 6. Dockerfile kontrolü

`monolith/Dockerfile` zaten `COPY new-backend/shared/ ./shared/` yapıyor ve
`GOWORK=off` ile derliyor — **değişiklik gerekmez**. Aynısı
`services/notification/Dockerfile` için de geçerli.

Doğrulama (kullanıcıya göster, sen çalıştırma — CLAUDE.md §5):

```bash
sudo docker build -f new-backend/monolith/Dockerfile -t mydreamcampus-monolith:test .
```

---

## Dikkat Edilecekler

- **`shared/platform/repository/`** içindeki `SimplePeriodRepository` ve
  `PeriodRepository`, `course_catalog.academic_periods` tablosuna ham SQL
  atıyor. Bu faz onu **olduğu gibi taşır** — düzeltmesi Faz 2'de. Şimdi
  dokunma.
- **`shared/platform/middleware/internal.go`** — `X-Internal-Secret`
  doğrulaması. Faz 3'te servisler arası REST için kullanılacak. Silme.
- `sed` komutu `_test.go` dosyalarını da kapsıyor — doğru davranış.
- Taşıma sonrası IDE'de kırmızı kalırsa `go clean -cache` deneyin.

---

## Bitiş Kriteri

```bash
# 1. Eski path hiçbir yerde kalmamalı (boş çıktı)
grep -rn "monolith/internal/platform" --include="*.go" new-backend

# 2. İki modül de derlenmeli
cd new-backend/shared && go build ./... && go vet ./...
cd ../monolith && go build ./... && go vet ./...

# 3. Testler geçmeli
cd new-backend/shared && go test ./...
cd ../monolith && go test ./...

# 4. Dizin yapısı
ls new-backend/shared/platform     # 14 dizin görünmeli
ls new-backend/monolith/internal   # platform/ ARTIK OLMAMALI
```

Dördü de yeşilse faz bitti.

---

## Commit

```
refactor(shared): move platform packages from monolith internal to shared module
```

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-0-shared-platform-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 0 satırını `[x]` yap
3. "Sıradaki faz" satırını **1** yap
