# Mikroservis Migrasyonu — Başlangıç Dosyası

> **MİGRASYON TAMAMLANDI (2026-08-10).** Faz dosyaları tarihsel referanstır;
> aşağıdaki çalışma protokolü artık işletilmiyor.
>
> **GÜNCEL MİMARİ: [`01-REFERANS-MIMARI.md`](01-REFERANS-MIMARI.md)** (bu
> klasörde) — projenin mimari kaynağı odur, faz dosyaları değil. Yanında
> geçerliliğini koruyan dört doküman daha var: `02-GUVENLIK.md`,
> `03-IZLENEBILIRLIK.md`, `04-PROD-HAZIRLIK.md`, `05-DAYANIKLILIK.md`.

---

## Çalışma Protokolü

### Oturuma başlarken: "nerede kaldık?"

```bash
ls microservices-migration/          # -TAMAMLANDI.md ekli olanlar bitmiş
git log --oneline -15                # gerçek ilerleme kaydı
git status                           # yarım kalmış değişiklik var mı
```

**Çelişki olursa `git log` doğrudur.** Durum tablosu ve dosya adları elle
güncelleniyor; oturum faz ortasında kesilirse güncellenmemiş olabilirler.
`git status` kirliyse önceki oturum faz ortasında kesilmiş demektir — o fazın
dosyasını aç ve kaldığın yerden devam et, baştan başlama.

### Faz akışı

1. Durum tablosundan sıradaki fazı bul (ilk `[ ]` olan satır).
2. O fazın dosyasını oku, **sadece onu** uygula.
3. **Faz İÇİNDE ilerledikçe:**
   - Her anlamlı birim bitince **commit at** — faz sonunu bekleme.
     Faz 4 dokuz servis çıkarıyor: her servis kendi commit'i.
     Faz 2/3 çok adımlı: mantıksal grup başına commit.
   - Faz dosyasında alt kontrol listesi varsa (Faz 4 §D — 9 servislik tablo)
     **her satır bitince işaretle**. Bu, oturum kesilirse tek kurtarma kaydın.
4. Faz bitince:
   - Fazın "Bitiş Kriteri" bölümündeki doğrulama komutlarını çalıştır.
   - Dosyayı yeniden adlandır: `faz-N-xxx.md` → `faz-N-xxx-TAMAMLANDI.md`
   - Bu dosyadaki durum tablosunda o satırı `[x]`, "Sıradaki faz" satırını
     bir sonraki numara yap.
   - Bu iki güncellemeyi de commit'e dahil et — yoksa kayıt kodla senkron olmaz.
5. **Doküman senkronu** — aşağıdaki tabloda o faza ait satır varsa uygula.
6. Kullanıcıya "Faz N bitti" diye rapor et.

### Doküman Senkronu — hangi faz hangi satırı geçersiz kılıyor

`CLAUDE.md` her oturumda otomatik yükleniyor. Bir faz onun bir satırını yanlış
hale getirdiği anda düzeltilmeli — Faz 8'e biriktirilirse aradaki her oturum
yanlış talimat okur.

| Faz sonunda | Dosya | Ne yapılacak |
|---|---|---|
| **1** | `CLAUDE.md` §12 "Database" satırı | "tek DB, modul basina ayri schema" → "servis basina ayri DB (tek Postgres konteyneri), schema adlari korunuyor" |
| **3** | `CLAUDE.md` §12 "Moduller arasi iletisim" satırı | "in-process client interface (HTTP YOK, `X-Internal-Secret` YOK)" → "internal REST + `X-Internal-Secret`; side-effect icin RabbitMQ event" |
| **3** | `new-backend/skills.md` §1 | "modul modulu HTTP ile CAGIRMAZ" maddesi — tam metin Faz 3 adım 8'de |
| **4** | `CLAUDE.md` giriş (satır 3) | "Go moduler monolith (`new-backend/`)" → "Go mikroservisler (`new-backend/services/`)" |
| **4** | `CLAUDE.md` §2 tablosu | `new-backend/monolith/**` → `new-backend/services/**` |
| **4** | `CLAUDE.md` §4 tablosu | `new-backend/monolith/` satırı → servis dizinleri, `make sqlc` servis kökünden |
| **4** | `CLAUDE.md` §14 tablosu | generated yollar → `services/<x>-service/internal/db/` |
| **5** | `CLAUDE.md` §13 | Caddy satırı: tek upstream → path-prefix routing |
| **8** | hepsi | Son süpürme + `SYSTEM-DESIGN.md` silme + §0 kaldırma (Faz 8 C bölümü) |

**`frontend/skills.md` ve `mobile/skills.md` hiç değişmiyor** — ikisinde de
backend mimarisine tek atıf yok (`monolith`, `8080`, `backend` kelimeleri
geçmiyor). Route prefix'leri servis sınırlarıyla örtüştüğü için bölünme onlar
için görünmez. Bu dosyalara **dokunma**.

