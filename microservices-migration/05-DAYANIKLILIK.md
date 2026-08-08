# Dayanıklılık — Circuit Breaker ve Idempotency

> Faz dosyası değil, **iki mekanizmanın tarifi**. Faz 3, 4, 7'de atıf var.

---

## Bölüm A — Idempotency: Üç Katman

Idempotency tek bir şey değil, üç ayrı katman. İkisi zaten var, biri eksik.

| Katman | Neyi korur | Mekanizma | Durum |
|---|---|---|---|
| **1. HTTP** | Client retry'ı → çift kayıt | `Idempotency-Key` header + Redis | **YOK** — bu dosyada ekleniyor |
| **2. Event** | Aynı event'in iki kez işlenmesi | `processed_events` tablosu + `event_id` | **VAR** — 8 modülde |
| **3. DB kısıtı** | Yarış koşulu → çift satır | UNIQUE index, advisory lock | **VAR** — kritik yollarda |

Katman 2 ve 3 mevcut ve migrasyonda korunuyor (`01-REFERANS-MIMARI.md` §3,
Faz 2/3/4). Bu bölüm katman 1'i anlatıyor.

### Neden eksik olması önemli

Mobil uygulama zayıf şebekede yemek rezervasyonu yapıyor. İstek sunucuya
ulaşıyor, işleniyor, **yanıt dönerken bağlantı kopuyor**. Kullanıcı tekrar
deniyor. Sonuç: iki rezervasyon, iki ödeme.

Bu monolithte de vardı — migrasyonun getirdiği bir sorun değil. Ama
mikroserviste zincir uzuyor (Caddy → servis → HTTP → event) ve timeout
ihtimali artıyor.

### Tasarım

Yeni dosya: `shared/platform/middleware/idempotency.go`

```
Idempotency-Key header (client üretir, UUID v4)
   │
   ├─ header YOK ────────────────────► geç (mevcut davranış korunur)
   │
   ├─ kayıt YOK ──► SETNX "in-flight" ──► handler çalışır
   │                                      └─► yanıtı Redis'e yaz (TTL 24s)
   │
   ├─ kayıt VAR, tamamlanmış ────────► saklanan yanıtı aynen dön
   │
   ├─ kayıt VAR, in-flight ──────────► 409 "İşleminiz sürüyor, lütfen bekleyin"
   │
   └─ kayıt VAR, gövde hash'i FARKLI ► 422 "Bu anahtar farklı bir istekle kullanıldı"
```

Redis anahtarı: `idem:<servis>:<user_id>:<key>`
— `user_id` dahil, çünkü bir kullanıcının anahtarı başkasının yanıtını
okuyamamalı.

Saklanan değer: `{body_sha256, status_code, response_body}`. TTL **24 saat**.

Kurallar:
- Sadece **POST / PUT / PATCH**. GET ve DELETE zaten idempotent.
- **Global middleware DEĞİL** — işaretli route'larda. Her isteğe Redis
  round-trip eklemek gereksiz.
