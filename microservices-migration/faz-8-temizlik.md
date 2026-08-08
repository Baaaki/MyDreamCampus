# Faz 8 — Temizlik ve Dokümantasyon

**Ön koşul:** Faz 7 tamamlandı, sistem çalışıyor
**Risk:** Düşük — ama CI ve deploy pipeline'ı burada düzeliyor, atlanırsa
otomatik deploy kırılır

---

## Amaç

Monolith kalıntılarını silmek ve dokümanları gerçekle uyumlu hale getirmek.
Doküman güncellemesi burada **isteğe bağlı değil**: `CLAUDE.md` bir sonraki
oturumda AI'a talimat olarak yükleniyor, eski mimariyi anlatırsa yanlış kod
üretilir.

---

## A — Kod Temizliği

### A1. `monolith/` dizinini sil

```bash
cd new-backend
git rm -r monolith
```

Silmeden önce içinde kalan bir şey olmadığını doğrula:

```bash
find monolith -name "*.go" -not -path "*/bin/*"
```

Sadece boş `cmd/`, `test/testdb`, `test/testfixtures` kalmış olmalı.
`test/testdb` ve `test/testfixtures` **hâlâ kullanılıyorsa**
`shared/testing/` altına taşı, silme.

`go.work`'ten `./monolith` satırını çıkar (Faz 4'te çıkarılmış olmalı).

### A2. Eski Postgres init dosyalarını sil

```bash
git rm new-backend/infrastructure/postgres/init-dbs.sql
git rm new-backend/infrastructure/postgres/init-notification-db.sql
```

Compose'daki mount satırlarını da kaldır. `init-databases.sh` tek kaynak kalır.

### A3. Eski `mydreamcampus` veritabanını düşür

Artık kimse kullanmıyor. **Önce yedek al:**

```bash
sudo docker exec mydreamcampus-postgres \
  pg_dump -U postgres -d mydreamcampus > ~/mydreamcampus-son-yedek-$(date +%F).sql

sudo docker exec mydreamcampus-postgres \
  psql -U postgres -c "DROP DATABASE mydreamcampus;"
```

