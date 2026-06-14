# Senior Code Review — MyDreamCampus

Üç paralel inceleme + `PROD_READINESS.md` ile çapraz doğrulama yaptım. PROD_READINESS dokümanın çoğu Tier 1 öğesini bitirmiş; aşağıda **henüz örtülmemiş veya yarı kalmış** sorunlara odaklandım.

---

## 1. Genel Değerlendirme (büyük resim)

**Güçlü yanlar:**
- Mimari kararlar tutarlı ve gerekçelendirilmiş (sqlc + pgx, Gin, Traefik, RabbitMQ, Outbox). CLAUDE.md "tartışılmaz" dediği için scope creep yok.
- `backend/shared/` paketleri (auth, ratelimit, csrf, audit, clock, errors) gerçek bir "platform katmanı" izlenimi veriyor — interview'da bu dosyaları açtırırsan iyi puan.
- PROD_READINESS.md'nin kendisi bir CV silahı: "scope dışı" listesi olgun developer sinyali.
- Test alt yapısı (CI'da postgres+redis+rabbitmq spin-up) var, integration test koşuluyor.

**Zayıf yanlar:**
- 9 mikroservis × ayrı DB = portfolyo için **abartılı bölünme**. "Dağıtık monolit" tuzağına yakın.
- Cross-service tutarsızlıklar (timeout, outbox enum, error import stili) — tek geliştirici varken bile pattern drift olmuş, bu kötü sinyal.
- Frontend test/i18n/form katmanı juniör seviye.
- Observability iddia ediyor (Loki/Grafana) ama `infrastructure/loki/`, `promtail/` boş `.gitkeep` veya unimplemented.

**Tek cümle hüküm:** Bireysel dosyalarda senior, sistem genelinde "henüz tutarlılığa zaman ayıramamış senior" izlenimi var.

---

## 2. Kritik — Gelecekte Patlama Potansiyeli Yüksek

### 2.1 Outbox şema enum tutarsızlığı (cross-service)
- staff: `('pending','processed','failed')`
- meal/student: `('pending','published','failed')`

Tek geliştirici bile aynı pattern'i iki farklı isimle yazmış. Yeni servis eklerken hangisini kopyalayacaksın? **Senior gözüyle red flag.** `backend/shared/sql/migrations/_outbox.sql` snippet veya goose'ye shared template yaz, her servis include etsin. Aksi halde 6 ay sonra "neden staff event'leri publisher'da takılı?" debug oturumun olur.

### 2.2 RabbitMQ DLQ kullanılmıyor
`shared/rabbitmq/consumer.go`'da `ConsumeWithDLQ` mevcut ama hiçbir servis çağırmıyor. Poison pill geldiğinde sonsuz requeue → consumer thrash. Bu **bilinen bir kod yolu**, test etmek 10dk: kasten panic atan handler ile event gönder, queue'da requeue sayısını izle.

### 2.3 Refresh token rotation yok (PROD_READINESS Tier 2.8)
24 saat sliding refresh + rotation yok = bir kez sızdırılan token tüm session ailesini açar. README'de "JWT + refresh" diye geçiyor; mülakatçı "rotation var mı?" diye soracak. Cevap "yapılmadı, scope" olabilir ama dokümanda bu kararı **README'de görünür** yap, içeride gizli kalmasın.

### 2.4 Audit log application user'la yazılıyor
`audit_log` tablosuna servis postgres user'ı INSERT da, UPDATE de yapabiliyor. PROD_READINESS Tier 2.12 atlanmış. Compliance argümanın çürür: "saldırgan iz silebilir mi?" → "evet, çünkü aynı user." Tek migration + ayrı user. Yarım yapma riskli (denied error) ama tamamı 1 saatlik iş.

### 2.5 PII redaction yok (audit + log)
`shared/audit/security.go`'da `Email`, `IP`, `UserAgent` ham yazılıyor. Portfolyoda GDPR şart değil ama **CV'de** "GDPR-aware logging" diyebilmek için 30dk lik `MaskEmail()` helper'ı eklemek skor yapar. Bedava kazanç.

### 2.6 `.env.example` içinde gerçek default secret
`ADMIN_INITIAL_PASSWORD=Admin123!`, `REDIS_PASSWORD=changeme_redis_secret` gibi değerlerin .env.example'da gerçek-vari görünmesi yanlış mesaj veriyor. PROD validation'da panic atıyor olsan bile (ki helpers.go'da var), placeholder olduğu **isminden** belli olsun: `CHANGE_ME`, `<your-password>`. `.env.example` kopyala-çalıştır akışında biri prod'a basacak gün geliyor.

---

## 3. Önemli — Yapı Yorgunlaştığında Kıracak

### 3.1 Servis sayısı vs. veri büyüklüğü
9 servis × 9 Postgres instance, hepsi tek üniversite varsayımıyla. Bu **paper microservices**. Üç senaryo:
- **Portfolyo açıklaması yeterli:** README'de "her servisin DB izolasyonunu göstermek için ayrı instance, prod'da aynı cluster'da schema-level ayrım önerilir" diye **neden böyle yaptığını yaz**. Aksi halde reviewer "junior microservices tuzağı" der.
- Pratik: lokal `docker compose up` 8 postgres ayağa kaldırıyor — yeni dev için 4-5 dakika ve 4GB RAM. README "first-time setup" notu ekle.

### 3.2 Inter-service auth zayıf nokta
`X-Internal-Secret` header tek tek servislerde mi kontrol ediliyor, yoksa Traefik'te mi? Infra agent Traefik'te validation yok diyor. İki uçtan birinde kalıcı bir test yaz (`TestInternalSecretEnforced`) — iyi niyet bug'ları yakar.

### 3.3 `defer tx.Rollback` pattern'i
Backend agent "double rollback" dedi ama pgx v5'te `Rollback` committed tx üzerinde `ErrTxClosed` döner ve idempotent sayılır — yani **aslında yanlış değil**. Ama loglara `"rollback on committed tx"` warning düşürmemek için pattern temizliği yapılabilir. Bunu Tier 3'e at, bug değil tat meselesi.

### 3.4 Pagination limit clamping
Handler'larda `maxPageLimit=100` constant'ı var ama enforcement bazılarında eksik. Adversarial input: `?limit=1000000` → seq scan. 5 satırlık helper'a çek (`ClampLimit(req.Limit, 100)`), tüm list endpoint'lerinde kullan.

### 3.5 Frontend type-gen pre-commit yok
`bun run gen:api-types` script'i var ama kimse koşmuyor → backend OpenAPI değişince frontend tip drift'i. Husky + lint-staged ile pre-commit hook 15 dakikalık iş. CI'da `git diff --exit-code` ile de kapatabilirsin (generated dosyalar commit'lenmemişse fail).

### 3.6 Frontend form validation manuel
`react-hook-form` + `zod` yok. 3-5 ekrandan sonra DRY'a vuracak. Şimdi 1 saat eklemek, 6 ay sonra refactor'dan kurtarır. Mobile için aynı argüman.

### 3.7 i18n hiç yok
"Yükleniyor..." string'i her yerde hardcode. Türkçe kalacaksa README'de "TR-only by design" yaz, scope kararı olarak göster — agent gibi "eksiklik" değil. Üniversite uygulaması zaten lokal pazar ürünü, savunulabilir.

### 3.8 Mobile offline / network resilience yok
`@react-native-community/netinfo` yok, axios retry yok. Kampüs WiFi'da 30sn dropouts olağan; ilk pratik kullanım deneyimi kötü olur. `axios-retry` + offline banner = yarım gün, gerçek kullanım yolu açar.

### 3.9 Graceful shutdown sadece auth-service'te
PROD_READINESS Tier 2.9 yarı uygulanmış. Diğer 8 servis SIGTERM'de in-flight RabbitMQ mesajını kaybedebilir. Kopyala-yapıştır iş. Bunu yapmamak "dağıtık sistem ciddi mi?" sorusunu çağırır.

---

## 4. Mimari Yorumlar (senior bakış)

### 4.1 Microservices justification
9 servis, tek ürün, tek geliştirici, tek üniversite. "Neden microservices?" sorusunun savunması zayıf. Üç tutum mümkün:
1. **README'de açıkça**: "Bu proje **mikroservisleri öğrenmek için** monolit yerine bölünmüştür. Prod kullanım için modular monolith (tek binary, modül ayrımı) daha uygundur."
2. Servis sayısını azalt (auth + staff + student → identity, course-catalog + enrollment + grades → academics, attendance + meal → operations). 9 → 3-4 servis daha gerçekçi.
3. Olduğu gibi bırak, "show, don't tell" portfolio gibi sun.

(1) en hızlı ve dürüst, (2) gerçek değer, (3) status quo.

### 4.2 Payment service mock-only
Memory'de "payment is mock-only" diyor — README'de bu **çok belirgin** olmalı. Reviewer "ödeme akışı nasıl?" diye baktığında "mock olduğunu öğrendi → güveni sarsıldı"yı önle. Payment'i PaymentIntent state machine'iyle (`pending → authorized → captured → refunded`) **mock'lasan bile** durumları gösteren bir tasarım, no-mock çalışan basit kod'dan iyi sinyal verir.

### 4.3 Notification servisi yok
Memory'de bilinçli scope dışı. README'de aynı şekilde — kullanıcıya "neden meal closed gününde mail gelmiyor?" sorusu çıkmasın diye. Event'lerin tüketicisi olmadığı yerlere "notification consumer would attach here" yorumu **kod içinde** dursun (kabul edilebilir bir comment, çünkü mimari decision).

### 4.4 Tek kullanıcı multi-role yok
PROD_READINESS bunu açıkça scope dışı tutmuş, iyi. Ama gerçek üniversitede asistan = student + teaching staff. Kararın **README'de görünür kalması** kritik.

---

## 5. Portfolyo Açısından (CV/iş bulma değerlendirmesi)

**Reviewer ne arar (90 saniyede):**
1. README'yi tarar — "ben ne öğreniyorum?" → 3 dakikada anlayabiliyor mu?
2. Bir servisin `cmd/main.go` + bir handler + bir migration açar — o servisin tutarlılık seviyesini ölçer.
3. CI yml'sini açar — gerçek mi tiyatro mu?
4. Yapı seçimlerinin **dokümante edilmiş gerekçesi** var mı?

**Senin proje:**
- (1): README üst seviyede iyi, ama `ROADMAP.md` + `PROD_READINESS.md` + `TEST_PLAN.md` + `TEST_STATUS.md` (40KB!) **fazla** doküman. Reviewer kaybolur. Hepsini `docs/` altına taşı, README'den 1-2 link ile referansla.
- (2): Auth-service iyi durumda. Ama meal-service vs staff-service vs student-service arası tutarsızlıklar (timeout, outbox enum, error import) ilk açılan **ikinci servis** karar verici. Şanssız bir reviewer staff-service'i ilk açar ve "5sn timeout, neden?" der.
- (3): CI ciddi. govulncheck + gosec + integration tests + path filter = artı puan. Coverage gate sadece shared'de %40 — services için de eklenmeli, gerçek "we test" iddiası için.
- (4): PROD_READINESS bu işi yapıyor. Ama bu **doküman kullanıcı için** değil, **AI için** yazılmış gibi (madde madde, durum işaretleri). Reviewer "dev'in to-do listesini okuyorum" hissine girer. **README'de** olgunluk seviyesini öz olarak sun (1 paragraf), `PROD_READINESS.md`'i `docs/internal/`'a koy.

**Ekleyince anlamlı puanlar (1 günden az):**
- README başına ekran kaydı GIF'i (login → ders kayıt → menü → çıkış). Görsel kanıt = en hızlı güven.
- "Architecture Decision Records" — `docs/adr/` altında 5-6 markdown: ADR-001 sqlc neden seçildi, ADR-002 her servise ayrı DB neden, ADR-003 RabbitMQ neden. **Junior'lardan ayıran detay budur.**
- Frontend → Vercel/Cloudflare Pages canlı demo. Backend mock veya read-only. Tek tıkla denenebilirlik = etkileyici.
- 1 sayfa "metrics" — kaç servis, kaç endpoint, test coverage, build süresi. Eylem değil özet.

---

## 6. Önerilen Sıralama (en yüksek ROI)

| Öncelik | Madde | Süre | Neden |
|---|---|---|---|
| 1 | Outbox enum/index'i shared template'e çıkar | 1-2 saat | Cross-service tutarlılık, en görünür design defekti |
| 2 | RabbitMQ DLQ'yu en az 1 consumer'da aktive et | 1 saat | Mevcut kodun kullanılmamış hali var, "yapılmamış" değil "bağlanmamış" |
| 3 | Graceful shutdown'u 8 servise yay | 2-3 saat | Tek servis yapıp diğerlerini bırakmak tutarsızlık sinyali |
| 4 | README sadeleşmesi + mimari ADR'ler | 3-4 saat | Reviewer ilk dakikalarındaki algı |
| 5 | Frontend `react-hook-form + zod` + i18n karari (yap veya scope-out yaz) | 0.5-1 gün | "Tek dil, by design" denirse 5dk, yapılırsa yarım gün |
| 6 | Refresh token rotation (Tier 2.8) | 3-4 saat | Auth iddiasına en yakışan tek özellik |
| 7 | Audit append-only role | 1-2 saat | "Compliance düşündüm" sinyali, ucuz |
| 8 | Mobile offline detection + axios-retry | 2-3 saat | Gerçek kullanım yolunu açar |

Toplam ~3 disiplinli gün. Bu listeden 4-5 madde proje algısını "ödevi gibi duruyor" → "junior+ ürünü" eşiğinden geçirir.

---

## 7. Yapma Listesi (zaman kuyusu)

- Bütün servislere distributed tracing (OTel) eklemek — tek başına yarım hafta, ekosistem hazır değil.
- Servisleri Kubernetes'e taşımak — Docker Compose şu hedef için yeterli.
- Tüm testleri mock'tan integration'a çevirmek — fast feedback'i öldürür.
- Yeni özellik eklemek (ROADMAP'tekiler) — **mevcut tutarsızlıkları kapatmadan** yeni feature **negatif sinyal**.

---

## Tek Cümle Özet

Mimari iddian olgun, uygulamanda tutarlılık borcu var; **3 günlük tutarlılık + dokümantasyon turu** seni "öğrenci projesi" çağrışımından "üretime götürebilirim" çağrışımına geçirir — Tier 2'nin kalan iki maddesini (refresh rotation, object-level authz) bonus olarak alma.