**Kullanıcının tek yapması gereken:** yeni oturumda bu dosyayı okutmak.
`CLAUDE.md` §0 zaten buraya yönlendiriyor, yani hatırlatması bile gerekmez.

**Ortak referanslar** — faz dosyaları bunlara atıf yapar, sadece gerektiğinde aç:

| Dosya | İçerik | Hangi fazlarda zorunlu |
|---|---|---|
| `01-REFERANS-MIMARI.md` | Port / DB / servis / route / event tabloları | hepsi |
| `02-GUVENLIK.md` | OWASP Top 10 karşılığı — bölünmenin yarattığı yeni risk yüzeyi | 1, 3, 4, 5, 6, 7, 8 |
| `03-IZLENEBILIRLIK.md` | Uçtan uca istek takibi: HTTP + event zinciri | 3, 4, 5, 7 |
| `04-PROD-HAZIRLIK.md` | İşletme boşlukları: DLQ, retention, yedekleme, timeout bütçesi | 4, 5, 6, 8 |
| `05-DAYANIKLILIK.md` | Circuit breaker + HTTP idempotency key | 3, 4, 7 |

> **`SYSTEM-DESIGN.md` Faz 8'de silindi.** Monolith mimarisini anlatıyordu ve
> silinmeden önce de koddan sapmıştı. Yerine geçen mimari kaydı
> `01-REFERANS-MIMARI.md`; davranış sorularında **koda** bak. Eski hâli
> gerekirse: `git show c2b34d9:SYSTEM-DESIGN.md`.

---

## Durum Tablosu

| Faz | Dosya | Konu | Durum |
|---|---|---|---|
| 0 | `faz-0-shared-platform-TAMAMLANDI.md` | `platform/` → `shared/platform/` taşıma | [x] |
| 1 | `faz-1-veritabani-ayrimi-TAMAMLANDI.md` | Tek Postgres, 9 ayrı DB + 9 DB kullanıcısı | [x] |
| 2 | `faz-2-period-projeksiyonu-TAMAMLANDI.md` | `academic_periods` cross-service okumasını kaldır | [x] |
| 3 | `faz-3-http-client-katmani-TAMAMLANDI.md` | `shared/client/` internal REST client'ları | [x] |
| 4 | `faz-4-servis-iskeletleri-TAMAMLANDI.md` | 9 × `cmd/main.go` + `go.mod` + `Dockerfile` + `go.work` | [x] |
| 5 | `faz-5-caddy-gateway-TAMAMLANDI.md` | Caddyfile path-prefix routing | [x] |
| 6 | `faz-6-docker-compose-TAMAMLANDI.md` | 16 konteynerli compose + kaynak limitleri | [x] |
| 7 | `faz-7-e2e-dogrulama-TAMAMLANDI.md` | Uçtan uca golden path testi | [x] |
| 8 | `faz-8-temizlik-TAMAMLANDI.md` | `monolith/` kaldırma + doküman güncelleme | [x] |

**Sıradaki faz: YOK — migrasyon tamamlandı.**

> Faz 7 yedi hata çıkardı; üçü bölünmenin kendi regresyonu, ikisi monolith'ten
> gelip in-process çağrılar handler bağlamasını atladığı için gizli kalmış
> hatalardı. Dördü kapsam dışı bırakıldı — listesi Faz 7 dosyasında, hiçbiri
> migrasyon kaynaklı değil.

---

## Kesinleşen Kararlar (yeniden tartışılmaz)

