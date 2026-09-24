# MyDreamCampus

**MyDreamCampus**, öğrencilerin ders kayıtlarından yoklamalara, not girişlerinden kafeterya işlemlerine kadar tüm üniversite süreçlerini yöneten tam kapsamlı bir platformdur. Hem **Web** hem de **Mobil** uygulama olarak hizmet verir.

Sistem, her biri kendi veritabanına ve kendi konteynerine sahip **10 mikroservisten** oluşur; hepsinin önünde tek bir Caddy ağ geçidi durur.

## Ekran Görüntüleri

> _Yer tutucular — proje içi ekran görüntülerini `docs/screenshots/` altına ekleyebilirsiniz._

| Web Arayüzü | Mobil Uygulama |
|-----|--------|
| ![Web dashboard](docs/screenshots/web-dashboard.png) | ![Mobile attendance](docs/screenshots/mobile-attendance.png) |

## Mimari

Sistem 10 servise bölünmüştür — her biri ayrı bir Go modülü, ayrı bir
konteyner ve **kendi veritabanı**:

| Servis | Sorumluluk | Servis | Sorumluluk |
|---|---|---|---|
| `auth` | Kimlik doğrulama, oturum, token | `attendance` | Yoklama oturumları, QR okutma |
| `staff` | Öğretim üyeleri, profiller, idari personel rehberi | `grades` | Not girişi, finalizasyon |
| `student` | Öğrenci kayıtları, danışman | `meal` | Yemekhane rezervasyonu |
| `catalog` | Ders kataloğu, dönemler | `payment` | Yemekhane ödemesi (test kartı, gerçek para yok) |
| `enrollment` | Ders seçimi, danışman onayı | `notification` | E-posta bildirimi (push iskelet) |

**Servisler birbirini nasıl görür**

- **Senkron okuma / doğrulama:** internal REST (`/internal/*`), `X-Internal-Secret`
  başlığıyla imzalı. Bu yollar ağ geçidinden dışarı açılmaz. Her çağrının
  hedef servis başına bir **devre kesicisi** (circuit breaker) vardır: bir
  servis düştüğünde çağıran taraf beklemek yerine anında hata döner.
- **Yan etki / bildirim:** **RabbitMQ** üzerinden olay (event) — her yayın
  **outbox** tablosundan geçer, böylece iş kaydı yazılıp olayın kaybolduğu bir
  ara durum oluşmaz.
- **Veri izolasyonu:** Bir servis başka bir servisin veritabanına bağlanamaz;
  bağlanma yetkisi rol seviyesinde verilmemiştir. Başka servisin verisi ya
  internal REST ile okunur ya da olayla kendi tarafına projekte edilir.
- **Tek giriş kapısı:** Tarayıcı yalnızca Caddy'yi görür. Caddy hem SPA'yı
  sunar hem `/api/<önek>` yolunu ilgili servise yönlendirir — aynı origin,
  CORS yok.

