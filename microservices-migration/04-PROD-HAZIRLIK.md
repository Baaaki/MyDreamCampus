# Prod Hazırlık — İşletme Boşlukları

> Faz dosyası değil, **eksik listesi**. Faz 4, 5, 6, 8'de atıf var.

Buraya kadarki dosyalar "mimari doğru mu"ya odaklandı. Bu dosya "günün sonunda
işletilebilir mi"ye bakıyor. Aşağıdakiler **mevcut kodda zaten eksik** olan
şeyler — migrasyonun getirdiği sorunlar değil, ama bölünme hepsini
**büyütüyor**: 2 konteynerlik bir sorun 16 konteynerlik hale geliyor.

---

## 1. DLQ bağlı değil — poison message sonsuz döngü [KRİTİK]

**Tespit:** `shared/platform/rabbitmq/dlq.go` ve `ConsumeWithDLQ` **yazılmış
ama hiçbir yerden çağrılmıyor.** Yedi consumer'ın hepsi düz `Consume`
kullanıyor ve hata halinde:

```go
// consumer.go:87
msg.Nack(false, true)   // requeue = true → mesaj kuyruğa geri döner
```

İşlenemeyen bir mesaj (bozuk payload, her seferinde hata veren bug) **sonsuza
kadar** kuyruğa geri döner: CPU yakar, kuyruğu tıkar, logu doldurur ve
arkasındaki sağlam mesajları geciktirir.

Monolith'te tek process'in derdiydi. Dokuz serviste hem daha olası hem fark
edilmesi daha zor.

**Yapılacak (Faz 4):** Tüm consumer'lar `ConsumeWithDLQ(queue, handler,
maxRetries)` kullansın. `SetupDLQ` her kuyruk için çağrılsın. DLQ
exchange/queue'ları `definitions.json`'a da girsin.

**Doğrulama (Faz 7):** Kasten bozuk bir mesaj yayınla — N denemeden sonra
DLQ'ya düşmeli, kuyruk tıkanmamalı.

---

## 2. Tablo şişmesi — hiçbir şey temizlenmiyor [YÜKSEK]

İki tablo sınırsız büyüyor:

| Tablo | Durum |
|---|---|
| `<schema>.outbox_events` | Satırlar `status='processed'` işaretleniyor, **hiç silinmiyor**. `outbox_worker.go`'da temizlik kodu yok. |
| `<schema>.processed_events` | Sadece **auth** temizliyor (`StartCleanupScheduler`, 30 gün). attendance, enrollment, grades, meal, student temizlemiyor. |

`PROCESSED_EVENTS_RETENTION_DAYS` config alanı var (default 30) ama auth
dışında kimse okumuyor — hatta auth bile sabit `"30 days"` string'i geçiyor,
config'i kullanmıyor.

Yoklama (sınıf başına QR taraması) ve yemek rezervasyonu yüksek hacimli
akışlar. Homeserver'da yıllar boyunca çalışacak bir sistemde bu er geç disk
ve sorgu performansı sorunu olur.

**Yapılacak (Faz 4):** Her servisin `main.go`'sunda ortak bir retention
worker'ı başlat:

```go
// shared/eventbus içinde ortak worker — 9 serviste tekrarlanmasın.
go eventbus.NewRetentionWorker(store, cfg.Timeout.ProcessedEventsRetentionDays,
    cfg.Timeout.CleanupSchedulerIntervalHours).Start(ctx)
