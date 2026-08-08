# Faz 6 — Docker Compose (16 Konteyner)

**Ön koşul:** Faz 4 ve Faz 5 tamamlandı
**Risk:** Orta — kaynak limitleri ve `depends_on` sırası homeserver'da kritik
**Bu fazın sonunda sistem uçtan uca çalışır**

---

## Amaç

`monolith` + `notification-postgres` servislerini compose'dan çıkarıp yerine
9 iş servisi koymak, kaynak limitlerini homeserver'a göre ayarlamak.

Konteyner haritası ve RAM tahminleri: `01-REFERANS-MIMARI.md` §4.
Env değişkenleri: `01-REFERANS-MIMARI.md` §6.

---

## Compose Dosya Bölünmesi (koru)

CLAUDE.md §13 kuralı **geçerliliğini koruyor**:

| Dosya | Publish ettiği host portu |
|---|---|
| `docker-compose.yml` (base) | **Sadece** Caddy `:80` |
| `docker-compose.standalone.yml` | Caddy `:443` + infra portları (`127.0.0.1`) |

**Servislerin hiçbiri host portu publish etmez** — sadece `expose`. Dışarı
açılan tek kapı Caddy. Yeni bir host portu gerekirse **standalone dosyasına**
ekle, base'e değil (Openship kendi edge'iyle `:443`'ü tutuyor).

---

## Adımlar

### 1. `notification-postgres` servisini sil

DB'si Faz 1'de tek Postgres'e katlandı. `notification-postgres-data` volume'unu
da `volumes:` bloğundan çıkar.

**Veri kaybı uyarısı:** Bu volume'da e-posta teslimat logları var. Önemliyse
önce yedekle:

```bash
sudo docker exec mydreamcampus-notification-postgres \
  pg_dump -U postgres -d notification > ~/notification-yedek.sql
```

### 2. `monolith` servisini sil, 9 servis ekle

Her servis için şablon:

```yaml
  auth-service:
    build:
      context: ../..
      dockerfile: new-backend/services/auth-service/Dockerfile
    image: mydreamcampus-auth:latest
    container_name: mydreamcampus-auth
    restart: unless-stopped
    mem_limit: 192m
    environment:
      ENVIRONMENT: production
      PORT: "8081"
      DB_URL: postgres://auth_svc:${SERVICE_DB_PASSWORD:?}@postgres:5432/auth?sslmode=disable
      RABBITMQ_URL: amqp://${RABBITMQ_USER:-rabbitmq}:${RABBITMQ_PASSWORD:?}@rabbitmq:5672/
      REDIS_ADDR: redis:6379
      REDIS_PASSWORD: ${REDIS_PASSWORD:?}
      JWT_SECRET: ${JWT_SECRET:?}
      INTERNAL_SERVICE_SECRET: ${INTERNAL_SERVICE_SECRET:?}
      CORS_ALLOWED_ORIGINS: ${PUBLIC_ORIGIN:?}
      ADMIN_EMAIL: ${ADMIN_EMAIL:-admin@university.edu.tr}
      ADMIN_INITIAL_PASSWORD: ${ADMIN_INITIAL_PASSWORD:?}
    expose:
      - "8081"
    depends_on:
      postgres:    { condition: service_healthy }
      redis:       { condition: service_healthy }
      rabbitmq:    { condition: service_healthy }
      migrate:     { condition: service_completed_successfully }
```

**Servise özel farklar** (`01-REFERANS-MIMARI.md` §6):

| Servis | Ek env |
|---|---|
| catalog | `STAFF_SERVICE_URL: http://staff-service:8082`, `MEAL_SERVICE_URL: http://meal-service:8088` |
| student | `STAFF_SERVICE_URL` |
| enrollment | `STUDENT_SERVICE_URL: http://student-service:8083`, `CATALOG_SERVICE_URL: http://catalog-service:8084` |
| attendance | `CATALOG_SERVICE_URL`, `QR_SECRET` |
| grades | `CATALOG_SERVICE_URL` |
| meal | `PAYMENT_SERVICE_URL: http://payment-service:8089`, `QR_SECRET` |
| payment | `DB_URL` **YOK** — stateless. `depends_on`'dan postgres ve migrate'i çıkar. |

