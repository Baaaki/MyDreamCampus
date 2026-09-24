# Faz 3 — Sahte kartla ödeme

## Başlamadan
- Oku: `new-backend/skills.md` (§5 servis kayıt yerleri, §6 outbox),
  `frontend/skills.md`, `mobile/skills.md`.
- Bağımlılık yok.

## Bağlam
- `services/payment-service` veritabanı olmadan çalışıyor. `InitiatePayment`
  2 sn sonra bir goroutine'den **doğrudan** `payment.completed` yayınlıyor
  (`internal/service/payment_service.go` ≈110). Kullanıcı hiçbir şey yapmadan
  ödeme "başarılı" oluyor; outbox kuralı da ihlal ediliyor.
- Web ödeme diyaloğu var olmayan
  `https://mock-payment.mydreamcampus.com/...` adresine yönlendiriyor
  (`frontend/src/pages/student/cafeteria/index.tsx` ≈141-560).
- Mobilde ödeme adımı hiç yok (`mobile/app/cafeteria.tsx`,
  `mobile/hooks/useMeals.ts:41`).
- meal `payment.completed` ve `payment.failed` event'lerini zaten dinliyor
  (`services/meal-service/internal/worker/payment_consumer.go`). Payload
  tanımı: `services/meal-service/internal/dto/event_dto.go` ≈55-78.
  **Event payload'ı değişmeyecek.**

## Hedef akış
1. Öğrenci rezervasyon veya toplu rezervasyon oluşturur. meal `pending`
   rezervasyon yazar ve `/internal/payments/initiate` çağırır. payment
   `pending` bir ödeme kaydı oluşturur. meal yanıtında `payment_url` yerine
   `payment_id` ve tutar döner.
2. Arayüz kart formunu gösterir ve `POST /api/payments/:payment_id/confirm`
   çağırır.
