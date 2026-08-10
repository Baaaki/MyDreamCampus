# Faz 7 — Uçtan Uca Doğrulama

**Ön koşul:** Faz 6 tamamlandı, 16 konteyner ayakta
**Risk:** Düşük (kod yazılmıyor, bulunan hatalar ilgili faza geri döner)

---

## Amaç

Mikroservis mimarisinin monolith ile **aynı davranışı** verdiğini kanıtlamak.
Yeni özellik eklenmiyor, davranış değişmiyor — değişen tek şey paketleme.

Burada bulunan her hata, **ilgili fazın dosyasına geri dönülüp** düzeltilir.
"Küçük bir fix" diye buraya yamamak faz sınırlarını bozar.

---

## Sonuç (2026-08-10) — tüm bitiş kriterleri karşılandı

Mimari monolith ile aynı davranışı veriyor. A'nın 13 adımı, B, C, D, E, F ve
G geçti. Golden path tarayıcı yerine `curl` ile koşuldu; her adım gerçek
Caddy → servis → event zincirinden geçtiği için kanıt değeri aynı.

### Bulunan yedi hata

| # | Hata | Sınıf | Commit |
|---|---|---|---|
| 1 | `SetBlacklistChecker` yalnız auth'ta çağrılıyordu; diğer 9 serviste `blacklistChecker` nil olduğu için JWTAuth revocation bloğu tümüyle atlanıyor, logout sadece auth'ta işliyordu | **Faz 4 regresyonu** | `7289872` |
| 2 | enrollment'ın catalog client'ı `limit=1000` gönderiyor, catalog `max=100` bağlaması 400 dönüyor, enrollment 500'e sarıyor → her ders seçimi düşüyordu | **Faz 3 regresyonu** | `6a4ece8` |
| 3 | `config.Load` `Reservation` bölümünü hiç okumuyordu: 0 TL rezervasyon, 0 dk zaman aşımı, QR'da `Unix()/0` → istek yolunda panic | Mevcut; bölünme görünür kıldı | `8267ff5` |
| 4 | Tekil ders sorguları `department`'ı SELECT edip struct'a kopyalamıyordu → enrollment'ın bölüm kontrolü her başvuruyu 422 ile reddediyordu | Mevcut; bölünme görünür kıldı | `bfbb2f7` |
| 5 | Seed `courses_view`'leri yalnız Güz ile dolduruyordu; Bahar programı onaylanınca FK patlayıp event DLQ'ya düşüyordu | **Faz 6 seed defekti** | `f25dadf` |
| 6 | `make deploy-meal` "no such service" veriyordu (compose adı `meal-service`) | Faz 6 defekti | `e956e09` |
| 7 | §B/§C doğrulama sorguları yanlış kolon ve şema adı kullanıyordu | Doküman | `d7305ba` |

1, 2 ve 5 bölünmenin kendi ürünü. 3 ve 4 monolith'ten geliyordu ama in-process
çağrılar handler bağlamasını atladığı için sessizdi; internal REST'e geçince
görünür hataya döndüler — bu fazın asıl kazancı bu.

### Düzeltilmeyen, kapsam dışı bırakılanlar

- `isValidSemesterFormat` `YYYY-YYYY-Fall` dayatıyor ama seed de frontend de
  `2025-2026 Güz` kullanıyor → `POST /api/semesters/:sem/courses` gerçek dönem
  adlarıyla çağrılamıyor.
