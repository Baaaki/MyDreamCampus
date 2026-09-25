# Faz 7 — Kalıcı durum ve gece 04:00 geri dönüşü

## Başlamadan
- Oku: `new-backend/skills.md`, `DEPLOY.md` (compose yapısı).
- README §4'teki **K3** (düzenleme sırasında yazma kilidi) ve **K4**
  (Cloudflare Access) cevaplanmış olmalı.
- Faz 4 (seed) ve Faz 6 (süper admin işareti) bitmiş olmalı.

## Bağlam
- Kullanıcı kuralı: herkes değişiklik yapabilir ama **yalnız süper adminin
  değişiklikleri kalıcıdır.** Her gece 04:00'te (Europe/Istanbul) sistem
  süper adminin en son kaydettiği hale döner.
- Gün içinde herkesin değişiklikleri birbirine karışıyor; süper adminin
  yaptıklarını kayıt kayıt ayıklamak güvenilir olmaz. Bu yüzden süper admin
  ayrı bir **düzenleme modunda** çalışır ve sonucu bir anlık görüntü (kalıcı
  durum) olarak kaydeder.

## Süper adminin akışı ("Kalıcı Veri" sayfası)
1. **Düzenlemeye başla:** Sistem son kalıcı duruma döner (günün ziyaretçi
   değişiklikleri atılır). K3 evetse süper admin dışındaki herkese yazma
   kapanır; ziyaretçiler okuyabilir ve giriş yapabilir.
2. Süper admin normal arayüzden düzenler.
3. Bitirmek için iki seçenek:
   - **Kaydet ve yayına al:** Event'ler boşalana kadar beklenir, tüm DB'lerin
     anlık görüntüsü alınır, bu yeni kalıcı durum olur ve kilit kalkar.
   - **Vazgeç:** Kaydetmeden kalıcı duruma dönülür ve kilit kalkar.

   Düzenleme modu 2 saat içinde kaydedilmezse otomatik olarak "Vazgeç"
   uygulanır; kaydetmek unutulsa bile sistem kilitli kalmaz.
4. **Her gece 04:00:** Kalıcı duruma dönülür. O sırada düzenleme modu açıksa
   geri dönüş atlanır ve loglanır.

**Ek özellikler:**
- **"Bugünkü değişiklikleri şimdi geri al":** biri uygunsuz bir şey yazarsa
  gece 04:00'ü beklemeye gerek kalmaz.
- **Sürüm geçmişi:** son 7 kalıcı durum tutulur; her biri için "Bu sürüme
  dön".
- **İlk kalıcı durum:** ilk kurulumda seed bittikten sonra otomatik alınır.

## Mimari
- Web servisleri Postgres süper kullanıcı şifresini **görmez**. Docker soketi
  **hiçbir** konteynere verilmez.
- Akış: panel → catalog API → Redis komut kuyruğu → `demo-ops` konteyneri işi
  yapar → durumu Redis'e yazar → panel durumu yoklar.

### Redis sözleşmesi (DB 0)
| Anahtar | Tür | İçerik |
|---|---|---|
| `ops:commands` | list (LPUSH / BRPOP) | `{"id","action","version?","requested_by","requested_at"}`; `action` ∈ `begin_edit`, `save`, `cancel_edit`, `restore_now`, `restore_version` |
| `ops:status` | string (JSON) | `{"mode":"normal\|editing\|busy","current":"<sürüm>","versions":[...],"edit_deadline","last_action","last_error","updated_at"}` |
| `ops:write_lock` | string `"1"`, TTL | Yazma kilidi (TTL = düzenleme süre sınırı) |

Geri dönüşte `FLUSHDB` yapılır; `demo-ops` ardından `ops:status`'u ve
gerekiyorsa kilidi yeniden yazar.

## Görevler