| Konu | Karar |
|---|---|
| Veritabanı | **Tek Postgres konteyneri**, servis başına ayrı DB + ayrı DB kullanıcısı |
| Schema adları | **Değişmiyor** — `auth` DB'sinin içinde `auth` schema'sı. Tüm `.sql` ve sqlc kodu olduğu gibi kalır. |
| Servis paketleme | Her servis kendi konteyneri (9 iş servisi + notification) |
| Sync iletişim | **Internal REST + `X-Internal-Secret`**. gRPC yok (ileride tek seam'de pilot yapılabilir). |
| Async iletişim | **RabbitMQ + outbox pattern** — mevcut yapı korunuyor, değişmiyor |
| Gateway | **Caddy**, path-prefix routing. Traefik/nginx değerlendirildi, elendi. |
| Frontend / Mobil | **Hiç değişmiyor** — route prefix'leri servis sınırlarıyla 1:1 örtüşüyor. `Idempotency-Key` sunucuda opsiyonel, client sonradan ekler. |
| Dayanıklılık | **Circuit breaker** (`sony/gobreaker`, hedef servis başına) + **HTTP idempotency** (Redis, sunucu tarafı) — `05-DAYANIKLILIK.md` |

---

## Genel Kurallar (her fazda geçerli)

- Her faz sonunda `go build ./...` hatasız olmalı.
- Faz 0-3 boyunca **monolith çalışır durumda kalır**. Kırılma Faz 4'te başlar.
- Migration **yazılır**, çalıştırılması için kullanıcıya sorulur (CLAUDE.md §6).
- Docker komutları `sudo` gerektirir → **çalıştırma, kullanıcıya göster** (CLAUDE.md §5).
- Generated dosyalara elle dokunma (`db/*.go`) — `make sqlc-<modul>` çalıştır.
- Commit formatı: `<type>(<scope>): <description>` (CLAUDE.md §7).

---

## Kod Yazım Kuralları

> Bu migrasyon bir **refactor**, yeniden yazım değil. Varsayılan davranış:
> kodu **taşı**, yeniden yazma. Bir dosyayı yeniden yazma ihtiyacı duyuyorsan
> önce "bu gerçekten bu fazın kapsamında mı" diye sor.

### Stack — sabit, tartışılmaz (CLAUDE.md §12)

| Katman | Ne kullanılır | Ne KULLANILMAZ |
|---|---|---|
| Query | **sqlc + pgx/v5** | GORM, ham SQL string, `database/sql` |
| Migration | **goose** | Elle DDL, uygulanmış migration'ı değiştirme |
| HTTP | **Gin v1.11** | Yeni router, echo/fiber/chi |
| Servisler arası sync | **net/http + `shared/client`** | gRPC (şimdilik), yeni RPC kütüphanesi |
| Async | **RabbitMQ + outbox** | Doğrudan publish (payment istisnası hariç) |
| Log | **Zap** | `fmt.Println`, `log` |

Yeni kütüphane eklemek **kullanıcı onayı** gerektirir (CLAUDE.md §6).

**Bu migrasyonda onaylanmış tek yeni kütüphane:** `github.com/sony/gobreaker`
(circuit breaker, `shared/go.mod`'a girer — bkz. `05-DAYANIKLILIK.md` Bölüm B).
Başka hiçbir bağımlılık onaysız eklenmez.

**Ham SQL'in tek meşru istisnası:** `shared/platform/repository/simple_period_repository.go`.
Zaten ham pgx sorgusu kullanıyor (schema-agnostik olması gerektiği için) ve
Faz 2 bunu koruyor. Yeni ham SQL **ekleme**.

### Go idiomları

- **Uber Go Style Guide** (CLAUDE.md §16) — mevcut kod bunu izliyor, sen de izle.
- Hata sarmalama: `fmt.Errorf("...: %w", err)`. Kontrol: `errors.Is` / `errors.As`.
  `err.Error()` string karşılaştırması **yapma**.
- Interface'i **tüketen taraf** tanımlar — mevcut desen bu
  (`StaffClient` catalog'da tanımlı, staff'ta değil). Bunu bozma.
- Compile-time assertion koy: `var _ StaffClient = (*HTTPStaffClient)(nil)`.
  Faz 3 ve 4'te servis sınırındaki her yeni implementasyon için zorunlu.
- `context.Context` ilk parametre, struct'ta saklanmaz.
- Request path'inde `panic` yok. `main.go`'daki `logger.Fatal` başlangıç
  hataları için — o desen korunuyor.
- Kabul edilen kısaltmalar mevcut koddan: `svc`, `repo`, `cfg`, `ctx`, `rg`.

### Hata yapısı — korunacak, değiştirilmeyecek

Mevcut zincir (`new-backend/skills.md` §8):

```
modül errors/ paketi (sentinel)  →  platform/errors.AppError  →  handler HTTP status
```

- `platform/errors.AppError`: `New` / `Wrap` / `WrapWithMessage`;
  kontrol `IsNotFound` / `IsValidation` / `IsUnauthorized` / `IsForbidden` / `IsConflict`
- HTTP'ye çeviri **sadece handler katmanında**
- Kullanıcıya dönen mesaj **Türkçe**, log **İngilizce** (CLAUDE.md §3)

**Migrasyona özel kritik kural:** In-process adapter'lar hata map'lemesi
yapıyor (örnek: `InProcessStaffClient` staff'ın `ErrStaffNotFound`'unu
catalog'un `ErrInstructorNotFound`'una çeviriyor). HTTP karşılıkları
**aynı map'lemeyi birebir** yapmalı — 404 → aynı sentinel. Bu kaybolursa
handler'lar yanlış status code üretir ve frontend hata ekranları bozulur.

### Gin kullanımı

- Middleware zinciri **mevcut haliyle korunur**: global
  (`Recovery → SecurityHeaders → CORS → BodySizeLimit → RequestLogger →
  IPRateLimit → SetCSRFToken`), route seviyesi
  (`JWTAuth → CSRFProtection → UserRateLimit → RequireRole`).
- Servisler ayrılınca bu zincir **her serviste tekrar kurulur** — JWT'yi her
  servis kendi doğrular, auth'a RPC yok.
- Handler'da iş kuralı yok: bind + validate + rol kontrolü. Kural service'te.
- Route mount deseni (`/api/<Name()>`) korunuyor — Caddy routing buna dayanıyor.

### Test

- Test dosyaları paketin **yanında**, ayrı test ağacı yok.
- İsimlendirme: `TestXxx_Scenario_ExpectedResult`.
- Başarısız testi `t.Skip()` ile atlama.
- Faz 3'te yazılan her HTTP client için `httptest` birim testi **zorunlu**.

---

## `new-backend/skills.md` ile Çelişki (Faz 3'te çözülüyor)

`skills.md` §1 şunu diyor:

> **Moduller arasi cagri**: In-process client interface — modul modulu HTTP ile CAGIRMAZ.

Bu monolith kuralı. **Faz 3'ten itibaren geçersiz** — kullanıcı onaylı
migrasyon planı bunun yerine geçer (CLAUDE.md §1 çakışma hiyerarşisi:
kullanıcı prompt'u en üstte).

Faz 3'te `skills.md`'ye bir uyarı satırı ekleniyor; dosyanın tamamı Faz 8'de
yeniden yazılıyor. **Faz 3'ten önce** bu kuralı ihlal etme.

---

## Gözlemlenebilirlik Hazırlığı (şimdi kurma, sadece yerini boş bırakma)

Prometheus / Loki / Grafana **bu migrasyonun kapsamında değil.** Ama sonradan
eklemesi pahalı olan iki şey var — onlar migrasyon sırasında yapılır, gerisi
sonraya kalır.

### Sonradan eklemesi PAHALI (migrasyon sırasında yapılacak)

| Ne | Nerede | Neden sonradan pahalı |
|---|---|---|
| **Uçtan uca istek takibi** (7 halka) | Faz 3, 4, 5 — tarifi `03-IZLENEBILIRLIK.md` | Zincirin **her** halkasına dokunmayı gerektirir: Caddy, HTTP client, outbox migration'ları (8 adet), envelope, consumer. Sonradan eklemek migrasyonu ikinci kez yapmak demek. |
| **Log satırlarında `service` alanı** | Faz 4, her `main.go`'da `logger.Init` | Loki'de `{service="grades"}` sorgusu bunun üstüne kurulur. Sonradan eklemek 10 servise tek tek dokunmak. |
| **Docker log rotasyonu** | Faz 6, compose | 16 konteynerin sınırsız json-file logu homeserver diskini doldurur. Operasyonel hijyen, gözlemlenebilirlik değil. |
| **Güvenlik kapıları** | `02-GUVENLIK.md` faz tablosu | Rate limit kova ayrımı, internal route izolasyonu, DB CONNECT yetkisi — hepsi kurulduktan sonra düzeltmesi, kurarken doğru yapmaktan pahalı. |

### Sonradan eklemesi UCUZ (şimdi yapma)

| Ne | Neden ucuz |
|---|---|
| `/metrics` endpoint | Faz 4'te tüm servisler **tek** `shared/httpserver.NewServer`'dan geçiyor. Prometheus handler'ı oraya eklenince 10 servis birden kazanır — servis başına 0 satır. |
| Prometheus/Grafana/Loki konteynerleri | Compose zaten `base + standalone` overlay desenini kullanıyor. Üçüncü bir `docker-compose.observability.yml` overlay'i doğal ev — Makefile'daki `COMPOSE` değişkenine bir `-f` eklemek yeterli. |
| Trace (OpenTelemetry) | Request ID taşınıyorsa trace'e geçiş kademeli yapılabilir; şimdi OTel bağımlılığı eklemek erken. |
| `/health` + `/ready` bazlı alerting | İkisi de her serviste zaten var. |

**Kural:** Bu tablodaki "ucuz" satırlardan hiçbirini migrasyon sırasında kurma.
Kapsam şişmesi, fazların bitiş kriterlerini bulanıklaştırır.

---

## Geri Dönüş Noktaları

Her faz kendi commit'inde. Bir faz bozarsa:

```bash
git log --oneline -12          # faz commit'lerini gör
git revert <commit>            # tek fazı geri al
```

**`git revert` veriyi geri getirmez.** Faz 1 (DB provisioning) ve Faz 6
(volume silme) durum değiştirir; kodu geri almak DB'yi eski haline döndürmez.
Bu iki fazın dosyasında `pg_dump` adımı var — **atlama**. Geri dönüş sırası:

1. `git revert <faz commit>`
2. `docker compose down -v` (volume'u sil)
3. Yedekten geri yükle (`psql < yedek.sql`)
4. `make deploy`