Neden bölündü: yoğunluk dönemsel ve dengesiz (ders kayıt haftası enrollment'ı,
öğle arası meal'i zorlar), ve tek servisin yeniden başlatılması diğer dokuzunu
etkilemez. Bedeli, ağ üzerinden yapılan çağrıların hata yüzeyidir — devre
kesici, idempotency anahtarları ve outbox bunun için var.

**Mimari referans:** [`microservices-migration/01-REFERANS-MIMARI.md`](microservices-migration/01-REFERANS-MIMARI.md)
(servis / port / veritabanı / route / olay tabloları)

> **Arşiv:** Proje daha önce iki mimari denedi — ilk sürüm 9 ayrı mikroservis,
> ardından modüler monolit. İkisinin de kaynağı `v0-microservices` git tag'i
> altında dondurulmuştur. İncelemek için:
>
> ```bash
> git checkout v0-microservices   # eski ağacı gez (salt-okunur)
> git checkout main               # güncel sürüme dön
> ```

## Kullanılan Modern Teknolojiler (Tech Stack)

Sistem tamamen sektör standartlarında, güncel ve yüksek performanslı araçlarla inşa edilmiştir:

*   **Arka Uç (Backend):** Go 1.27, Gin, PostgreSQL 18, RabbitMQ 4.3, Redis 8.10
*   **Ön Yüz (Web):** React 19, Vite, Tailwind CSS v4, shadcn/ui
*   **Mobil Uygulama:** React Native 0.86, Expo 57
*   **Bildirim Sistemi:** Ayrı bir servis olayları asenkron tüketip e-posta gönderir (geliştirmede MailHog, üretimde `.env`'deki SMTP). Mobil push gönderimi henüz iskelet: çağrılar yalnızca loglanır.

## Güvenlik (Security by Design)

Sistem, OWASP tavsiyeleri temel alınarak katmanlı savunma (defense in depth) prensibiyle tasarlanmıştır.

**Kimlik Doğrulama ve Oturum Yönetimi**
- Parolalar **Argon2id** ile hash'lenir (OWASP önerilen parametreler) ve constant-time karşılaştırılır. Var olmayan kullanıcı için de dummy hash doğrulaması çalıştırılır; login yanıt süresi üzerinden **kullanıcı adı sızdırma (user enumeration)** engellenir.
- **JWT (HS256, algoritma pinlemeli)** + kısa ömürlü access token (15 dk) + **refresh token rotation**. Redis üzerinde JTI blacklist ve token-version takibiyle tek oturum veya tüm oturumlar anında iptal edilebilir (logout-all).
- Token'lar tarayıcıda **httpOnly + SameSite=Strict** (production'da **Secure**) cookie'lerde taşınır; refresh token yanıt gövdesine yazılmaz ve yalnızca `/api/auth` altına gönderilir. localStorage'da token tutulmaz. Access ve refresh token'ları `token_type` ile ayrılır; refresh token bir API isteğini doğrulayamaz.
- Brute-force'a karşı **hesap + istemci adresi bazlı geçici kilitleme**: kilitli deneme yanlış şifreyle aynı yanıtı alır (kullanıcı adı sızmaz) ve saldırgan, hesap sahibini başka bir adresten dışarıda bırakamaz. Redis tabanlı **rate limiting** kimliği doğrulanmış istekte kullanıcı, anonim istekte istemci IP'si bazlıdır; login gibi hassas endpoint'lerde fail-closed.

**Uygulama Katmanı**
- **RBAC**: admin / teacher / student rolleri route seviyesinde middleware ile zorunlu kılınır.
- **CSRF** koruması (double-submit cookie) ve **CORS** allow-list; production'da eksik CORS yapılandırmasıyla uygulama açılmayı reddeder.
- Security header'ları: **Content-Security-Policy**, **HSTS** (production), X-Frame-Options, nosniff, Referrer-Policy, Permissions-Policy.
- SQL erişimi **sqlc + pgx** ile tamamen parametrize edilir; string birleştirmeli sorgu yoktur (SQL injection yüzeyi kapalı).
- 1 MB **request body limiti** ve slowloris'e karşı HTTP read/write timeout'ları.
- Servisler arası çağrılar **X-Internal-Secret** başlığı ile doğrulanır (constant-time compare); `/internal/*` yolları ağ geçidinden dışarı açılmaz.
- Yoklama ve yemekhane QR'ları **HMAC-SHA256** imzalıdır ve kısa bir zaman penceresine bağlıdır. Yoklama QR'ı 15 saniyede bir yenilenir; sınıftan paylaşılan bir fotoğraf birkaç saniye içinde geçersizleşir. İmzasız veya süresi geçmiş QR reddedilir.
- Güvenlik olayları (başarısız login, hesap kilitleme, yetki ihlali) **audit log**'a yazılır.

**Yapılandırma ve Tedarik Zinciri**
- Production ortamında zayıf veya default secret (JWT, Redis, admin parolası, internal secret) tespit edilirse uygulama **başlamayı reddeder**.
- Altyapı portları (PostgreSQL, Redis, RabbitMQ) yalnızca **127.0.0.1**'e bağlıdır; dışarıya sadece 80/443 açılır.
- CI/CD'de otomatik güvenlik taramaları: **gitleaks** (secret taraması), **CodeQL** (SAST), **gosec** (Go güvenlik lint'i), **govulncheck** (erişilebilir CVE analizi) — her push'ta ve haftalık zamanlanmış olarak çalışır.

## Yerel Ortamda Çalıştırma (Geliştiriciler İçin)

**Gereksinimler:** Docker, Go 1.27+, Bun (web), Node 20+ (mobil)

### Tümü container'da (önerilen)

Tek komut 16 konteyneri ayağa kaldırır — altyapı, 10 servis, migration, demo
veri ve SPA'yı sunan Caddy dahil:

```bash
cp new-backend/infrastructure/.env.example new-backend/infrastructure/.env
# .env içindeki CHANGE_ME değerlerini doldurun: openssl rand -base64 48
make deploy
```

Sonrasında `make deploy-ps` durumu, `make deploy-logs` logları gösterir.
Tek bir servisi diğerlerine dokunmadan güncellemek için `make deploy-grades`.

### Elle çalıştırma (hot reload)

10 servisi elle çalıştırmak bir iş akışı değil; bu yol tek bir servis üzerinde
çalışırken kullanılır:

```bash
# 1. Altyapı (Postgres, Redis, RabbitMQ, MailHog + migration)
#    Repo kökünden çalıştırın: `make` doğru compose dosyalarını birlikte yükler
#    ve infra portlarını 127.0.0.1'e açar (çıplak `docker compose up` açmaz).
make infra

# 2. Üzerinde çalıştığınız servisi host'ta başlatın
make run-grades          # veya run-auth, run-catalog, run-notification ...

# 3. Web arayüzü (yeni terminal)
cd frontend && bun install && bun dev
```

**Erişim Noktaları:**
- Web Arayüzü (container'da): `http://localhost` — Vite dev sunucusu: `http://localhost:3000`
- Giden E-postaları Görme (MailHog): `http://localhost:8025`
- RabbitMQ Yönetim Paneli: `http://localhost:15672`
- API: doğrudan servis portu yoktur — hepsi Caddy üzerinden `/<origin>/api/<önek>`

Sunucuya kurulum ve dağıtım için: [`DEPLOY.md`](DEPLOY.md).

## Lisans ve İletişim

Bu proje **GNU General Public License v3.0 (GPL-3.0)** altında lisanslanmıştır. Detaylar için [LICENSE](LICENSE) dosyasına bakabilirsiniz.

İletişim ve destek: [contact@madebybaki.com](mailto:contact@madebybaki.com)