**Sync bağımlılığı `depends_on`'a YAZMA.** catalog, staff'a HTTP atıyor diye
`depends_on: staff-service` eklersen başlangıç sırasını zincirlersin ve tek
servisin yavaş açılması tüm stack'i bekletir. Servisler karşı taraf henüz
ayakta değilken hata dönmeli, başlamayı reddetmemeli.

**Servis konteynerlerine `healthcheck:` YAZMA.** Image'lar
`gcr.io/distroless/static-debian12:nonroot` — içlerinde **shell, wget veya
curl yok**. `test: ["CMD-SHELL", "wget ..."]` yazarsan konteyner sonsuza kadar
`unhealthy` görünür. `depends_on` zaten sadece postgres / redis / rabbitmq /
migrate'e bakıyor ve onların hepsinde çalışan healthcheck var — servisler için
gerekmiyor.

İleride gerekirse doğru çözüm: binary'ye `-health` bayrağı ekleyip
`test: ["CMD", "/app/grades", "-health"]` yazmak. Şimdi yapma.

### 2b. Log rotasyonu (zorunlu)

16 konteynerin varsayılan json-file logu sınırsız büyür ve homeserver diskini
doldurur. **Her servise** ekle:

```yaml
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
```

Konteyner başına tavan 30 MB, 16 konteyner için ~480 MB. Bu bir
gözlemlenebilirlik özelliği değil, operasyonel hijyen — ileride Loki eklenince
de gerekli kalır.

### 3. Kaynak limitleri

Homeserver'da bir servisin kaçmasını engellemek için `mem_limit` zorunlu:

| Servis grubu | `mem_limit` | Gerekçe |
|---|---|---|
| postgres | `768m` | 9 DB, connection pool'lar |
| rabbitmq | `384m` | Kuyruk birikimi payı |
| redis | `128m` | Blacklist + QR buffer |
| auth, attendance, meal | `192m` | Redis'li, yük yoğun |
| diğer 6 servis | `128m` | |
| notification | `128m` | |
| caddy | `128m` | |
| mailhog | `64m` | |

Toplam tavan ≈ 2.5 GB; gerçek kullanım ≈ 760 MB. Limit tavan, rezervasyon
değil.

**Connection pool uyarısı:** 9 servis × varsayılan pgx pool (`max_conns`
CPU sayısına göre, tipik 4-8) = 36-72 bağlantı. Postgres varsayılanı
`max_connections=100`. Sıkışırsa Postgres komutuna
`-c max_connections=200 -c shared_buffers=256MB` ekle.

### 4. `seed` servisini düzelt

Şu an `API_URL: http://monolith:8080`. Seed birden fazla modülün API'sini
kullanıyor (staff, student, catalog, enrollment, grades, attendance, meal) —
tek servise yönlendirilemez.

```yaml
  seed:
    environment:
      API_URL: http://caddy:80        # gateway üzerinden, tüm prefixler erişilebilir
      DB_URL: postgres://auth_svc:${SERVICE_DB_PASSWORD:?}@postgres:5432/auth?sslmode=disable
    depends_on:
      caddy: { condition: service_started }
```

`DB_URL`'i neden kullandığını kontrol et (admin bootstrap doğrulaması olabilir)
ve hangi DB'ye bakması gerektiğine ona göre karar ver.

### 5. `caddy` servisi

