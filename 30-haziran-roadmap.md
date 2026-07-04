# MyDreamCampus — Bitirme Roadmap'i (30 Haziran 2026)

> Amaç: Bu projeyi "potansiyeli var" seviyesinden "çalışıyor, deploy edilmiş, mülakatta savunulabilir" seviyesine taşımak.
> Strateji (sabit): **Çalışan demo + tanınır stack keyword'leri** > kod derinliği. Hedef pazar: TR junior/new-grad (Trendyol/Getir/Hepsiburada/Insider tarzı). İş tecrübesi sıfır — beklenti yüksek değil, "çabalamış, şirketin stack'ini öğrenmiş" algısı yeterli.

---

## Eğer sadece 3 şey yapacaksan

1. **Deploy et + canlı link + 2 dk demo videosu.** (Faz 2) — Eşiği geçiren tek şey budur.
2. **Demo akışını bitir:** login → ders kayıt → yemekhane uçtan uca patlamasın. (Faz 1)
3. **README'yi anlatıya çevir** + her teknoloji için 3 cümle "neden". (Faz 4)

Gerisi sinyal/bonus. Bu 3'ü olmadan keyword'ler havada kalır.

---

## Altın Kural

Eklediğin **her** teknoloji için "neden bunu kullandın?" sorusuna **1 cümlelik** cevabın olacak. Cevabın yoksa ekleme. Buzzword-as-decoration geri teper; savunulabilir genişlik istiyoruz, derinlik değil.

---

## Faz 0 — Repo Hijyeni & Kararlar (eşik öncesi temizlik) · Efor: S

Hızlı kazanımlar. Open-source'a açmadan önce.

- [ ] **Eski backend kararı.** Repo'da iki canlı backend var (`legacy-codebase/backend` + `new-backend`). Karar ver:
  - Önerilen: `legacy-codebase`'i ayrı bir git tag/branch'e al (`v0-microservices`), ana koldan çıkar. Migration anlatına referans olarak kalsın ama "iki canlı backend" kafa karışıklığını bitir.
