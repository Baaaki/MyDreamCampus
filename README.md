# MyDreamCampus

**MyDreamCampus**, öğrencilerin ders kayıtlarından yoklamalara, not girişlerinden kafeterya işlemlerine kadar tüm üniversite süreçlerini yöneten tam kapsamlı bir platformdur. Hem **Web** hem de **Mobil** uygulama olarak hizmet verir.

Bu proje, hem hızlı geliştirme yapılabilmesi hem de ileride kolayca ölçeklenebilmesi için **Modüler Monolit (Modular Monolith)** mimarisiyle sıfırdan, modern teknolojilerle geliştirilmiştir.

## Ekran Görüntüleri

> _Yer tutucular — proje içi ekran görüntülerini `docs/screenshots/` altına ekleyebilirsiniz._

| Web Arayüzü | Mobil Uygulama |
|-----|--------|
| ![Web dashboard](docs/screenshots/web-dashboard.png) | ![Mobile attendance](docs/screenshots/mobile-attendance.png) |

## Mimari ve Vizyon (Neden Bu Altyapı Seçildi?)

Proje, yönetimi ve dağıtımı zor olan parçalı mikroservis mimarisinden, daha sağlam ve yönetilebilir olan **Modüler Monolit** mimariye geçirilmiştir.

**1. İnsan Kaynakları ve Proje Yönetimi İçin Avantajları:**
- **Hızlı Geliştirme:** Tek bir kod tabanı sayesinde yeni özellikler çok daha hızlı eklenir, ürün pazara daha çabuk çıkar.
- **Düşük Maliyet:** Sunucu maliyetleri ve bakım eforu minimuma indirilmiştir. Sistem az kaynakla çok iş yapar.
- **Mobil ve Web Uyumu:** Tüm platformlar aynı güçlü arka ucu (backend) kullanır, böylece veri tutarsızlığı yaşanmaz.

**2. Yazılım Uzmanları İçin Teknik Detaylar (Geleceğe Hazır Yapı):**
- **Mantıksal İzolasyon:** Her modül (Auth, Öğrenci, Notlar) kendi paketi içinde tamamen izoledir (`internal/modules/`). "Spagetti kod" oluşumu engellenmiştir.
- **Veritabanı İzolasyonu:** Tek bir PostgreSQL veritabanı çalışsa da, her modülün kendi şeması (Schema) vardır. Modüller arası sıkı bağ (Foreign Key) kurulmamıştır.
- **Mikroservise Geçiş (Future-Proof):** Eğer ileride sistem çok büyürse (örn: Ders Kayıt dönemi yoğunluğu), bu mimari sayesinde istenilen modül birkaç saat içinde koparılıp ayrı bir **Mikroservis** olarak dışarı çıkartılabilir. Modüller arası iletişim halihazırda asenkron olarak **RabbitMQ** (Event-Driven) ile sağlanmaktadır.

> **Eski Mimari (Arşiv):** Bu projenin ilk sürümü 9 ayrı mikroservisten oluşuyordu. O mimarinin tüm kaynak kodu, geçiş öncesi son hâliyle `v0-microservices` git tag'i altında dondurulup arşivlendi; `main` dalını kirletmemesi için buradan çıkarıldı. İncelemek için:
>
> ```bash
> git checkout v0-microservices   # eski mikroservis ağacını gez (salt-okunur)
> git checkout main               # güncel modüler monolite geri dön
> ```

## Kullanılan Modern Teknolojiler (Tech Stack)

Sistem tamamen sektör standartlarında, güncel ve yüksek performanslı araçlarla inşa edilmiştir:

*   **Arka Uç (Backend):** Go 1.26, Gin, PostgreSQL 18, RabbitMQ 4.0, Redis 7.2
*   **Ön Yüz (Web):** React 19, Vite, Tailwind CSS v4, shadcn/ui
*   **Mobil Uygulama:** React Native 0.81, Expo 54
*   **Bildirim Sistemi:** E-posta (MailHog ile test) ve Mobil Anlık Bildirim (Push Notification) altyapısı ayrı bir servis olarak asenkron çalışır.