`depends_on: - monolith` → 9 servise çevir (veya tamamen kaldır; Caddy upstream
yokken de ayağa kalkar ve 502 döner — homeserver'da bu daha dayanıklı davranış).

Öneri: `depends_on` listesini kaldır. Caddy'nin ayakta olması SPA'nın
servis edilmesi için yeterli; backend gecikirse kullanıcı beyaz ekran yerine
API hatası görür.

### 6. Kök `Makefile`

```make
deploy-logs:
	$(SUDO) docker compose $(COMPOSE) logs -f caddy auth-service catalog-service

# Tek servisin logu
logs-%:
	$(SUDO) docker compose $(COMPOSE) logs -f $*

# Tek servisi yeniden başlat
restart-%:
	$(SUDO) docker compose $(COMPOSE) restart $*
```

`backend:` ve `notification:` hedefleri (host'ta `go run`) artık tek servis
çalıştırmak için anlamlı değil — sil veya `run-%` deseniyle değiştir.

### 7. Gözlemlenebilirlik overlay'i için yer aç (dosya oluşturma, sadece not)

Prometheus/Grafana/Loki bu migrasyonun kapsamında **değil**. Ama compose zaten
`base + standalone` overlay desenini kullanıyor — üçüncü bir overlay doğal ev:

```make
# İleride: make deploy OBSERVABILITY=1
COMPOSE := -f $(COMPOSE_FILE) -f $(INFRA)/docker-compose.standalone.yml \
           $(if $(OBSERVABILITY),-f $(INFRA)/docker-compose.observability.yml)
```

`docker-compose.observability.yml` dosyasını **şimdi oluşturma**. Sadece
`COMPOSE` değişkenindeki bu deseni yorumla belgele ki sonradan eklerken
Makefile'ı yeniden düşünmek gerekmesin.

Gözlemlenebilirlik altyapısı **sıfırdan kurulacak** — repoda hazır
Grafana/Loki/Promtail config'i yok (`SYSTEM-DESIGN.md` var diyor, yanlış; o
doküman kaynak değil, bkz. `00-BASLANGIC.md`).

### 8. `definitions.json` mount'u

Faz 4'te oluşturulan RabbitMQ topoloji dosyasını bağla:

```yaml
  rabbitmq:
    volumes:
      - rabbitmq-data:/var/lib/rabbitmq
      - ./rabbitmq/rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf:ro
      - ./rabbitmq/definitions.json:/etc/rabbitmq/definitions.json:ro
```

`rabbitmq.conf`'ta `management.load_definitions` satırının yorumu Faz 4'te
kaldırılmıştı — dosya mount edilmezse RabbitMQ **başlamaz**. İkisi birlikte
gider.

Doğrulama: `http://localhost:15672` → Queues sekmesinde, hiçbir servis
başlamamışken bile tüm kuyruklar görünmeli.

---

## Başlatma ve Doğrulama

Kullanıcıya göster, sen çalıştırma (CLAUDE.md §5):

```bash
cd "/home/nautilus/Desktop/Playground/mydreamcampus (mikroservis)"

# .env'e SERVICE_DB_PASSWORD eklendiğinden emin ol
grep SERVICE_DB_PASSWORD new-backend/infrastructure/.env

make deploy

# 16 konteyner ayakta mı?
make deploy-ps

# Kaynak kullanımı
sudo docker stats --no-stream
```

---

## Bitiş Kriteri

```bash
# 1. 16 konteyner, hepsi Up (seed ve migrate Exited(0))
sudo docker compose -f new-backend/infrastructure/docker-compose.yml \
     -f new-backend/infrastructure/docker-compose.standalone.yml ps

# 2. Her servis kendi /health'ini döndürüyor.
#    Servis image'ları distroless (shell yok) — kontrol caddy konteynerinden
#    yapılır, o caddy:2-alpine tabanlı ve wget içeriyor.
for s in auth:8081 staff:8082 student:8083 catalog:8084 enrollment:8085 \
         attendance:8086 grades:8087 meal:8088 payment:8089; do
  name="${s%%:*}"; port="${s##*:}"
  printf "%-12s " "$name"
  sudo docker exec mydreamcampus-caddy \
    wget -qO- --timeout=3 "http://${name}-service:${port}/health" || echo "ERISILEMEDI"
done

# 3. Gateway routing çalışıyor
curl -s -o /dev/null -w "%{http_code}\n" localhost/api/auth/login    # 400/405 (401 değil ama 502 de değil)
curl -s -o /dev/null -w "%{http_code}\n" localhost/api/catalog       # 401 (auth gerekiyor)
curl -s -o /dev/null -w "%{http_code}\n" localhost/api/yanlis-prefix # 404
curl -s localhost/health

# 4. RAM tahmini tutuyor mu
sudo docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}"

# 5. İnternal route'lar DIŞARIDAN erişilemiyor — 404 dönmeli
curl -s -o /dev/null -w "%{http_code}\n" localhost/internal/staff/00000000-0000-0000-0000-000000000000
```

**5. madde bir güvenlik kontrolü** — 404 dışında bir şey dönerse Caddy
`/internal`'i proxy'liyor demektir, düzeltmeden fazı kapatma.

Toplam RAM 1.2 GB'ı aşıyorsa `mem_limit`'leri ve pgx pool ayarlarını gözden
geçir.

---

## Commit

```
chore(infra): replace the monolith container with nine service containers
```

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-6-docker-compose-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 6 satırını `[x]` yap
3. "Sıradaki faz" satırını **7** yap