3. payment kartı doğrular; ödeme durumunu günceller ve outbox'a
   `payment.completed` / `payment.failed` yazar (ikisi aynı transaction'da).
4. meal consumer rezervasyonu onaylar veya iptal eder. Arayüz rezervasyon
   durumunu yoklayarak sonucu gösterir.
5. Ödeme yapılmazsa meal'in mevcut zaman aşımı worker'ı rezervasyonu düşürür;
   payment kaydı da `expired` olur.

### Test kartları
UI'da "Demo ödeme: gerçek para çekilmez" notuyla birlikte gösterilir.

| Kart | Sonuç |
|---|---|
| 4242 4242 4242 4242 | Başarılı |
| 4000 0000 0000 0002 | Reddedildi: "Kart reddedildi" |
| 4000 0000 0000 9995 | Reddedildi: "Yetersiz bakiye" |
| Diğer Luhn-geçerli kartlar | Başarılı |

Luhn geçersizse, son kullanma tarihi geçmişse (iş saati `clock.Now()`) veya
CVC 3-4 hane değilse → 422 doğrulama hatası. Bu durumda ödeme `failed` olmaz,
öğrenci tekrar deneyebilir.

### Kart verisi güvenliği
Kart numarası ve CVC hiçbir yerde **saklanmaz, loglanmaz, event'e girmez**.
Yalnız `card_brand` (prefix'ten: Visa, Mastercard, Amex, Troy `9792`) ve
`card_last4` tutulur.

## Görevler

- [x] **3.1 payment veritabanı altyapısı** — `cb011cd`
  - `new-backend/infrastructure/postgres/init-databases.sh`:
    `create_service_db payment payment`.
  - `new-backend/infrastructure/migrate/Dockerfile` ve `entrypoint.sh`:
    payment migration'larını ekle.
  - `new-backend/infrastructure/docker-compose.yml` payment-service: `DB_URL`
    (`payment_svc`, diğer servislerle aynı kalıp).
  - `services/payment-service/cmd/main.go`:
    - `bootstrap.Options.NeedsDatabase: true`
    - `StartOutbox("payment.events", ...)`
    - `StartRetention(...)`
  - payment-service'e ekle: `sqlc.yaml`, `Makefile` (diğer servislerden
    uyarla), `internal/sql/{migrations,queries}`, generated `internal/db`.
  - `.github/workflows/ci.yml`: payment'ın matriste olduğunu ve sqlc/migration
    adımlarının onu kapsadığını kontrol et.
  - Not: `init-databases.sh` yalnız boş volume'da çalışır. Mevcut volume'u olan
    kullanıcıya elle provision komutunu göster (`migrate/entrypoint.sh`'in
    hata mesajındaki gibi). Migration'ı kendin ÇALIŞTIRMA.
  - **Commit:** `chore(payment): give the payment service its own database`
  > Not (24.09): `StartOutbox` outbox tablosu olmadan çalışamadığı için
  > `payment.outbox_events` (migration, sorgular, repository/store/retention)
  > bu göreve alındı; 3.2'de yalnız `payment.payments` kaldı.
  > `payment.processed_events` eklenmedi: payment event tüketmiyor (staff ile
  > aynı durum), retention'daki karşılığı no-op. Mevcut volume için
  > `init-databases.sh` artık `PROVISION_ONLY="payment"` ile yalnız istenen
  > veritabanını kurabiliyor; migrate eksik veritabanlarını bu komutla
  > listeliyor (DEPLOY.md sorun giderme). CI matrisinde payment zaten var;
  > CI'da sqlc adımı yok, migration e2e'deki migrate konteyneriyle koşuyor.

- [x] **3.2 Şema** — `e9ca176`
  - `payment.payments`:
    - id uuid pk (`uuidv7()`), reference_id text unique, student_id uuid
    - amount numeric(10,2), currency char(3), description text
    - status enum: `pending, completed, failed, expired, refunded`
    - card_brand text null, card_last4 char(4) null, failure_reason text null
    - expires_at timestamptz, completed_at timestamptz null
    - created_at, updated_at
  - `payment.outbox_events` ve `payment.processed_events`: staff'taki desenin
    kopyası (`services/staff-service/internal/repository/outbox_repository.go`,
    `outbox_store.go`, retention store).
  - `sqlc.yaml`'a rename bloğunu ekle (skills.md §1).
  - **Commit:** `feat(payment): add the payments table` (outbox tabloları 3.1'de)

- [x] **3.3 Initiate kalıcı olsun** — `c0e56df`, `76e43a9`
  - `/internal/payments/initiate`:
    - `pending` kayıt oluştur. reference_id zaten varsa mevcut kaydı döndür
      (idempotent).
    - `expires_at = clock.Now() + RESERVATION_TIMEOUT_MINUTES`.
    - Goroutine ve doğrudan publish silinsin; "MOCK" log ve yorumları kalksın.
  - `/internal/payments/refund`: kaydı `refunded` yap.
  - Süresi dolan `pending` kayıtları `expired` yapan küçük bir worker. meal
    kendi rezervasyonunu zaten düşürüyor; burada event gerekmez.
  - `new-backend/shared/contracts/payment.go`: `PaymentURL` alanını kaldır.
  - **Commit:** `feat(payment): persist initiated payments`
  > Not (24.09): `PaymentURL` alanı `contracts`'tan 3.5'te kaldırılıyor —
  > meal onu okuduğu için burada silmek meal'in derlemesini bozardı; payment
  > alanı artık doldurmuyor. İade: meal tek öğünü iptal ettiğinde toplu
  > ödemenin yalnız bir kısmı döner; bunun için `payments`'a (henüz hiçbir
  > yerde uygulanmamış 00002'ye) `refunded_amount` eklendi, ödeme toplam
  > iade ödenen tutara ulaşınca `refunded` olur. Bulunamayan referans 404,
  > tamamlanmamış ödeme / fazla iade 409.

- [x] **3.4 Kart onayı ucu** — `afba2c1`
  - `services/payment-service/internal/module.go` `RegisterRoutes` (şu an
    boş, ≈46). Zincir: `JWTAuth`, `CSRFProtection`, `UserRateLimit`,
    `RequireStudent`.
    - `POST /api/payments/:payment_id/confirm` + `Idempotency()`; gövde
      `{card_number, exp_month, exp_year, cvc, cardholder_name}`.
    - `GET /api/payments/:payment_id`: ödemenin durumu (yalnız sahibine).
  - Kurallar:
    - Ödeme çağıranın (`student_id`) değilse 404.
    - `pending` değilse 409.
    - Süresi dolmuşsa `expired` işaretle ve 409 dön.
    - Doğrulama hatası 422.
    - Test kartına göre `completed` veya `failed` + outbox; payload meal'in
      beklediği biçimde.
  - Caddy `/api/payments` yolunu zaten payment-service'e yönlendiriyor
    (`frontend/Caddyfile`); değişiklik gerekmez.
  - Birim testleri: Luhn, test kartları, sahiplik, durum geçişleri; kart
    numarasının loga düşmediğini doğrula (logger'ı yakalayan test).
  - **Commit:** `feat(payment): confirm payments with test cards`
  > Not (24.09): Reddedilen kart bir istek hatası değil, 200 + `status:
  > "failed"` ve `failure_reason` döner. Doğrulama `internal/card`'da; YY veya
  > YYYY yıl kabul edilir, kart son kullanma ayının son gününe kadar
  > geçerlidir. Bilinmeyen marka (`6011...`) `card_brand = NULL` olur.
  > Idempotency kaydı yalnız istek gövdesinin SHA-256 özetini ve yanıtı
  > (son 4 hane) tutar. Event sabitleri `shared/events`'e eklendi.

- [x] **3.5 meal tarafı** — `bf64c97`
  - `services/meal-service/internal/dto/reservation_dto.go:41`, `:51`:
    `PaymentURL` → `PaymentID` (`json:"payment_id"`).
  - `reservation_service.go` `CreateReservation` ve `CreateBatchReservation`
    yanıtlarını güncelle.
  - Testleri güncelle.
  - **Commit:** `feat(meal): return the payment id for card checkout`
  > Not (24.09): `contracts.InitiatePaymentResponse.PaymentURL` burada
  > kaldırıldı (3.3 notu). Hedef akışın 4. adımı için `payment.failed`
  > consumer'ı rezervasyonu artık `expired` değil `cancelled` yapıyor
  > (`expired` süresi dolan ödemelere kaldı; event payload'ı değişmedi).
  > İptalde iade, rezervasyonun ödendiği referansla (`res_<id>` veya toplu
  > ise `bat_<batch>`) isteniyor — eskiden çıplak rezervasyon id'si
  > gidiyordu ve toplu rezervasyonun ödemesi bulunamazdı.

- [x] **3.6 Web** — `dac05ee`
  - `frontend/src/lib/api-client.ts`: `paymentApi` (`/api/payments`).
  - Yeni `frontend/src/lib/services/payment-service.ts` ve tipleri
    (`lib/types.ts`). `meal-service.ts` tiplerinde `payment_url` →
    `payment_id`.
  - `pages/student/cafeteria/index.tsx`, diyalog akışı:
    1. özet,
    2. kart formu (numara 4'lü gruplar, AA/YY, CVC, kart üzerindeki ad),
    3. onay,
    4. rezervasyon durumu yoklanır (confirmed/cancelled),
    5. sonuç.

    "Ödeme Sayfasına Git" düğmesi ve `window.open(paymentUrl)` silinsin. Test
    kartları kutusu eklensin.
  - Vitest: form doğrulaması (Luhn, son kullanma tarihi).
  - **Commit:** `feat(frontend): add card checkout for meal reservations`
  > Not (24.09): `react-hook-form`/`zod` projede kurulu olmadığı için (yeni
  > kütüphane onay ister) form `useState` + `lib/payment-card.ts`'teki saf
  > doğrulamayla yazıldı. Akış `pages/student/cafeteria/checkout-dialog.tsx`
  > bileşeninde; diyalog kapatılıp açılınca aynı ödemeye devam eder, kart
  > bilgisi gönderildiği anda state'ten silinir. Reddedilen kartta sonuç,
  > rezervasyon `cancelled` görülünce (en geç 30 sn) gösterilir. Sayfadaki
  > ekrana düşen `\n` metni de kaldırıldı. Vitest: kart doğrulaması,
  > servisler ve diyaloğun başarılı/reddedilen/geçersiz kart akışları;
  > Chromium'da API taklit edilerek masaüstü ve mobil genişlikte denendi.

- [x] **3.7 Mobil** — `a829f2a`, `a57f684`
  - Yeni `mobile/services/paymentService.ts` ve testi.
  - `mobile/types/meal.types.ts`.
  - `mobile/hooks/useMeals.ts` (≈41'deki mock yorumunu güncelle).
  - `mobile/app/cafeteria.tsx`: toplu rezervasyondan sonra kart formu (modal
    veya ayrı ekran).
  - **Commit:** `feat(mobile): add card checkout for meal reservations`
  > Not (24.09): Randevu sihirbazına 5. adım (Kart) eklendi; akış
  > `components/CardCheckout.tsx`'te, doğrulama `lib/payment-card.ts`'te (web
  > ile aynı kurallar). Rezerve edilmiş ama ödenmemiş toplu randevu sihirbaz
  > kapatılınca silinmez, yeniden açılınca aynı ödemeden devam edilir. Eski
  > "2,5 sn sonra listeyi yenile" mock kalıntısı kaldırıldı; sonuç yoklanırken
  > randevu listesi de güncelleniyor. Temiz checkout'ta `tsc` TS2882 verdiği
  > (gitignore'daki `expo-env.d.ts`) ve bu faz mobil CI job'ını ilk kez
  > tetikleyeceği için `nativewind-env.d.ts`'e `expo/types` referansı ayrı
  > commit'le eklendi. Cihazda manuel test bu ortamda yapılamadı; jest +
  > `tsc` temiz.

- [x] **3.8 e2e** — `2375165`
  - CI golden path:
    - öğrenci rezervasyon → 4242 ile confirm → rezervasyon `confirmed`
    - 4000 0000 0000 0002 → rezervasyon `cancelled`
  - **Commit:** `test(infra): cover card checkout in the e2e job`
  > Not (24.09): Golden path'e eklendi: gelecek haftanın Perşembe ve Cuma
  > öğlesi (seed Pzt–Çar'ı dolduruyor) toplu rezerve edilir; 4242 →
  > `confirmed`, 0002 → `cancelled` (30 sn yoklama); sonuçlanmış ödemeyi
  > tekrar onaylamak 409, admin'in ödemeyi okuması 403. İptal/iade adımı
  > eklenmedi: iptal kilidi önceki cuma 23:59'a bağlı, hafta sonu koşan CI
  > kırmızıya düşerdi. Docker bu ortamda çalışmadığı için doğrulama push
  > sonrası CI'da.

## Faz sonu
- README §5 yeşil olmalı.
- README §1 madde 6.

> Not (24.09): §5 sonuçları: backend 11 modül vet temiz, 1111 test geçiyor
> (faz başı aynı sayımla 1033; +72 payment, +6 meal); golangci-lint v2.13.2
> ve gosec v2.29.0 değişen 3 modülde (shared, meal, payment) 0 bulgu,
> go.mod'lar tidy. Frontend typecheck + lint + Prettier + build temiz, 112
> test. Mobil `tsc` temiz (temiz checkout dahil, `a829f2a`), 101 test.
> Faz dışı ekler: `a829f2a` (mobil CI job'ı bu fazda ilk kez koşacağı için),
> `fbb8107` (kök README'de ödeme artık mock değil).
> e2e push sonrası CI'da doğrulanacak; Faz 2'nin PR çalıştırmasındaki e2e
> hatası koddan değil, runner'ın imaj derlerken kapatılmasından
> ("received a shutdown signal"). Mevcut volume'u olan kurulumda
> `payment` veritabanı elle eklenmeli (migrate logu komutu yazar;
> DEPLOY.md sorun giderme).