- [ ] **AI izlerini temizle.** Kodda `// plan section 5.5.2`, `// Faz 3` gibi iç referansları sil ([new-backend/monolith/cmd/main.go](new-backend/monolith/cmd/main.go#L133) ve eventbus dosyaları). "AI'a yazdırdım" sinyali veriyor. [utils/jwt.go](new-backend/monolith/internal/platform/utils/jwt.go)'daki mikroservis dönemi yorumları da ("Other services should validate...", "Should ONLY be called by Auth Service") monolitte anlamsız — sil ya da güncelle.
- [ ] **Ölü kod/config temizliği.** Mikroservis kalıntıları: `ExtractUserFromHeaders` / `StripInternalHeaders` middleware'leri (Traefik forward-auth içindi, monolitte hiçbir route kullanmıyor; grades handler'daki bayat yorumlar da bunlara referans veriyor), `PaymentConfig.GRPCAddress` (monolitte gRPC yok). Kullanılmayan kod "taşınmış proje" izlenimi veriyor.
- [ ] **Kök `infrastructure/` artıklarını sil.** İçinde sadece `loki-config.yml/` ve `promtail-config.yml/` adında *boş dizinler* var — eksik dosyayı volume-mount eden eski bir compose'un yarattığı çöp. Klasörü komple kaldır.
- [ ] **Kopya kağıtlarını kaldır.** `CLAUDE.md` ve `*/skills.md` public repo'da kalmasın (ya `.gitignore` ya private). Bunlar senin AI talimatların; işverene gösterilecek şey değil.
- [ ] **Binary/artifact temizliği.** `bin/`, `tmp/` klasörleri git'te olmasın; `.gitignore` kontrol et. Gerekirse geçmişten temizle. Lokal derleme artıkları `new-backend/monolith/main` (45 MB) ve `cmd/monolith_bin` untracked ama `.gitignore`'a ekle ki yanlışlıkla commit'lenmesin.
- [ ] **Kök klasördeki `fix_*.py` scriptleri** (`fix_claude.py`, `fix_issues.py`, `fix_meal.py`, `fix_skills.py`, `legacy-codebase/*.py`) — tek seferlik migration scriptleri, repo'dan çıkar.

**Done:** Repo açılınca tek net backend, AI iç notu yok, kopya kağıdı yok.

---

## Faz 1 — "Çalışır" Eşiği · Efor: M-L · EN KRİTİK

Demo'da gösterilecek altın yolu kurşungeçirmez yap. Ürün hedefi: **yemekhane + ders kayıt**. Demo-kritik akış:
`login → öğrenci ders kaydı → yemekhane kullanımı`. Bu akış patlamamalı.

- [ ] **Build + CI yeşil.** Main'deki CI kırmızıyken hiçbir Faz 1 işine başlama. Durum (02.07.2026):
  - [x] `backend-build` + `backend-test`: enrollment derleme hataları düzeltildi, 834 test yeşil
  - [x] notification: lint (12) + gosec (1) + go.mod tidy düzeltildi
  - [ ] `backend-lint`: monolith'te **52 bulgu** (43 errcheck — unchecked `Close`/`Rollback`/`Ack`/`auditLogger.Log`; 8 staticcheck — SA1029 context key, ST1005, QF10xx; 1 unused func)
  - [ ] `backend-security`: monolith'te **40 gosec bulgusu** (34 G115 int→int32/16 dönüşümü, 3 G109, 2 G118 goroutine context). Ya bounds-check'li güvenli dönüşümlerle düzelt ya da gerekçeli gosec konfigürasyonu yaz — sessizce `#nosec` serpiştirme.
  - Bundan sonra her commit öncesi `make build && make test` zorunlu.
- [ ] **Prerequisite bypass'ını kapat.** [enrollment_service.go:335-338](new-backend/monolith/internal/modules/enrollment/service/enrollment_service.go#L335-L338) — `passed := true` stub'ı gerçek `gradesClient.CheckPrerequisitePassed(...)` çağrısıyla değiştir. Bu hem bayrak feature'ı tamamlar hem modüller-arası seam örneği olur (mülakatta gösterilecek şey).
- [ ] **Demo-kritik modülleri uçtan uca test et (manuel):**
  - [ ] auth: login / logout / refresh çalışıyor
  - [ ] enrollment: öğrenci ders seçiyor, kapasite/çakışma/prerequisite kontrolleri devrede
  - [ ] meal (yemekhane): kredi/ödeme akışı uçtan uca (payment mock'la)
  - [ ] attendance: yoklama kaydı çalışıyor (Faz 3'te Kafka'ya taşınacak)
- [ ] **Frontend golden path** (3 rol: admin/teacher/student): her rolün ana dashboard'u ve demo-kritik sayfaları açılıyor, 500/boş ekran yok.
- [ ] **Mobile (Expo) minimum:** login + en az 1 anlamlı ekran (ör. ders programı veya yoklama) gerçek backend'e bağlı çalışıyor. Yarım ekranları "planned" olarak gizle, demo'da gösterme.
- [ ] **Dockerfile yaz (ön şart — şu an repo'da hiç yok).** Monolith için multi-stage Dockerfile (+ notification için ikinci bir tane), compose'a `monolith` servisi olarak ekle. Ek A'daki Caddy adımı (`reverse_proxy monolith:8080`) ve Faz 3 K8s bu olmadan çalışmaz.
- [ ] **Tam stack local'de tek komutla ayağa kalkıyor:** `docker compose up` → backend + frontend + Postgres + Redis + RabbitMQ. Sıfırdan kurulum README adımlarıyla doğrulanıyor. (Not: compose şu an yalnızca altyapı servislerini içeriyor; grafana/loki/promtail klasörleri boş — observability ya gerçekten ekle ya da bahsini kaldır.)
- [ ] **Smoke test:** demo-kritik akış için 2-3 service-level happy-path testi (depth değil, "çalışıyor" kanıtı).

**Done:** Temiz makinede `docker compose up` sonrası login→ders kayıt→yemekhane akışı baştan sona çalışıyor.

---

## Faz 2 — Deploy + Canlı Demo · Efor: M · EŞİĞİ GEÇİREN ADIM

Bu olmadan proje "öğrenme projesi"; bununla "çalışan ürün".

- [ ] **Bir yere deploy et.** Önerilen sıra: ucuz VPS (Hetzner/DO ~5$) > Fly.io > Railway. Monolith tek binary olduğu için tek VPS yeter.
- [ ] **Canlı URL** — frontend + API erişilebilir, HTTPS (Caddy/Traefik ya da platform TLS).
- [ ] **Seed data** — demo için hazır kullanıcılar (admin/teacher/student), örnek dersler, yemekhane menüsü. Hiring manager login olup gezebilmeli.
- [ ] **README'ye:** canlı link + demo login bilgileri + **2 dakikalık demo videosu** (Loom/YouTube unlisted).
- [ ] **Gerçek ekran görüntüleri** (placeholder'ları değiştir).

**Done:** README'deki linke tıklayan biri, hesapla girip ders kaydı + yemekhane akışını kendi görebiliyor.

---

## Faz 3 — Tanınır Stack Sinyalleri · Efor: M · BONUS (eşik sonrası)

Sadece Faz 1-2 bittikten sonra. Her biri savunulabilir yere konacak.

- [ ] **Kafka → attendance.** Yoklama yüksek-hacimli append stream (arch doc: 100k istek/2 saat, burst 50-100 RPS). Onu Kafka'ya taşı; **geri kalan düşük-hacimli routing event'leri RabbitMQ'da kalsın**. Savunma cümlesi: *"Yoklama yüksek-throughput append akışı, Kafka'ya uygun; rotalama gerektiren düşük-hacimli event'ler RabbitMQ'da."*
- [ ] **Kubernetes.** Monolith + notification + Postgres/Redis/RabbitMQ için temel manifest'ler (Deployment, Service, ConfigMap, Secret). Local'de kind/minikube, ya da küçük managed cluster. Savunma: *"Tek binary'i K8s'e koydum çünkü şirket stack'i; HPA ile yatay ölçeklemeyi denedim."*
- [ ] **Terraform (opsiyonel, K8s ile birlikte).** En azından tek `main.tf` ile VPS/cluster + DNS provisioning. "IaC" keyword'ünü doldurur.
- [ ] **Elasticsearch (lüks, sadece zaman kalırsa).** Course/student full-text arama. Savunma: *"İsim/ders araması için full-text, Postgres ILIKE yerine."*

**Done:** Her eklenen tech çalışıyor + 1 cümlelik savunması hazır. Yarım/bozuk eklenti yok.

---

## Faz 4 — Anlatı & Mülakat Hazırlığı · Efor: S-M

Aradığın "araştırmış, öğrenmiş" algısı koddan değil **README'den** çıkar.

- [ ] **README "Tech & Kararlar" bölümü.** Her ana teknoloji için 3 cümle: *neden ekledim, alternatifi neydi, ne öğrendim.* (Go, Postgres, sqlc, RabbitMQ, Kafka, Redis, K8s, modular monolith.)
- [ ] **Migration anlatısı.** "Microservice'ten modular monolith'e neden geçtim" — README'de bir bölüm ya da kısa blog yazısı. Bu, projeden bile güçlü bir senior sinyali (yargı gösterir).
- [ ] **Mülakat kartı (kendine, repo'ya koyma).** Her teknoloji + her mimari karar için 1 cümlelik "neden". Sorulduğunda ezberden değil, anlayarak cevap ver:
  - Modular monolith neden? (tek dev, ops maliyeti, seam korundu)
  - Outbox pattern ne işe yarar?
  - Kafka vs RabbitMQ farkı, neden ikisi birden?
  - Argon2id neden bcrypt değil?
  - fail-closed vs fail-open trade-off'u?

**Done:** README'yi okuyan biri kodu açmadan "bu kişi düşünmüş" diyebiliyor; sen her kararı savunabiliyorsun.

---

## Önceliklendirme Özeti

| Faz | Ne | Tip | Sıra |
|---|---|---|---|
| 0 | Repo hijyeni | Eşik öncesi | İlk, hızlı |
| 1 | Çalışır akış (demo-kritik) | **Eşik (zorunlu)** | 1 |
| 2 | Deploy + canlı demo | **Eşik (zorunlu)** | 2 |
| 3 | Kafka / K8s / Terraform | Sinyal (bonus) | 3 |
| 4 | README anlatısı + mülakat hazırlığı | Sinyal (zorunlu) | 1-2 ile paralel |

**Kural:** Faz 3'e Faz 1+2 bitmeden başlama. Bozuk demo üstüne Kafka eklemek negatif sinyal.

---

## Projenin "Bitti" Tanımı

- [ ] Canlı, erişilebilir URL + demo hesapları + demo videosu
- [ ] login → ders kayıt → yemekhane akışı uçtan uca çalışıyor
- [ ] Mobile'da en az 1 anlamlı ekran canlı backend'e bağlı
- [ ] En az 1 tanınır enterprise tech savunulabilir şekilde eklenmiş (Kafka önerilen)
- [ ] README: canlı link + tech kararları + migration anlatısı
- [ ] Repo hijyeni temiz (AI izi yok, kopya kağıdı yok, tek net backend)
- [ ] Her mimari kararı 1 cümleyle savunabiliyorsun

---

## Ek A — Deploy Rehberi (Faz 2)

### Seçenek 1 (önerilen): DigitalOcean Droplet + docker-compose

Mevcut `docker-compose.yml` olduğu için en hızlı yol. Monolith tek binary → tek Droplet yeter.

1. **Droplet oluştur:** Ubuntu 24.04, **2GB RAM ($12/ay)** (dar bütçede 1GB'da observability'yi kapatarak da olur), SSH key ekle, sana yakın bölge (Frankfurt).
2. **Cloud Firewall (DO panelinden):** sadece **22 (SSH, ideali kendi IP'ine kısıtla)**, **80**, **443** açık. RabbitMQ UI (15672), Postgres (5432), Redis (6379), Grafana **internete kapalı** kalsın.
3. **SSH gir + Docker kur:**
   ```bash
   ssh root@<droplet-ip>
   curl -fsSL https://get.docker.com | sh        # Docker + compose plugin
   ```
4. **Repo + secrets:**
   ```bash
   git clone <repo-url> && cd mydreamcampus
   cp new-backend/infrastructure/.env.example new-backend/infrastructure/.env
   cp new-backend/monolith/.env.example new-backend/monolith/.env
   openssl rand -base64 48     # JWT_SECRET ve QR_SECRET üretmek için (iki ayrı değer)
   ```
   **`ENVIRONMENT=production` ayarla — bu satır olmadan prod korumalarının hiçbiri devreye girmez:** config default-secret reddi, HSTS, Secure cookie ve CORS zorunluluğu hepsi bu değişkene bağlı. Development modda default JWT secret'la sessizce açılır.

   Prod'da **zorunlu** doldurulacaklar (eksiği startup'ta panic/hata verir):
   | Değişken | Not |
   |---|---|
   | `ENVIRONMENT=production` | Tüm prod validasyonlarının anahtarı |
   | `JWT_SECRET` | 32+ byte, `openssl rand -base64 48` |
   | `DB_URL` / `POSTGRES_PASSWORD` | Güçlü parola, iki dosyada tutarlı |
   | `REDIS_PASSWORD` | Default `changeme_redis_secret` prod'da reddedilir; compose ve monolith .env'de aynı değer |
   | `CORS_ALLOWED_ORIGINS` | `https://demo.alanadin.com` — yoksa startup panic |
   | `QR_SECRET` | Yemekhane QR imzası, default reddedilmeli |
   | `ADMIN_INITIAL_PASSWORD` | Default `Admin123!` prod'da reddedilir |
   | `FRONTEND_STATIC_ENABLED=true` + `FRONTEND_STATIC_DIR` | SPA'yı Gin servis edecekse |

   Frontend build'i de unutma: `cd frontend && bun install && bun run build` → çıktıyı `FRONTEND_STATIC_DIR`'in gösterdiği yere kopyala (örn. `frontend_dist/`).
5. **Reverse proxy + otomatik HTTPS — Caddy.** (Ön şart: Faz 1'deki Dockerfile görevi — `monolith` compose'ta bir servis olmadan `monolith:8080` adresi çözülmez.) Compose'a bir Caddy servisi ekle (ya da host'ta çalıştır). Monolith Gin hem `/api/*` hem static frontend'i tek binary'de servis ettiği için Caddy sadece `:8080`'e proxy'ler. Minimal `Caddyfile`:
   ```
   demo.seninalanadin.com {
       reverse_proxy monolith:8080
   }
   ```
6. **Ayağa kaldır:**
   ```bash
   docker compose -f new-backend/infrastructure/docker-compose.yml up -d
   ```
7. **Migration + seed:** `make migrate-up` (ya da compose içindeki migrate servisi) + demo seed data (admin/teacher/student hesapları, örnek ders/menü).
8. **DNS:** domain'in A kaydını Droplet IP'sine yönlendir.
9. **Doğrula:** `https://demo.seninalanadin.com` açılıyor, demo hesabıyla login → ders kayıt → yemekhane akışı çalışıyor.

> Faz 3'te K8s keyword'ü istersen aynı DO hesabında **DOKS**'a taşı — "önce Droplet, sonra K8s öğrenmek için DOKS" tutarlı bir anlatı.

### Seçenek 2 (alternatif): Başka bir VPS — aynı yöntem

DO kullanmazsan **yöntem birebir aynı**, sadece sağlayıcı değişir. En ucuzu **Hetzner Cloud** (~€4/ay, 2GB CX22). 1-4. adımlar (Docker kur → repo → .env → compose up → Caddy) aynen geçerli. docker-compose yaklaşımı taşınabilir; bu yüzden hangi VPS olduğu önemli değil.

### Seçenek 3 (alternatif): Railway — SSH'sız PaaS

Sunucu yönetmek istemezsen: **Railway**'de git push ile deploy. Postgres + Redis tek tıkla managed eklenti; RabbitMQ servis template'i var; TLS otomatik. Demo'lar için pratik ama: ücretsiz kota sınırlı, self-hosted RabbitMQ/observability daha az kontrol. (Fly.io de benzer; CLI'dan `fly deploy` ile Dockerfile deploy eder, managed Postgres + Upstash Redis + CloudAMQP gibi dış servislerle birleştirilir.)

**Özet:** Çalışan demo'ya en hızlı yol = bir VPS (DO/Hetzner) + docker-compose + Caddy. Railway/Fly, "sunucu yönetmek istemiyorum" diyorsan makul alternatif.
