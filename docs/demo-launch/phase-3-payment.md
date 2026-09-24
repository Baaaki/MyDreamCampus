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

- [x] **3.1 payment veritabanı altyapısı**
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

- [x] **3.2 Şema**
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

- [x] **3.3 Initiate kalıcı olsun**
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

- [ ] **3.4 Kart onayı ucu**
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

- [ ] **3.5 meal tarafı**
  - `services/meal-service/internal/dto/reservation_dto.go:41`, `:51`:
    `PaymentURL` → `PaymentID` (`json:"payment_id"`).
  - `reservation_service.go` `CreateReservation` ve `CreateBatchReservation`
    yanıtlarını güncelle.
  - Testleri güncelle.
  - **Commit:** `feat(meal): return the payment id for card checkout`

- [ ] **3.6 Web**
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

- [ ] **3.7 Mobil**
  - Yeni `mobile/services/paymentService.ts` ve testi.
  - `mobile/types/meal.types.ts`.
  - `mobile/hooks/useMeals.ts` (≈41'deki mock yorumunu güncelle).
  - `mobile/app/cafeteria.tsx`: toplu rezervasyondan sonra kart formu (modal
    veya ayrı ekran).
  - **Commit:** `feat(mobile): add card checkout for meal reservations`

- [ ] **3.8 e2e**
  - CI golden path:
    - öğrenci rezervasyon → 4242 ile confirm → rezervasyon `confirmed`
    - 4000 0000 0000 0002 → rezervasyon `cancelled`
  - **Commit:** `test(infra): cover card checkout in the e2e job`

## Faz sonu
- README §5 yeşil olmalı.
- README §1 madde 6.