## Güvenlik (Security by Design)

Sistem, OWASP tavsiyeleri temel alınarak katmanlı savunma (defense in depth) prensibiyle tasarlanmıştır.

**Kimlik Doğrulama ve Oturum Yönetimi**
- Parolalar **Argon2id** ile hash'lenir (OWASP önerilen parametreler) ve constant-time karşılaştırılır. Var olmayan kullanıcı için de dummy hash doğrulaması çalıştırılır; login yanıt süresi üzerinden **kullanıcı adı sızdırma (user enumeration)** engellenir.
- **JWT (HS256, algoritma pinlemeli)** + kısa ömürlü access token (15 dk) + **refresh token rotation**. Redis üzerinde JTI blacklist ve token-version takibiyle tek oturum veya tüm oturumlar anında iptal edilebilir (logout-all).
- Token'lar tarayıcıda **httpOnly + Secure + SameSite=Strict** cookie'lerde taşınır; localStorage'da token tutulmaz.
- Brute-force'a karşı **hesap kilitleme** ve Redis tabanlı **rate limiting** (IP / kullanıcı / endpoint bazlı; login gibi hassas endpoint'lerde fail-closed).

**Uygulama Katmanı**
- **RBAC**: admin / teacher / student rolleri route seviyesinde middleware ile zorunlu kılınır.
- **CSRF** koruması (double-submit cookie) ve **CORS** allow-list; production'da eksik CORS yapılandırmasıyla uygulama açılmayı reddeder.
- Security header'ları: **Content-Security-Policy**, **HSTS** (production), X-Frame-Options, nosniff, Referrer-Policy, Permissions-Policy.
- SQL erişimi **sqlc + pgx** ile tamamen parametrize edilir; string birleştirmeli sorgu yoktur (SQL injection yüzeyi kapalı).
- 1 MB **request body limiti** ve slowloris'e karşı HTTP read/write timeout'ları.
- Modüller arası loopback çağrılar **X-Internal-Secret** başlığı ile doğrulanır (constant-time compare).
- Yemekhane QR doğrulaması **HMAC-SHA256** imzalı ve kısa geçerlilik pencereli; imzasız/expired QR reddedilir.
- Güvenlik olayları (başarısız login, hesap kilitleme, yetki ihlali) **audit log**'a yazılır.

**Yapılandırma ve Tedarik Zinciri**
- Production ortamında zayıf veya default secret (JWT, Redis, admin parolası, internal secret) tespit edilirse uygulama **başlamayı reddeder**.
- Altyapı portları (PostgreSQL, Redis, RabbitMQ) yalnızca **127.0.0.1**'e bağlıdır; dışarıya sadece 80/443 açılır.
- CI/CD'de otomatik güvenlik taramaları: **gitleaks** (secret taraması), **CodeQL** (SAST), **gosec** (Go güvenlik lint'i), **govulncheck** (erişilebilir CVE analizi) — her push'ta ve haftalık zamanlanmış olarak çalışır.

## Yerel Ortamda Çalıştırma (Geliştiriciler İçin)

Projeyi kendi bilgisayarınızda test etmek oldukça basittir. 

**Gereksinimler:** Docker, Go 1.26+ ve Node 20+

```bash
# 1. Altyapıyı ayağa kaldırın (Veritabanı, Redis, RabbitMQ vb.)
cd new-backend/infrastructure
docker compose up -d

# 2. Ana Uygulamayı (Backend) başlatın
cd ../monolith
make run

# 3. Bildirim Servisini (E-posta ve Push) başlatın (Yeni bir terminalde)
cd ../services/notification
go run cmd/main.go

# 4. Web Arayüzünü başlatın (Yeni bir terminalde)
cd ../../../frontend
npm install
npm run dev
```

**Erişim Noktaları:**
- Web Arayüzü: `http://localhost:3000`
- Giden E-postaları Görme (MailHog): `http://localhost:8025`
- Backend API: `http://localhost:8080`
- RabbitMQ Yönetim Paneli: `http://localhost:15672`