- Aynı endpoint'te pgx, `course_catalog.day_of_week_enum[]` parametresini
  kodlayamıyor (`NewPostgresPool`'da `LoadType`/`AfterConnect` yok) → 500.
- Frontend `admin/scores/lock` çağırıyor, backend'de o route yok.
- auth ve meal consumer'ları başarı yolunda ctx logger'ıyla satır yazmıyor;
  correlation ctx'te var ama loglarda görünmüyor.

Dördü de monolith commit'lerinden geliyor, migrasyonla ilgisi yok.

### Bölüm bölüm kanıt

| Bölüm | Sonuç |
|---|---|
| A | 13/13 — aşağıdaki tablo |
| B | Period projeksiyonu 3 serviste dolu; `students_view` 10/10/10; auth projeksiyonu öğrenci+öğretmen sayısıyla tutuyor; `grade.appeal_processed` 3 sn'de catalog `audit_log`'una düştü |
| C | Sekiz outbox da 0 bekleyen; 16 DLQ boş |
| D1 | meal kapalı: `/api/meals` 502, catalog 200, staff 401 |
| D2 | Tek `X-Request-ID` **beş serviste** (student, enrollment, catalog, grades, attendance). Not: `correlation_id` UUID kolonu — UUID olmayan client id'si event tarafında düşer |
| D3 | 1-2 idempotency (aynı gövde → aynı id, farklı gövde → 422); 3 breaker 6. istekte 503 + `closed→open` logu, panic yok; 4 half-open 35 sn sonra toparladı; 5 404 breaker'ı açmıyor |
| D4 | grades restart'ı boyunca 502 yok — `lb_try_duration` çalışıyor |
| D5 | DLQ mekanizması doğru çalıştı: 3 denemeden sonra park etti, kuyruk tıkanmadı |
| E1 | Dört internal path de 404 |
| E2 | `grades_svc→auth`, `meal_svc→grades`, `student_svc→payment` üçü de reddedildi |
| E3 | Tokensiz 401, aynı token'la meal/attendance/staff/student kabul |
| E4 | Logout sonrası beş serviste de 401 (catalog `/courses` bilinçli public) |
| E5 | 120 istek catalog+grades'e dağıtıldı, **başka üç servis de 429** — kova bölünmüyor |
| F | Toplam 382 MiB (beklenen ~760), `/api/catalog/courses` 4.5 ms |
| G | backend tümü `ok`, frontend 56/56, mobile 62/62 |

### Bir dahaki koşuş için

- §B/§C sorguları düzeltildi: `audit_log` kolonları `service`/`action`/`timestamp`;
  outbox şeması DB adıyla aynı, **tek istisna catalog → `course_catalog`**; kolon
  adı yedi serviste `processed_at`, yalnız meal'de `published_at`.
- Dönem fan-out'u `UpdatePeriod` ile tetiklenmez (o yalnız audit yazar) —
  `POST /internal/periods/republish` kullan.
- Catalog'da dönem başına dört `period_type` satırı var; `/admin/periods`
  yalnız `catalog` tipini gösterir.

---

## A — Golden Path (kullanıcı akışı)

Tarayıcıda sırayla, her adımda **hem sonuç hem network sekmesi** kontrol
edilir:

| # | Adım | Hangi servisleri sınar | Sonuç |
|---|---|---|---|
| 1 | Admin login | auth + Redis + JWT | [x] 200, request-id header'da |
| 2 | Personel (öğretim üyesi) ekle | staff → event → auth user projection | [x] 201 → öğretmen aynı id ile login olabildi |
| 3 | Öğrenci ekle | student → staff (HTTP, danışman doğrulama) → event → auth | [x] 201, `advisor_name` HTTP ile çözüldü |
| 4 | Ders kataloğuna ders ekle | catalog → staff (HTTP, instructor doğrulama) | [x] 201; olmayan instructor → 404 `INSTRUCTOR_NOT_FOUND` |
| 5 | Dönem oluştur + aktifleştir | catalog → **period event fan-out** | [x] republish → üç projeksiyon da yeniden yazıldı |
| 6 | Öğrenci login + ders seçimi | enrollment → student + catalog (HTTP), period kilidi | [x] 200, Bahar penceresinde CENG201 listelendi |
| 7 | Danışman onayı | enrollment → event → attendance + grades view sync | [x] ön koşulsuz 422, sınıf altı 422, uygun öğrenci `approved` |
| 8 | Öğretmen yoklama oturumu aç + QR | attendance → catalog (HTTP, SemesterInfo) + Redis | [x] 201 |
| 9 | Mobilden QR okut | attendance Redis buffer → BufferFlusher → DB | [x] 200, kayıt 3 sn'de DB'ye |
| 10 | Not girişi | grades → catalog (HTTP) + audit **event** | [x] 201; audit `grade.appeal_processed` catalog'a ulaştı |
| 11 | Not finalize (bağıl) | grades self-loop event (`grade.finalize.requested`) | [x] `is_finalized: true`, ders `completed_courses`'a geçti |
| 12 | Yemek rezervasyonu | meal → payment (HTTP) → `payment.completed` event → confirm | [x] 200, 15 TL, 2 sn'de `pending`→`confirmed` |
| 13 | Şifre sıfırlama talebi | auth → event → notification → MailHog | [x] 2 sn'de MailHog'a düştü |

**13. adım özellikle önemli:** notification zaten ayrı servisti, event zinciri
bozulmadıysa mimarinin async tarafı sağlam demektir.

---

## B — Projeksiyon Doğrulaması

Servisler artık ayrı DB'lerde. Event'lerin karşı tarafa **gerçekten** ulaştığını
DB seviyesinde doğrula:

```bash
# Dönem projeksiyonu (Faz 2) — üçü de dolu olmalı
for db in enrollment grades attendance; do
  echo "--- $db"
  sudo docker exec mydreamcampus-postgres psql -U postgres -d $db \
    -c "SELECT semester, period_start, period_end FROM $db.academic_periods;"
done

# View tabloları (mevcut mimari)
sudo docker exec mydreamcampus-postgres psql -U postgres -d attendance \
  -c "SELECT count(*) FROM attendance.students_view;"
sudo docker exec mydreamcampus-postgres psql -U postgres -d grades \
  -c "SELECT count(*) FROM grades.students_view;"
sudo docker exec mydreamcampus-postgres psql -U postgres -d meal \
  -c "SELECT count(*) FROM meal.students_view;"

# auth user projection — staff/student sayısıyla tutmalı
sudo docker exec mydreamcampus-postgres psql -U postgres -d auth \
  -c "SELECT role, count(*) FROM auth.users GROUP BY role;"

# Audit event'i catalog'a ulaştı mı (Faz 3)
sudo docker exec mydreamcampus-postgres psql -U postgres -d catalog \
  -c "SELECT service, action, timestamp FROM course_catalog.audit_log ORDER BY timestamp DESC LIMIT 5;"
```

Boş kalan varsa: RabbitMQ management UI'da (`localhost:15672`) o kuyruğun
derinliğine bak. Mesaj birikmişse consumer çalışmıyor; kuyruk boşsa binding
eksik.

---

## C — Outbox Sağlığı

Her serviste outbox'ın boşaldığını doğrula — birikiyorsa publisher kopuk:

```bash
# Şema adı DB adıyla aynı, tek istisna catalog → course_catalog.
# Kolon adı yedi serviste processed_at, yalnız meal'de published_at.
for pair in auth:auth staff:staff student:student catalog:course_catalog \
            enrollment:enrollment attendance:attendance grades:grades; do
  db=${pair%%:*}; schema=${pair##*:}
  echo -n "$db: "
  sudo docker exec mydreamcampus-postgres psql -U postgres -d $db -tA \
    -c "SELECT count(*) FROM $schema.outbox_events WHERE processed_at IS NULL;"
done
echo -n "meal: "
sudo docker exec mydreamcampus-postgres psql -U postgres -d meal -tA \
  -c "SELECT count(*) FROM meal.outbox_events WHERE published_at IS NULL;"
```

Birkaç saniyede sıfıra inmeli. Kalıcı olarak artıyorsa o servisin
`OutboxWorker`'ı başlatılmamıştır (Faz 4, adım 4).

---

## D — Dayanıklılık Testleri

Mikroservisin asıl kazancı burada görünür:

```bash
# 1. Bir servisi durdur — diğerleri ayakta kalmalı
sudo docker stop mydreamcampus-meal
curl -s -o /dev/null -w "%{http_code}\n" localhost/api/meals      # 502
curl -s -o /dev/null -w "%{http_code}\n" localhost/api/grades     # 401 (çalışıyor)
sudo docker start mydreamcampus-meal

# 2. Sync bağımlılık kopuk — çağıran servis 5xx dönmeli, ÇÖKMEMELİ
sudo docker stop mydreamcampus-staff
#   → catalog'da ders oluşturmayı dene: hata mesajı gelmeli
sudo docker logs --tail 20 mydreamcampus-catalog   # panic OLMAMALI
sudo docker start mydreamcampus-staff

# 3. RabbitMQ kopuk — event'ler outbox'ta birikmeli, kayıp olmamalı
sudo docker stop mydreamcampus-rabbitmq
#   → personel ekle (yazma başarılı olmalı, outbox'a düşer)
sudo docker start mydreamcampus-rabbitmq
#   → 30 sn sonra outbox boşalmalı, auth.users'a yansımalı

# 4. Servis yeniden başlatma — tek servis, diğerlerini etkilemeden
sudo docker restart mydreamcampus-grades
#   → restart SIRASINDA /api/grades'e istek at: Caddy lb_try_duration
#     sayesinde 502 DEĞİL, normal yanıt gelmeli (04-PROD-HAZIRLIK.md §4)

# 5. Poison message DLQ'ya düşüyor mu (04-PROD-HAZIRLIK.md §1)
#    Bir kuyruğa kasten bozuk payload yayınla (RabbitMQ UI → Publish message).
#    Beklenen: N denemeden sonra DLQ'ya düşer, kuyruk tıkanmaz, CPU yanmaz.
sudo docker exec mydreamcampus-rabbitmq rabbitmqctl list_queues name messages | grep dlq
```

**2. maddede panic görürsen** HTTP client'ta nil kontrolü eksiktir → Faz 3'e dön.
**3. maddede event kaybı varsa** outbox transaction sınırı bozulmuştur → Faz 4'e dön.

---

## D3 — Circuit Breaker ve Idempotency

`05-DAYANIKLILIK.md` "Doğrulama" bölümündeki 5 testi çalıştır.

En kritik olanı **5. test**: olmayan bir id ile 10 istek at, breaker
**açılmamalı**. Açılıyorsa `isFailure` 404'ü hata sayıyor demektir — o haliyle
breaker, olmamasından kötüdür (normal kullanımda çalışan servisleri ölü ilan
eder).

---

## D2 — Uçtan Uca İzlenebilirlik

`03-IZLENEBILIRLIK.md` "Doğrulama" bölümünü çalıştır.

Başarı kriteri: öğrenci ekleme gibi çok servisli bir akışta tek
`X-Request-ID` ile **beş servisin** logu ve outbox satırları bulunabilmeli
(student → event → auth + attendance + grades + meal).

Zincir kopuyorsa hangi halkada koptuğunu bul — `03-IZLENEBILIRLIK.md`'deki
[2]-[7] numaraları hangi fazın hangi adımına döneceğini söylüyor.

---

## E — Güvenlik Kontrolleri

`02-GUVENLIK.md` faz tablosundaki **tüm** kapıları burada bir kez daha koştur.
Aşağıdakiler o listenin en kritik dördü:

```bash
# 1. Internal route'lar dışarıdan erişilemiyor
curl -s -o /dev/null -w "%{http_code}\n" localhost/internal/staff/x        # 404
curl -s -o /dev/null -w "%{http_code}\n" localhost/api/staff/internal/x    # 404

# 2. DB izolasyonu — HATA vermeli
sudo docker exec mydreamcampus-postgres \
  psql "postgres://grades_svc:<parola>@localhost:5432/auth" -c "SELECT 1"

# 3. JWT tüm servislerde geçerli (ortak HS256 secret)
#    → aynı token ile /api/grades ve /api/meals'a istek at, ikisi de kabul etmeli

# 4. Logout sonrası token her serviste reddedilmeli (Redis blacklist ortak)
#    → logout ol, eski token ile /api/attendance dene → 401
```

```bash
# 5. Rate limit kovası ortak mı (02-GUVENLIK.md A07)
#    Aynı IP'den iki FARKLI servise dağıtılmış istekler ORTAK limite takılmalı.
#    Ayrı ayrı limite takılıyorsa ServiceName sabiti Faz 4'te yanlış verilmiş.
for i in $(seq 1 200); do
  curl -s -o /dev/null localhost/api/catalog
  curl -s -o /dev/null localhost/api/grades
done
curl -s -o /dev/null -w "%{http_code}\n" localhost/api/meals   # 429 bekleniyor
```

**4. madde kritik:** blacklist Redis'te ortak. Bir servis blacklist'i kontrol
etmiyorsa logout o servis için işlemiyor demektir.

**5. madde migrasyonun getirdiği regresyonu yakalar** — 429 gelmiyorsa
saldırgan isteği servislere yayarak IP limitini 9 katına çıkarabiliyor.

---

## F — Performans / Kaynak

```bash
sudo docker stats --no-stream --format "table {{.Name}}\t{{.MemUsage}}\t{{.CPUPerc}}"
```

Beklenen: toplam ~760 MB. 1.2 GB üzerindeyse Faz 6'daki `mem_limit` ve pgx
pool ayarlarına dön.

Latency karşılaştırması (monolith'e göre +2-5ms bekleniyor):

```bash
curl -s -o /dev/null -w "toplam: %{time_total}s\n" localhost/api/catalog/courses
```

---

## G — Test Süitleri

```bash
make test
```

`test-backend` Faz 4'te `./services/...` olarak güncellenmişti. Frontend ve
mobile testleri değişmemiş olmalı — değiştiyse gereksiz yere frontend'e
dokunulmuş demektir.

---

## Bitiş Kriteri

- A bölümündeki 13 adımın hepsi `[x]`
- B, C bölümlerinde boş tablo yok
- D bölümünde panic yok, event kaybı yok
- E bölümünde 4 kontrol de geçti
- `make test` yeşil

---

## Commit

Bu faz kod değiştirmez. Bulunan hatalar ilgili fazın scope'uyla commit'lenir.
Sadece doküman güncellemesi varsa:

```
docs(infra): record microservices end-to-end verification results
```

---

## Faz Sonu

1. Bu dosyayı yeniden adlandır: `faz-7-e2e-dogrulama-TAMAMLANDI.md`
2. `00-BASLANGIC.md` durum tablosunda Faz 7 satırını `[x]` yap
3. "Sıradaki faz" satırını **8** yap