- [x] **7.1 demo-ops konteyneri** (gemini ile yapıldı)
  - Yeni dizin `new-backend/infrastructure/demo-ops/`:
    - `Dockerfile`: `FROM postgres:18-alpine`. `pg_dump` sürümü sunucuyla aynı
      olmalı; migrate imajındaki alpine istemcisi 16 sürümünde, bu yüzden o
      imaj kullanılamaz.
      - goose'u `migrate/Dockerfile`'daki build aşamasıyla ekle.
      - Paketler: `redis`, `curl`, `jq`, `tzdata`.
      - Migration dosyalarını `migrate/Dockerfile`'daki COPY satırlarıyla
        kopyala.
    - Scriptler: `ops.sh` (ana döngü: `BRPOP` 30 sn zaman aşımı + saat
      kontrolü), `lib.sh`, `snapshot.sh`, `restore.sh`.
  - `docker-compose.yml`'e `demo-ops` servisi:
    - `restart: unless-stopped`, host portu yok, `mem_limit`.
    - `depends_on`: `seed: service_completed_successfully` ve
      postgres/redis/rabbitmq `service_healthy`.
    - env: `POSTGRES_USER`/`POSTGRES_PASSWORD`, `SERVICE_DB_PASSWORD`,
      `REDIS_PASSWORD`, `RABBITMQ_USER`/`RABBITMQ_PASSWORD`,
      `TZ=Europe/Istanbul`, `NIGHTLY_RESET_AT=04:00`, `BASELINE_KEEP=7`,
      `EDIT_TIMEOUT_MINUTES=120`, `DEMO_MODE`.
    - volume `demo-baselines:/baselines`.
  - `DEMO_MODE` kapalıysa konteyner hiçbir şey yapmadan beklesin (ya da
    compose profile `demo` kullan; hangisini seçtiğini not et).
  - **Commit:** `feat(infra): add the demo-ops container`

