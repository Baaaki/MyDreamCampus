# Mikroservis Migrasyonu — Başlangıç Dosyası

> **AI için:** Oturuma bu dosyayla başla. Sadece bu dosyayı ve sıradaki tek faz
> dosyasını oku. Diğer faz dosyalarını **açma** — bağımsız yazıldılar.

---

## Çalışma Protokolü

1. Aşağıdaki **Durum Tablosu**'ndan sıradaki fazı bul (ilk `[ ]` olan satır).
2. O fazın dosyasını oku, **sadece onu** uygula.
3. Faz bitince:
   - Fazın "Bitiş Kriteri" bölümündeki doğrulama komutlarını çalıştır.
   - Atomic commit at (commit mesajı faz dosyasının sonunda yazılı).
   - Dosyayı yeniden adlandır: `faz-N-xxx.md` → `faz-N-xxx-TAMAMLANDI.md`
   - Bu dosyadaki durum tablosunda o satırı `[x]` yap.
4. Sıradaki faza geç veya kullanıcıya "Faz N bitti" diye rapor et.

**Ortak referanslar** — faz dosyaları bunlara atıf yapar, sadece gerektiğinde aç:

| Dosya | İçerik | Hangi fazlarda zorunlu |
|---|---|---|
| `01-REFERANS-MIMARI.md` | Port / DB / servis / route / event tabloları | hepsi |
| `02-GUVENLIK.md` | OWASP Top 10 karşılığı — bölünmenin yarattığı yeni risk yüzeyi | 1, 3, 4, 5, 6, 7, 8 |
| `03-IZLENEBILIRLIK.md` | Uçtan uca istek takibi: HTTP + event zinciri | 3, 4, 5, 7 |

> **`SYSTEM-DESIGN.md`'yi okuma, kaynak olarak kullanma.** O doküman monolith
> mimarisini anlatıyor ve şimdiden koddan sapmış (var olmayan Grafana/Loki
> config'lerini "hazır" gösteriyor, bozuk `/internal/periods` fan-out'unu
> çalışıyor gibi anlatıyor). Mimari bilgi için `01-REFERANS-MIMARI.md`,
> davranış için **koda** bak. Dosya Faz 8'de siliniyor.

---

## Durum Tablosu

| Faz | Dosya | Konu | Durum |
|---|---|---|---|
| 0 | `faz-0-shared-platform.md` | `platform/` → `shared/platform/` taşıma | [ ] |
| 1 | `faz-1-veritabani-ayrimi.md` | Tek Postgres, 9 ayrı DB + 9 DB kullanıcısı | [ ] |
| 2 | `faz-2-period-projeksiyonu.md` | `academic_periods` cross-service okumasını kaldır | [ ] |
| 3 | `faz-3-http-client-katmani.md` | `shared/client/` internal REST client'ları | [ ] |
| 4 | `faz-4-servis-iskeletleri.md` | 9 × `cmd/main.go` + `go.mod` + `Dockerfile` + `go.work` | [ ] |
| 5 | `faz-5-caddy-gateway.md` | Caddyfile path-prefix routing | [ ] |
| 6 | `faz-6-docker-compose.md` | 16 konteynerli compose + kaynak limitleri | [ ] |
| 7 | `faz-7-e2e-dogrulama.md` | Uçtan uca golden path testi | [ ] |
| 8 | `faz-8-temizlik.md` | `monolith/` kaldırma + doküman güncelleme | [ ] |

**Sıradaki faz: 0**

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
| Frontend / Mobil | **Hiç değişmiyor** — route prefix'leri servis sınırlarıyla 1:1 örtüşüyor |

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