`migrate/entrypoint.sh`'daki geriye dönük `DB_URL` / `NOTIF_DB_URL` yolunu
(Faz 1'de eklenmişti) da sil. Compose'daki o env satırları da gider.

### A4. Ölü kod taraması

```bash
cd new-backend
# Monolith'e dair kalan referanslar
grep -rn "monolith" --include="*.go" --include="*.yml" --include="*.yaml" \
  --include="Makefile" --include="Dockerfile" . | grep -v "^./services/.*/vendor"

# SPA static serving artık hiçbir serviste kullanılmıyor
grep -rn "FRONTEND_STATIC" --include="*.go" .
```

`FRONTEND_STATIC_ENABLED` / `FRONTEND_STATIC_DIR` config alanlarını ve
`shared/httpserver`'daki SPA fallback kodunu sil — Caddy servis ediyor.

---

## B — CI / Deploy Pipeline

### B1. `.github/workflows/ci.yml`

Monolith referansları var (satır ~61, 73, 94, 115, 182-208). Matris yapısına
çevir:

```yaml
strategy:
  matrix:
    service: [auth, staff, student, catalog, enrollment, attendance, grades, meal, payment, notification]
steps:
  - run: cd new-backend/services/${{ matrix.service }}-service && go vet ./... && go test ./...
```

Integration job'undaki "start monolith → wait for health" bloğu, tek servis
yerine `docker compose up` ile tüm stack'i ayağa kaldıracak şekilde
değiştirilmeli (veya integration testi compose üzerinden çalıştırılmalı).

### B2. `.github/workflows/cd.yml`, `deploy.yml`, `security.yml`

`monolith` build/scan adımlarını servis listesine çevir. `gosec` ve
`govulncheck` artık `./services/...` üzerinde koşmalı.

### B3. `openship.json`

Yeni secret eklendi:

```json
"SERVICE_DB_PASSWORD": { "value": "", "secret": true }
```

Bu Faz 1'de eklenmiş olmalı — eklenmediyse **şimdi ekle**, yoksa Openship
deploy'u `SERVICE_DB_PASSWORD is required` ile patlar.

### B4. `scripts/auto-deploy.sh`

Health check yolunu ve rebuild edilen servis listesini kontrol et. `/health`
Caddy üzerinden auth-service'e gidiyor (Faz 5) — yol değişmediyse dokunma.

---

## C — Dokümantasyon

### C1. `CLAUDE.md` — en kritik dosya

| Bölüm | Yapılacak |
|---|---|
| Başlık paragrafı | "Go moduler monolith (`new-backend/`)" → "Go mikroservisler (`new-backend/services/`)" |
| §2 Zorunlu okuma | `new-backend/monolith/**` satırını `new-backend/services/**` yap |
| §4 Paket yöneticisi | `new-backend/monolith/` → servis dizinleri; `make sqlc-<module>` → servis kökünde `make sqlc` |
| §12 Mimari kararlar | **Mimari** satırı: "Modüler monolith" → "Mikroservis (10 servis)". **Modüller arası iletişim** satırı: in-process client → **internal REST + X-Internal-Secret**. **Database** satırı: tek DB + schema → **servis başına ayrı DB, tek Postgres konteyneri**. |
| §13 Portlar | Tabloyu `01-REFERANS-MIMARI.md` §1 ve §4 ile değiştir |
| §14 Generated dosyalar | Yolları `services/<x>-service/internal/db/` yap |
| §7 Commit scope | `catalog` scope'u zaten var, değişiklik gerekmez |

### C2. `SYSTEM-DESIGN.md`

En büyük iş burada (353 satır). Değişecekler:

- §1 Genel bakış + ASCII diyagram → `01-REFERANS-MIMARI.md` §4 ve bu
  klasördeki hedef şema
- §2 HTTP katmanı → "moduller arası sync iletişim" paragrafı internal REST'i
  anlatmalı
- §4.1 Modül şablonu → servis şablonu (dizin yapısı `01-REFERANS-MIMARI.md` §5)
- §4.2 Her modülün "Sync bağımlılık" satırları → HTTP client'lara güncelle
- §5 Veri mimarisi → schema-per-module değil, **database-per-service**
- §6 Event mimarisi → yeni kuyruklar (period, audit) eklendi
- §10 Altyapı → 16 konteyner
- §11 Bilinen eksikler → "HTTP loopback kalıntısı" maddesini **sil** (çözüldü),
  "meal/proto gRPC kalıntısı" notunu güncelle (gRPC ileride pilot olarak
  değerlendirilecek)
- **§10 ve §11'deki yanlış iddiayı düzelt:** ikisi de *"Grafana/Loki/Promtail
  config'leri hazır ama compose'a ekli değil"* diyor. Böyle bir dizin yok — ne
  `main`'de ne `v0-microservices` tag'inde. Doğru ifade: *"Gözlemlenebilirlik
  (Prometheus/Loki/Grafana) henüz kurulmadı. `shared/httpserver` tüm
  servislerin ortak giriş noktası olduğu için `/metrics` tek yerden eklenebilir;
  compose için üçüncü bir overlay dosyası öngörülüyor."*

### C3. `DEPLOY.md`

32 KB, Openship deploy adımlarını anlatıyor. Değişecekler:
- Konteyner listesi ve beklenen `docker compose ps` çıktısı
- `.env` bölümüne `SERVICE_DB_PASSWORD`
- Sorun giderme bölümündeki `docker logs mydreamcampus-monolith` komutları →
  servis adları
- RAM/disk gereksinimleri (~760 MB / ~650 MB image)

### C4. `README.md`

Mimari özeti ve "nasıl çalıştırılır" bölümü.

### C5. `new-backend/skills.md`

Backend geliştirme rehberi — modül şablonu yerine servis şablonu, `make`
komutları servis köküne taşındı.

### C6. Bu migrasyon klasörü

`microservices-migration/` klasörünü **silme**. Tarihsel referans olarak dursun
— `00-BASLANGIC.md`'nin başına bir satır ekle:

```
> MIGRASYON TAMAMLANDI (<tarih>). Bu klasör tarihsel referanstır.
> Güncel mimari için SYSTEM-DESIGN.md'ye bak.
```

---

## D — Son Kontrol

```bash
# 1. Monolith kelimesi kod ve config'te kalmadı
grep -rn "monolith" --include="*.go" --include="*.yml" --include="Dockerfile" \
  --include="Makefile" new-backend .github

# 2. Dokümanlar tutarlı
grep -rn "modüler monolith\|moduler monolith\|schema-per-module\|schema per module" \
  CLAUDE.md SYSTEM-DESIGN.md README.md DEPLOY.md

# 3. Tam yeniden kurulum çalışıyor (temiz volume)
sudo docker compose -f new-backend/infrastructure/docker-compose.yml \
  -f new-backend/infrastructure/docker-compose.standalone.yml down -v
make deploy
# → 16 konteyner, seed geçiyor, golden path çalışıyor

# 4. Testler
make test
```

**3. madde en önemlisi:** sıfırdan kurulum çalışmıyorsa init script'i, migrate
sırası veya seed'de eksik var. Bunu doğrulamadan migrasyonu kapatma.

---

## Commit

```
chore(infra): remove the monolith and switch CI to the per-service matrix
docs: rewrite architecture documentation for the microservices layout
```

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-8-temizlik-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 8 satırını `[x]` yap
3. "Sıradaki faz" satırını **YOK — migrasyon tamamlandı** yap
4. Kullanıcıya özet rapor ver: neyin değiştiği, RAM/konteyner sayısı, kalan
   bilinen eksikler (`01-REFERANS-MIMARI.md` §8)