- [x] **7.2 Anlık görüntü (save)** (gemini ile yapıldı)
  - Veritabanları: `auth staff student catalog enrollment attendance grades meal payment notification`.
  - Boşalmayı bekle (en fazla 60 sn):
    - her DB'de outbox tablosunda `pending` satır sayısı 0 (tablo adlarını her
      serviste doğrula; genelde `<schema>.outbox_events`);
    - RabbitMQ management API'de (`http://rabbitmq:15672/api/queues`) tüm
      kuyruklarda `messages` 0.

    Zaman aşımında `last_error` yaz; kalıcı durum değişmez.
  - Her DB için `pg_dump -Fc` → `/baselines/<YYYYMMDD-HHMMSS>/<db>.dump`, ayrıca
    `manifest.json` (tarih, her DB'nin goose versiyonu, isteyen kullanıcı).
  - Hepsi başarılıysa `current` işaretçisini güncelle; en eski sürümleri
    `BASELINE_KEEP`'e göre sil.
  - İsteğe bağlı sunucu dışı kopya: `BASELINE_OFFSITE_CMD` doluysa her
    kayıttan sonra çalıştır (örn. rclone); boşsa atla.
  - **Commit:** `feat(infra): snapshot the permanent state`

- [ ] **7.3 Geri dönüş (restore)**
  - Sıra:
    1. Yazma kilidini koy.
    2. Her DB için `pg_restore --clean --if-exists --single-transaction`
       (süper kullanıcıyla). Sahipliğin `*_svc` rollerinde kaldığını doğrula.
       `lock_timeout` ayarla ve hata olursa bir kez yeniden dene.
    3. Her DB için goose up (`migrate/entrypoint.sh`'teki mantık, `*_svc`
       rolüyle). Arada deploy edilmiş yeni migration'lar böylece uygulanır.
    4. RabbitMQ'daki tüm kuyrukları purge et (management API).
    5. Redis `FLUSHDB`: zaman makinesi, token kara listesi, rate limit,
       idempotency ve yoklama tamponu sıfırlanır.
    6. `ops:status`'u yeniden yaz.
    7. Kilidi kaldır. Geri dönüş `begin_edit` için yapıldıysa kilit kalır.
  - Servisler çalışırken yapılır; birkaç saniyelik hata logları kabul
    edilebilir. Oturumlar da geri döndüğü için herkes oturumdan düşer.
  - **Kabul** (kullanıcıya sudo komutlarıyla yaptır):
    - demo admin ile bir kayıt ekle → "şimdi geri al" → kayıt yok;
    - süper admin düzenle, kaydet → "şimdi geri al" → süper adminin kaydı
      duruyor.
  - **Commit:** `feat(infra): restore the permanent state`

- [ ] **7.4 Zamanlama ve ilk kalıcı durum**
  - Açılışta `/baselines/current` yoksa seed bitmiş demektir; hemen ilk
    kalıcı durumu al.
  - Döngü her dakika kontrol eder:
    - saat `NIGHTLY_RESET_AT` ve bugün henüz yapılmadıysa: mod `editing`
      değilse restore et, `editing` ise atla ve logla;
    - düzenleme süresi dolduysa `cancel_edit` uygula.
  - **Commit:** `feat(infra): schedule the nightly restore`

- [ ] **7.5 Yazma kilidi middleware'i (K3)**
  - Yeni `shared/platform/middleware/writelock.go`. Global zincire ekle
    (`shared/httpserver/server.go` `NewServer`, rate limit'ten sonra).
  - Kilit koşulu, hepsi birlikte:
    - istek POST, PUT, PATCH veya DELETE;
    - `ops:write_lock` mevcut (Redis GET, süreç içinde 1 sn önbellek);
    - istek süper admin token'ı taşımıyor (imzayı `requestIdentity`'deki gibi
      doğrula).

    Bu durumda 503:
    `{"error":"Sistem şu an yönetici tarafından güncelleniyor. Birkaç dakika sonra tekrar deneyin.","code":"SYSTEM_EDITING"}`.
  - Muaf yollar: `/internal/*`, `/api/auth/login`, `/api/auth/refresh`,
    `/api/auth/logout`, `/health`, `/ready`. Redis hatasında fail-open.
  - Kilit açıkken yanıtlara `X-System-Editing: 1` ekle (CORS expose).
  - Testler.
  - **Commit:** `feat(shared): lock public writes while the super admin edits`

- [ ] **7.6 Süper admin API'si**
  - catalog'da `/api/catalog/admin/ops` grubu:
    `JWTAuth(WithFailClosed())` + `RequireSuperAdmin()`.
    - `GET /status`
    - `POST /begin-edit`, `/save`, `/cancel-edit`, `/restore-now`
    - `POST /restore/:version`
  - Her istek komutu `ops:commands`'a yazar ve 202 döner. İşlemleri audit
    log'a yaz.
  - K4 evetse bu yol Cloudflare Access ile korunacak (Faz 8'de kullanıcı
    ayarlar). Uygulama tarafında ek iş yok, ama yol adını değiştirme.
  - **Commit:** `feat(catalog): add super admin endpoints for the permanent state`

- [ ] **7.7 "Kalıcı Veri" sayfası**
  - `frontend/src/pages/admin/system/baseline/index.tsx` ve `routes.tsx`'de
    route (≈154 civarı). Menüde yalnız `user.is_superadmin` iken görünsün;
    route guard da olsun.
  - Gösterilecekler: mod, kalıcı durumun tarihi, düzenlemenin kalan süresi,
    son hata, sürüm listesi. Düğmeler onay diyaloğu ile çalışsın. Mod `busy`
    iken durumu 2 sn'de bir yokla.
  - Genel şerit: `X-System-Editing` başlığı gelince "Sistem güncelleniyor, şu
    an değişiklik yapılamaz."
  - **Commit:** `feat(frontend): add the permanent state page for the super admin`

- [ ] **7.8 e2e (önerilir)**
  - CI `backend-e2e`'ye ekle:
    1. demo-ops ayakta;
    2. begin-edit;
    3. süper admin bir kayıt ekler;
    4. save;
    5. demo admin başka bir kayıt ekler;
    6. restore-now;
    7. kontrol: demo adminin kaydı yok, süper adminin kaydı var.
  - **Commit:** `test(infra): cover the permanent state cycle in e2e`

## Faz sonu
- README §5 yeşil olmalı.
- README §1 madde 6.