- Redis erişilemezse **fail-open** (isteği geçir). Idempotency bir güvenlik
  kontrolü değil, çift kayıt koruması; Redis düştü diye rezervasyon almayı
  durdurmak daha kötü. (Rate limit'ten farkı bu — orada login fail-closed.)

### Hangi endpoint'ler

Öncelik: para ve kota etkileyenler.

| Servis | Endpoint | Neden |
|---|---|---|
| meal | rezervasyon oluştur | **Para** — çift ödeme |
| meal | rezervasyon iptal / iade | Çift iade |
| enrollment | program gönder | **Kota** — kontenjan |
| enrollment | danışman onay / red | Çift onay |
| grades | not girişi (tekil + bulk) | Veri bozulması |
| attendance | manuel yoklama | Çift kayıt |
| student, staff | oluşturma | Çift kişi kaydı |

QR tarama (`attendance/scan`) **listede yok** — Redis `SADD` ile zaten atomik
dedup yapıyor, ikinci katman gereksiz.

### Client tarafı — migrasyonda DEĞİŞMİYOR

`00-BASLANGIC.md` "Frontend / Mobil hiç değişmiyor" diyor ve bu **korunuyor**.
Middleware header yoksa isteği aynen geçiriyor, yani:

- Migrasyon sırasında: sunucu tarafı hazır, client hiç dokunulmuyor
- Sonra (ayrı iş kalemi): `frontend/src/lib/api-client.ts` ve
  `mobile/services/` mutasyon isteklerine `crypto.randomUUID()` ile header ekler

Bu ayrım bilinçli — client değişikliği migrasyonun kapsamını ve test yükünü
büyütür, üstelik sunucu hazır olmadan zaten anlamsız.

---

## Bölüm B — Circuit Breaker

> Not: `04-PROD-HAZIRLIK.md` §5 önceki halinde "circuit breaker ekleme" diyordu.
> Kullanıcı kararıyla **ekleniyor**; o bölüm bu dosyaya yönlendirilmek üzere
> güncellendi.

### Ne zaman devreye giriyor

Yedi sync seam var (`01-REFERANS-MIMARI.md` §2). Bir servis ölü veya çok
yavaşsa, breaker olmadan çağıran servis her istekte 10 saniye timeout bekler:

- Kullanıcı 10 sn beekliyor, sonra hata alıyor
- Çağıran servisin goroutine ve bağlantı havuzu doluyor
- Kendi istekleri de yavaşlıyor → **kaskad arıza**

Breaker, hedefin ölü olduğunu öğrendikten sonra **beklemeden** hata dönüyor.

### Kütüphane: `sony/gobreaker` — onaylandı

Kullanıcı onayı alındı (CLAUDE.md §6). Kendi state machine'imizi yazmak yerine
bu tercih edildi: yarım saatlik iş gibi görünen half-open geçişi klasik yarış
koşulu tuzağıdır ve testini yazmak kütüphaneyi eklemekten pahalıdır.

```bash
cd new-backend/shared && go get github.com/sony/gobreaker/v2
```

- **Sürümü `go get` belirlesin**, dokümana sabitlenmedi.
- **v1 / v2 farkına dikkat:** v2 generic API kullanıyor
  (`NewCircuitBreaker[*http.Response]`), v1 kullanmıyor. Hangisi geldiyse ona
  göre yaz — v2 gelirse tip parametresi zorunlu.
- `shared/go.mod`'a giriyor, servisler `shared` üzerinden alıyor — 10 ayrı
  `go.mod`'a tek tek eklenmiyor.
- Transitive bağımlılık getirmiyor; `go mod tidy` sonrası `go.sum` tek satır
  büyümeli. Daha fazlaysa yanlış paketi almışsındır.

Yazacağımız kısım sadece `isFailure` predicate'i ve loglama (~40 satır):

```go
b := gobreaker.NewCircuitBreaker[*http.Response](gobreaker.Settings{
    Name:        "staff-service",
    MaxRequests: 1,                 // half-open sonda sayısı
    Timeout:     30 * time.Second,  // açık kalma süresi
    ReadyToTrip: func(c gobreaker.Counts) bool {
        return c.ConsecutiveFailures >= 5
    },
    // Sessizce açılan breaker teşhisi en zor arızalardan biri.
    OnStateChange: func(name string, from, to gobreaker.State) {
        logger.Warn("circuit breaker state change",
            zap.String("target", name),
            zap.String("from", from.String()),
            zap.String("to", to.String()))
    },
})
```

`MaxRequests`, `Timeout` ve `ReadyToTrip` eşiği config'ten okunsun
(`CIRCUIT_BREAKER_*`), yukarıdaki sabitler sadece varsayılan.

### Yerleşim

`shared/client/base.go` — tek yer. Yedi seam de oradan geçtiği için bir kez
yazılıyor, hepsi kazanıyor.

**Breaker başına kapsam: hedef servis.** Endpoint başına değil — staff servisi
ölüyse `/internal/staff/:id` de `/internal/staff?department=` de ölüdür.

```go
// Her hedef servis için ayrı breaker. Bir servisin çökmesi diğerine
// giden çağrıları kesmemeli.
type Base struct {
    baseURL string
    secret  string
    http    *http.Client
    breaker *breaker.Breaker   // bu hedefe özel
}
```

### Eşikler

| Parametre | Değer | Gerekçe |
|---|---|---|
| Ardışık hata → aç | **5** | Tek bir geçici hata breaker'ı açmamalı |
| Açık kalma süresi | **30 sn** | Konteyner yeniden başlaması ~5-10 sn; 30 sn güvenli pay |
| Half-open sonda sayısı | **1** | Tek istek dener; başarılıysa kapanır, değilse tekrar 30 sn açık |

Bu değerler config'ten okunsun (`CIRCUIT_BREAKER_*`), hardcode edilmesin —
homeserver'da ayar gerekebilir.

### Neyin "hata" sayıldığı — en kritik detay

```go
// 404 ve 4xx HATA DEĞİL. "Öğrenci bulunamadı" geçerli bir iş cevabıdır;
// hata sayılırsa normal kullanımda breaker açılır ve çalışan servisi
// ölü ilan ederiz.
func isFailure(resp *http.Response, err error) bool {
    if err != nil {
        return true              // timeout, connection refused, DNS
    }
    return resp.StatusCode >= 500
}
```

Bu kuralı yanlış yazmak, breaker'ı olmamasından **daha kötü** hale getirir.

### Breaker açıkken ne dönüyor

`platform/errors.AppError` üzerinden 503:

```
HTTP 503
{"error": "Servis şu anda yanıt vermiyor, lütfen birazdan tekrar deneyin"}
```

Türkçe mesaj, İngilizce log (CLAUDE.md §3).

### Güvenlikle etkileşim — atlanmaması gereken nokta

`02-GUVENLIK.md` A04: dönem kontrolleri catalog erişilemezse **fail-open**
davranıyor. Breaker bu durumu **daha hızlı ve daha sık** tetikler — catalog 5
hata sonrası 30 saniye boyunca "ölü" sayılır ve o süre boyunca her çağrı
anında fail-open'a düşer.

Faz 2'den sonra dönem penceresi kontrolleri **lokal projeksiyondan** okunuyor,
yani breaker'dan etkilenmiyor. Ama `SemesterInfo` (hard deadline) hâlâ HTTP
üzerinden geliyor.

**Kural:** Breaker açıkken `SemesterInfo` alınamıyorsa "hard deadline yok"
varsayma — **isteği reddet**. Hard deadline'ı admin bile aşamıyor
(`02-GUVENLIK.md` A04, `platform/rules` üç katmanlı kilit); erişilemezlik onu
gevşetmenin gerekçesi olamaz.

### Gözlemlenebilirlik

Durum geçişleri **log'lanmalı** — sessizce açılan bir breaker, teşhisi en zor
arızalardan biri:

```go
logger.Warn("circuit breaker opened",
    zap.String("target", "staff-service"),
    zap.Int("consecutive_failures", 5))
```

`/metrics` eklendiğinde breaker durumu ilk metriklerden biri olmalı
(`00-BASLANGIC.md` gözlemlenebilirlik tablosu).

---

## Uygulama Yeri

| İş | Faz | Dosya |
|---|---|---|
| Idempotency middleware | 3 | `shared/platform/middleware/idempotency.go` |
| Idempotency route işaretleme | 4 | her servisin `module.go` `RegisterRoutes` |
| Circuit breaker `Base`'e | 3 | `shared/client/base.go` |
| Breaker config alanları | 3 | `shared/config/config.go` |
| Breaker eşikleri `.env` + compose | 6 | `.env.example`, `docker-compose.yml` |

---

## Doğrulama (Faz 7)

```bash
# 1. Idempotency — aynı anahtarla iki kez rezervasyon
KEY=$(uuidgen)
for i in 1 2; do
  curl -s -o /dev/null -w "%{http_code}\n" -X POST localhost/api/meals/reservations \
    -H "Authorization: Bearer $TOKEN" -H "Idempotency-Key: $KEY" \
    -H "Content-Type: application/json" -d '{"date":"2026-09-01","meal_type":"lunch"}'
done
# Beklenen: iki kez de 201, ve DB'de TEK rezervasyon
sudo docker exec mydreamcampus-postgres psql -U postgres -d meal -tA \
  -c "SELECT count(*) FROM meal.reservations WHERE ...;"   # 1 olmalı

# 2. Idempotency — aynı anahtar, farklı gövde → 422
curl -s -o /dev/null -w "%{http_code}\n" -X POST localhost/api/meals/reservations \
  -H "Idempotency-Key: $KEY" -d '{"date":"2026-09-02","meal_type":"dinner"}'   # 422

# 3. Circuit breaker — hedefi öldür, 5 istek at
sudo docker stop mydreamcampus-staff
for i in $(seq 1 6); do
  curl -s -o /dev/null -w "%{http_code} " localhost/api/catalog/semester-courses
done; echo
# Beklenen: ilk ~5 istek yavaş (timeout), 6.'sı ANINDA 503
sudo docker logs --tail 5 mydreamcampus-catalog | grep "circuit breaker opened"

# 4. Half-open — servisi geri aç, 30 sn bekle
sudo docker start mydreamcampus-staff
sleep 35
curl -s -o /dev/null -w "%{http_code}\n" localhost/api/catalog/semester-courses   # 200

# 5. 404 breaker'ı AÇMAMALI — olmayan id ile 10 istek
for i in $(seq 1 10); do
  curl -s -o /dev/null localhost/api/catalog/instructors/00000000-0000-0000-0000-000000000000
done
sudo docker logs --tail 20 mydreamcampus-catalog | grep -c "circuit breaker opened"   # 0 olmalı
```

**5. madde en önemlisi** — 0 değilse `isFailure` yanlış yazılmıştır ve breaker
normal kullanımda çalışan servisleri ölü ilan eder.