```

- `outbox_events` → `status='processed' AND processed_at < now() - retention`
- `processed_events` → `processed_at < now() - retention`
- auth'un mevcut `StartCleanupScheduler`'ı session temizliğini yapmaya devam
  etsin; processed_events kısmını ortak worker'a bırak (çift silme zararsız
  ama tek yerden yönetmek daha iyi).

`correlation_id` index'i (bkz. `03-IZLENEBILIRLIK.md`) bu tabloların
büyümesini daha da pahalı hale getiriyor — retention onunla birlikte gelmeli.

---

## 3. Yedekleme yok [YÜKSEK]

**Tespit:** `scripts/`, `Makefile` ve `DEPLOY.md` içinde `pg_dump` geçmiyor.
Hiçbir yedekleme mekanizması yok.

Bölünmeden sonra 9 veritabanı olacak. Tek Postgres konteynerinde durdukları
için `pg_dumpall` hepsini birden alabiliyor — bu, kararımızın beklenmedik bir
faydası.

**Yapılacak (Faz 6):**

```make
DB_BACKUP_DIR ?= $(HOME)/mydreamcampus-backups

backup:
	@mkdir -p $(DB_BACKUP_DIR)
	$(SUDO) docker exec mydreamcampus-postgres pg_dumpall -U postgres \
		| gzip > $(DB_BACKUP_DIR)/all-$$(date +%F-%H%M).sql.gz
	@ls -lh $(DB_BACKUP_DIR) | tail -5

# Geri yükleme YIKICI — mevcut veritabanlarını ezer.
restore:
	@test -n "$(FILE)" || { echo "kullanim: make restore FILE=/yol/yedek.sql.gz"; exit 1; }
	gunzip -c $(FILE) | $(SUDO) docker exec -i mydreamcampus-postgres psql -U postgres
```

- Otomatik çalıştırma: `scripts/systemd/` altında zaten autodeploy timer deseni
  var — aynı desenle günlük bir `backup.timer` eklenebilir.
- **Tutarlılık uyarısı:** `pg_dumpall` veritabanı başına tutarlı, **9 DB
  arasında atomik değil**. Yedek anında uçuşta olan bir event zinciri
  (student oluşturuldu ama attendance'ın view'ına henüz yansımadı) yarım
  yakalanabilir. Event'ler idempotent olduğu için geri yüklemede kendini
  toparlar, ama bunu bil.
- `DEPLOY.md`'ye geri yükleme prosedürünü yaz (Faz 8).

---

## 4. Tek servis restart'ında 502 [ORTA]

Mikroservisin asıl kazancı tek servisi yeniden başlatabilmek. Ama Caddy'nin
upstream'i o an ölü olduğu için kullanıcı **502 görür** — yani kazanç pratikte
kullanıcıya kesinti olarak yansır.

**Yapılacak (Faz 5):**

```caddyfile
	reverse_proxy grades-service:8087 {
		# Tek servisin yeniden başlatılması kullanıcıya 502 olarak yansımasın:
		# Caddy kısa süre yeniden dener. Servis 5 sn'de ayağa kalkıyorsa istek
		# hiç düşmez.
		lb_try_duration 5s
		lb_try_interval 250ms
	}
```

**Ayrıca tek servis deploy hedefi (Faz 6):**

```make
# Tek servisi yeniden derle ve başlat — diğer 15 konteynere dokunmadan.
deploy-%:
	$(SUDO) docker compose $(COMPOSE) up -d --no-deps --build $*
