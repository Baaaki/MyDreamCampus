# Faz 1 — Tek Postgres, 9 Ayrı Veritabanı

**Ön koşul:** Faz 0 tamamlandı
**Risk:** Orta — **veri kaybı riski var**, "Volume Uyarısı" bölümünü atlama
**Monolith durumu:** Faz sonunda **hâlâ çalışır** (eski `mydreamcampus` DB'sinde kalır)

---

## Amaç

Tek Postgres konteynerinde servis başına ayrı veritabanı ve ayrı DB kullanıcısı
oluşturmak. Bu faz **altyapıyı hazırlar**; monolith hâlâ eski `mydreamcampus`
DB'sini kullanmaya devam eder. Yeni DB'lere geçiş Faz 4/6'da olur.

**Kritik tasarım kararı:** Schema adları değişmiyor. `auth` veritabanının
içinde `auth` schema'sı olur. Böylece tüm migration `.sql` dosyaları ve sqlc
generated kodu **hiç değişmeden** çalışır. Bu, migrasyonun en büyük riskini
sıfırlıyor — bu kuralı bozma.

---

## Oluşturulacak Veritabanları

| DB adı | İçindeki schema | Kullanıcı | Migration kaynağı |
|---|---|---|---|
| `auth` | `auth` | `auth_svc` | `monolith/internal/modules/auth/sql/migrations` |
| `staff` | `staff` | `staff_svc` | `.../staff/sql/migrations` |
| `student` | `student` | `student_svc` | `.../student/sql/migrations` |
| `catalog` | `course_catalog` | `catalog_svc` | `.../course_catalog/sql/migrations` |
| `enrollment` | `enrollment` | `enrollment_svc` | `.../enrollment/sql/migrations` |
| `attendance` | `attendance` | `attendance_svc` | `.../attendance/sql/migrations` |
| `grades` | `grades` | `grades_svc` | `.../grades/sql/migrations` |
| `meal` | `meal` | `meal_svc` | `.../meal/sql/migrations` |
| `notification` | `public` | `notification_svc` | `services/notification/sql/migrations` |

**payment DB'si yok** — stateless mock servis, migration'ı da yok.

Toplam 9 veritabanı.

---

## Adımlar

### 1. Init script'ini yaz

Yeni dosya: `new-backend/infrastructure/postgres/init-databases.sh`

Neden `.sh` ve `.sql` değil: Postgres entrypoint `.sh` dosyalarına environment
değişkenlerini geçirir, `.sql` dosyalarına geçirmez. Parolaları env'den almamız
gerekiyor.

Yapması gerekenler (her servis için):

```sh
#!/bin/bash
set -e

# Servis kullanıcılarının parolası. Tek parola kullanılıyor çünkü izolasyon
# parolayla değil, DB seviyesindeki CONNECT yetkisiyle sağlanıyor:
# auth_svc, catalog DB'sine parolayı bilse bile bağlanamaz.
SVC_PASSWORD="${SERVICE_DB_PASSWORD:?SERVICE_DB_PASSWORD is required}"

create_service_db() {
  db="$1"; schema="$2"; user="${1}_svc"

  psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres <<-EOSQL
    CREATE DATABASE "$db";
    CREATE USER "$user" WITH PASSWORD '$SVC_PASSWORD';
    REVOKE CONNECT ON DATABASE "$db" FROM PUBLIC;
    GRANT CONNECT, TEMPORARY ON DATABASE "$db" TO "$user";
EOSQL

  # Extension'lar superuser gerektirir, o yüzden burada (postgres olarak) açılır.
  # Schema AUTHORIZATION ile servis kullanıcısına verilir — goose migration'ları
  # servis kullanıcısıyla koştuğu için tabloların sahibi de o olur, ekstra
  # GRANT gerekmez.
  psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$db" <<-EOSQL
    CREATE EXTENSION IF NOT EXISTS "pgcrypto";
    CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
    CREATE SCHEMA IF NOT EXISTS "$schema" AUTHORIZATION "$user";
    GRANT CREATE ON DATABASE "$db" TO "$user";
EOSQL
}

create_service_db auth        auth
create_service_db staff       staff
create_service_db student     student
create_service_db catalog     course_catalog
create_service_db enrollment  enrollment
create_service_db attendance  attendance
create_service_db grades      grades
create_service_db meal        meal
```

Notification ayrı (schema'sı `public`, kullanıcıya `public` üzerinde yetki
gerekiyor):

```sh
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres <<-EOSQL
  CREATE DATABASE "notification";
  CREATE USER "notification_svc" WITH PASSWORD '$SVC_PASSWORD';
  REVOKE CONNECT ON DATABASE "notification" FROM PUBLIC;
  GRANT CONNECT, TEMPORARY ON DATABASE "notification" TO "notification_svc";
EOSQL
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d notification <<-EOSQL
  CREATE EXTENSION IF NOT EXISTS "pgcrypto";
  ALTER SCHEMA public OWNER TO "notification_svc";
EOSQL
```

Dosyayı çalıştırılabilir yap: `chmod +x` (git mode bit'i commit'lenmeli).

### 2. Eski init script'ini koru

`postgres/init-dbs.sql` ve `postgres/init-notification-db.sql` **silinmiyor**.
Monolith Faz 4'e kadar `mydreamcampus` DB'sini kullanmaya devam ediyor.
Silinmeleri Faz 8'de.

Entrypoint dosyaları alfabetik sırayla çalıştırır — isimlendirme compose'daki
mount adına bağlı, sıralamayı orada kontrol et.

### 3. Compose'da init script'ini mount et ve env ekle

`docker-compose.yml` içindeki `postgres` servisine:

```yaml
    environment:
      # ... mevcutlar ...
      SERVICE_DB_PASSWORD: ${SERVICE_DB_PASSWORD:?SERVICE_DB_PASSWORD is required — see .env.example}
    volumes:
      - postgres-data:/var/lib/postgresql
      - ./postgres/init-dbs.sql:/docker-entrypoint-initdb.d/00-init.sql:ro
      - ./postgres/init-databases.sh:/docker-entrypoint-initdb.d/01-service-dbs.sh:ro
```

`.env.example`'a ekle:

```
# Servis DB kullanıcılarının ortak parolası (auth_svc, staff_svc, ...).
# Izolasyon parolayla değil, DB CONNECT yetkisiyle sağlanır.
SERVICE_DB_PASSWORD=CHANGE_ME
```

`make check-env` hedefi `CHANGE_ME` kalırsa zaten hata veriyor — ek iş yok.

**`openship.json`'a da ekle** — yoksa Openship deploy'u `SERVICE_DB_PASSWORD
is required` ile patlar:

```json
"SERVICE_DB_PASSWORD": { "value": "", "secret": true }
```

### 4. migrate/entrypoint.sh'ı yeniden yaz

Şu an tek `DB_URL`'e 9 modülün migration'ını uyguluyor. Yeni davranış: her
servis kendi DB'sine, **kendi kullanıcısıyla**.

```sh
: "${PG_HOST:=postgres}"
: "${SERVICE_DB_PASSWORD:?}"

svc_url() {   # $1 = db adı  →  o servisin DSN'i
  echo "postgres://${1}_svc:${SERVICE_DB_PASSWORD}@${PG_HOST}:5432/${1}?sslmode=disable"
}

# modül dizini : hedef DB adı
migrate_pairs="auth:auth staff:staff student:student \
course_catalog:catalog enrollment:enrollment attendance:attendance \
grades:grades meal:meal"

for pair in $migrate_pairs; do
  module="${pair%%:*}"; db="${pair##*:}"
  dir="/migrations/modules/$module"
  [ -d "$dir" ] || continue
  url="$(svc_url "$db")"
  wait_for_db "$db database" "$url"
  echo ">> goose up — $module → db:$db"
  goose -dir "$dir" -table "goose_db_version_$module" postgres "$url" up
done

url="$(svc_url notification)"
wait_for_db "notification database" "$url"
goose -dir /migrations/notification -table goose_db_version_notification postgres "$url" up
```

Korunacaklar:
- Mevcut `wait_for_db()` fonksiyonu (`pg_isready` ile) — **silme**, dosyadaki
  yorum neden gerekli olduğunu açıklıyor (Openship `depends_on` condition'ı
  düşürüyor)
- `set -e`
- goose version tablosu adları (`goose_db_version_<module>`) — değiştirme

Eski `DB_URL` / `NOTIF_DB_URL` yolunu **geriye dönük destekle**: bu env'ler
set edilmişse eski davranışı da çalıştır. Monolith Faz 4'e kadar `mydreamcampus`
DB'sine ihtiyaç duyuyor.

### 5. migrate/Dockerfile — değişiklik gerekmiyor

Migration dosyaları hâlâ `monolith/internal/modules/*/sql/migrations` altında
(Faz 4'te servislere taşınacak). `COPY` satırları aynı kalır.

`payment` için `COPY` zaten yok — doğru, DB'si de yok.

### 6. Compose'da migrate servisine env ekle

```yaml
  migrate:
    environment:
      DB_URL: ...        # mevcut, monolith için — Faz 8'de silinecek
      NOTIF_DB_URL: ...  # mevcut — Faz 8'de silinecek
      PG_HOST: postgres
      SERVICE_DB_PASSWORD: ${SERVICE_DB_PASSWORD:?...}
```

---

## Volume Uyarısı — Bunu Atlama

`docker-entrypoint-initdb.d` script'leri **sadece boş volume'da ilk açılışta**
çalışır. Mevcut `postgres-data` volume'u varsa 9 DB **oluşmaz**.

İki yol var:

### Yol A — Temiz başlangıç (önerilen, homeserver dev ortamı için)

Demo verisi zaten `seed` container'ı ile admin API üzerinden yeniden yükleniyor.

```bash
# ÖNCE yedek al
sudo docker exec mydreamcampus-postgres pg_dumpall -U postgres > ~/mydreamcampus-yedek-$(date +%F).sql

# Volume'u sil ve yeniden başlat
cd new-backend/infrastructure
sudo docker compose -f docker-compose.yml -f docker-compose.standalone.yml down -v
sudo docker compose -f docker-compose.yml -f docker-compose.standalone.yml up -d
```

### Yol B — Mevcut veriyi koru

Init script'ini konteyner içinde elle çalıştır:

```bash
sudo docker cp new-backend/infrastructure/postgres/init-databases.sh \
  mydreamcampus-postgres:/tmp/init-databases.sh
sudo docker exec -e SERVICE_DB_PASSWORD=<parola> -e POSTGRES_USER=postgres \
  mydreamcampus-postgres bash /tmp/init-databases.sh
```

Ardından mevcut veriyi schema bazında taşımak istersen (opsiyonel — Faz 7 seed
ile yeniden üretilebilir):

```bash
# örnek: auth schema'sını mydreamcampus'tan auth DB'sine
sudo docker exec mydreamcampus-postgres \
  pg_dump -U postgres -d mydreamcampus -n auth --no-owner \
  | sudo docker exec -i mydreamcampus-postgres psql -U postgres -d auth
```

**Docker komutlarını sen çalıştırma** — kullanıcıya kopyala-yapıştır olarak
göster (CLAUDE.md §5).

---

## Bitiş Kriteri

Kullanıcının çalıştırması gereken doğrulamalar:

```bash
# 1. 9 DB oluşmuş mu?
sudo docker exec mydreamcampus-postgres psql -U postgres -c "\l" \
  | grep -E "auth|staff|student|catalog|enrollment|attendance|grades|meal|notification"

# 2. 9 kullanıcı oluşmuş mu?
sudo docker exec mydreamcampus-postgres psql -U postgres -c "\du"

# 3. Migration'lar uygulanmış mı? (örnek: auth)
sudo docker exec mydreamcampus-postgres psql -U postgres -d auth -c "\dt auth.*"

# 4. IZOLASYON TESTİ — bu komut HATA VERMELİ ("permission denied for database catalog")
sudo docker exec mydreamcampus-postgres \
  psql "postgres://auth_svc:<parola>@localhost:5432/catalog" -c "SELECT 1"

# 5. Monolith hâlâ ayakta mı? (eski DB'de çalışmaya devam ediyor)
curl -s localhost/health
```

**4. madde en önemlisi** — hata vermiyorsa `REVOKE CONNECT` uygulanmamıştır,
izolasyon yok demektir. Düzeltmeden fazı kapatma.

---

## Commit

```
chore(infra): provision one database and role per service on the shared postgres
```

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-1-veritabani-ayrimi-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 1 satırını `[x]` yap
3. "Sıradaki faz" satırını **2** yap