```

Bu hedef olmadan herkes `make deploy` çalıştırır ve tüm stack'i yeniden
başlatır — bölünmenin operasyonel faydası kullanılmamış olur.

---

## 5. Timeout bütçesi tutarsız [ORTA]

Zincir: `Caddy → servis A → servis B`. İç timeout dış timeout'tan **kısa**
olmalı, yoksa dış taraf vazgeçtikten sonra iç çağrı boşuna çalışmaya devam
eder ve bağlantı havuzunu tüketir.

Hedef bütçe:

| Katman | Değer | Nerede |
|---|---|---|
| Caddy upstream timeout | 30 sn | Faz 5 (Caddy varsayılanı yeterli, değiştirme) |
| Servis HTTP sunucu (`REQUEST_TIMEOUT_SECONDS`) | 20 sn | `.env` |
| Servisler arası client timeout | **10 sn** | Faz 3, `base.go` — zaten belirtildi |
| pgx sorgu / RabbitMQ publish | client'tan kısa | mevcut değerler yeterli |

**Circuit breaker ekleniyor** — kullanıcı kararı. Tarifi, eşikleri ve
"neyin hata sayıldığı" kuralı: `05-DAYANIKLILIK.md` Bölüm B. Timeout bütçesi
breaker'ın önkoşulu: breaker ancak timeout'lar tutarlıysa doğru çalışır.

---

## 6. CI'da uçtan uca duman testi yok [ORTA]

Faz 7 bir **kerelik manuel** doğrulama. Otomatik hale gelmezse üç ay sonra
golden path sessizce bozulur — ve 10 servisle bozulmanın hangi serviste
olduğunu bulmak zorlaşır.

**Yapılacak (Faz 8):** `.github/workflows/ci.yml` içinde compose ile tüm
stack'i ayağa kaldıran bir job:

```
docker compose up -d --build
→ caddy /health bekle
→ admin login → personel ekle → ders ekle → öğrenci ekle → ders seç
→ her adımda HTTP kodu doğrula
→ docker compose logs (hata durumunda artifact olarak yükle)
```

Mevcut `ci.yml`'de zaten "start monolith → wait for health → integration"
bloğu var (satır ~182-208); onu compose tabanlıya çevirmek sıfırdan yazmaktan
kolay.

---

## 7. Frontend kısmi kesintiyi bilmiyor [DÜŞÜK — kapsam dışı, ama bilinsin]

Monolith'te sistem ya tamamen ayaktaydı ya tamamen ölüydü. Artık **grades
ölüyken meal çalışıyor** olabilir. SPA bu durumu muhtemelen "her şey bozuk"
gibi gösterir.

Bu migrasyonun kapsamında **değil** — frontend'e dokunmama kararı bilinçli.
Ama yeni bir kullanıcı deneyimi durumu doğuyor, Faz 8'de `README.md`'ye not
düş ve ayrı iş kalemi olarak aç.

---

## Kapsam Dışı (bilerek — migrasyona ekleme)

| Ne | Neden şimdi değil |
|---|---|
| Prometheus / Grafana / Loki | `00-BASLANGIC.md` gözlemlenebilirlik tablosu — seam'ler hazır, kurulum sonra |
| OpenTelemetry | Correlation ID zinciri kurulduktan sonra doğal adım |
| Idempotency-Key'in **client tarafı** | Sunucu middleware'i Faz 3'te hazır olacak ve header'sız isteği geçirecek; frontend/mobil değişikliği ayrı iş (`05-DAYANIKLILIK.md` Bölüm A) |
| Servisler arası mTLS | Tek makine, compose bridge network (`02-GUVENLIK.md` A02) |
| Rolling deploy / zero-downtime | compose ile gerçek anlamda mümkün değil; Swarm/k8s gerekir. `lb_try_duration` (§4) pratikte yeterli. |
| Yatay ölçekleme (replica) | Outbox worker ve bazı worker'lar tekil çalışacak şekilde yazılmış; replica öncesi leader election gerekir |

Son satır önemli: **servisleri şimdilik `replicas: 1`'de tut.** Outbox
worker'ı iki replikada koşarsa aynı event iki kez publish edilir (consumer'lar
idempotent olduğu için felaket değil, ama gereksiz). Ölçekleme ayrı bir iş.

---

## Öncelik Sırası

Migrasyon içinde yapılacaklar, önem sırasına göre:

| # | Konu | Faz | Neden migrasyon içinde |
|---|---|---|---|
| 1 | DLQ bağlama | 4 | Consumer wiring'ine zaten dokunuluyor |
| 2 | Retention worker | 4 | `main.go`'lar zaten yazılıyor |
| 3 | Yedekleme hedefi | 6 | 9 DB'ye çıkmadan önce prosedür otursun |
| 4 | Caddy retry + tek servis deploy | 5, 6 | 2'şer satır |
| 5 | Timeout bütçesi | 3, 6 | Sadece değer tutarlılığı |
| 6 | CI duman testi | 8 | Golden path'in çürümesini engeller |
