# Deploy Rehberi

Proje tek bir makinede `docker compose` ile **tek komutta** ayağa kalkar:
Caddy (SPA + `/api` proxy) → monolith → Postgres/Redis/RabbitMQ.

> **Ayrı bir "frontend sunucusu" yok.** SPA `bun run build` ile statik dosyalara
> derlenip Caddy imajının içine kopyalanıyor ([frontend/Dockerfile](frontend/Dockerfile)).
> Caddy 80/443'ü dinler: `/api/*` → monolith:8080, geri kalan her şey SPA.
> Tarayıcı tek origin görür — CORS yok, ayrı port yok.

İki senaryo var:

| | **A. Ev sunucusu (LAN)** | **B. Public VPS** |
|---|---|---|
| Erişim | Ev ağındaki cihazlar | İnternetten herkes |
| Adres | `http://192.168.1.50` | `https://203-0-113-5.sslip.io` |
| HTTPS | Yok (özel IP'ye sertifika verilmez) | Let's Encrypt, otomatik |
| Kurulum | Aşağıdaki **A** bölümü | **B** bölümü |

---

## A. Ev sunucusu — tek komut

### A1. Sunucuda Docker — ve şifre sormasını kapat

```bash
curl -fsSL https://get.docker.com | sh

# Docker'i sudo'suz kullan: kendini docker grubuna ekle
sudo usermod -aG docker $USER
newgrp docker          # ya da: cikis yapip tekrar SSH ile baglan

docker info >/dev/null && echo "sudo'suz calisiyor"
```

> Bu adım olmadan her `make deploy` sudo şifresi sorar — ve `ssh sunucu 'make
> deploy'` gibi tek satırlık uzaktan komutlar tty olmadığı için tamamen
> başarısız olur. Makefile sudo'ya **sadece** docker soketine doğrudan
> erişemediğinde başvurur (`SUDO := $(shell docker info ... || echo sudo)`),
> yani bu adımdan sonra hiçbir komut şifre istemez.
>
> Güvenlik notu: `docker` grubu pratikte root yetkisine denktir (container
> host'un diskini mount edebilir). Zaten sudo yetkin olan kendi ev sunucunda
> bu kabul edilebilir; çok kullanıcılı bir makinede tercih etme.

### A2. Repoyu al ve LAN IP'sini öğren

```bash
git clone <REPO_URL> mydreamcampus
cd mydreamcampus
hostname -I | awk '{print $1}'      # örn. 192.168.1.50 — not al
```

> Bu IP'yi router'ın DHCP ayarlarından **sabitle** (static lease), yoksa
> sunucu yeniden başlayınca adres değişir ve `.env`'i güncellemen gerekir.

### A3. Secrets

```bash
cp new-backend/infrastructure/.env.example new-backend/infrastructure/.env
nano new-backend/infrastructure/.env
```

Her `CHANGE_ME` için ayrı ayrı `openssl rand -base64 48` çalıştırıp yapıştır.
`PUBLIC_HOST` / `PUBLIC_ORIGIN` **kendi LAN IP'n** olacak, `http://` önekiyle:

```
PUBLIC_HOST=http://192.168.1.50
PUBLIC_ORIGIN=http://192.168.1.50
```

> `http://` öneki kritik: Caddy şemayı açıkça görünce otomatik HTTPS'i kapatır.
> Öneksiz bırakırsan Let's Encrypt'ten özel IP için sertifika almaya çalışır ve
> başarısız olur. `localhost` da yazma — o sadece sunucunun kendisi demek,
> telefonundan/laptop'undan erişemezsin.

### A4. Başlat — **tek komut, repo kökünden**

```bash
make deploy
```

Bu komut her şeyi yapar: frontend'i derler, Go binary'lerini derler, infra'yı
kaldırır, migration'ları uygular, monolith + notification + Caddy'yi başlatır ve
demo veriyi seed eder. İlk sefer 3-6 dakika (sonrakiler ~30 sn).

Sunucuda dizin değiştirmene gerek yok; hepsi kökten:

```bash
make deploy-ps       # durum
make deploy-logs     # canlı log (monolith + caddy)
make deploy-update   # git pull + değişenleri derle + yeniden başlat
make deploy-down     # durdur (veri korunur)
```

### A5. Firewall (ufw kuruluysa)

```bash
sudo ufw allow 80/tcp
```

### A6. Doğrula

Ev ağındaki **herhangi bir cihazdan** tarayıcı: `http://192.168.1.50`

Giriş bilgileri için aşağıdaki [demo hesaplar](#demo-giriş-bilgileri) tablosuna bak.

### A7. Sunucu yeniden başlarsa

Bir şey yapman gerekmez — tüm servisler `restart: unless-stopped` ile
tanımlı, Docker daemon boot'ta kalkınca stack de kalkar. `make deploy` yalnızca
ilk kurulumda ve kod güncellemesinde gerekir.

### Mobil (Expo) uygulamayı bağlamak

Telefon Caddy üzerinden değil, doğrudan `8080`'e gitmek isterse override
dosyasıyla kaldır (base compose 8080'i dışarı açmaz):

```bash
sudo docker compose \
  -f new-backend/infrastructure/docker-compose.yml \
  -f new-backend/infrastructure/docker-compose.override.yml up -d
```

Sonra `mobile/.env` içinde API adresini `http://192.168.1.50:8080` yap.

### İnternete açmak istersen (opsiyonel)

Ev bağlantılarında port yönlendirme çoğu zaman çalışmaz (ISP CGNAT arkasına
alır, 80/443 kapalı olabilir). Router'da port açmak yerine **Cloudflare Tunnel**
kullan: giden bağlantı kurar, port yönlendirme gerektirmez, gerçek HTTPS ve
domain verir. Tunnel'ı `localhost:80`'e (Caddy) yönlendir, sonra `.env`'de
`PUBLIC_HOST` / `PUBLIC_ORIGIN`'i tunnel domain'ine çevirip `make deploy` çalıştır.

---

## B. DigitalOcean droplet

Domain almadan **gerçek HTTPS** için `sslip.io` kullanıyoruz.

İlk kez VPS deploy'u yapıyorsan sırayla takip et; her komut kopyala-yapıştır.

---

## 0. Ön koşul

- DigitalOcean hesabı ($200 kredi yeterli — 4GB droplet ~8 ay)
- Bilgisayarında bir SSH anahtarı (`ls ~/.ssh/id_ed25519.pub`; yoksa `ssh-keygen -t ed25519`)
- SSH public key'i DigitalOcean → Settings → Security → **Add SSH Key**

---

## 1. Droplet oluştur

DigitalOcean panelinden **Create → Droplet**:

| Ayar | Değer |
|---|---|
| Image | Ubuntu 24.04 LTS |
| Plan | **Basic → Regular, 4GB / 2 vCPU** (~$24/ay). Bütçe: 2GB + swap (bkz. adım 4b) |
| Region | Frankfurt (FRA1) — Türkiye'ye en yakın |
| Authentication | **SSH Key** (parola değil) |
| Hostname | `mydreamcampus` |

Oluşunca **IP adresini not al** (örnek: `203.0.113.5`).

---

## 2. Firewall (DigitalOcean panelinden)

Networking → Firewalls → **Create Firewall**. Sadece şunlar açık olsun:

- Inbound: **SSH 22**, **HTTP 80**, **HTTPS 443**
- Postgres/Redis/RabbitMQ portları **kapalı** kalsın (compose zaten bunları sadece `127.0.0.1`'e bağlıyor).

Firewall'ı droplet'e ata.

---

## 3. Sunucuya bağlan + Docker kur

```bash
ssh root@203.0.113.5          # kendi IP'nle değiştir

# Docker + compose plugin (tek komut)
curl -fsSL https://get.docker.com | sh
docker version                # çalıştığını doğrula
```

---

## 4. Repoyu al + secrets hazırla

```bash
git clone <REPO_URL> mydreamcampus
cd mydreamcampus/new-backend/infrastructure
cp .env.example .env
```

`.env`'i düzenle (`nano .env`). **PUBLIC_HOST'u IP'nden üret**: noktaları tireye
çevir + `.sslip.io` ekle. `203.0.113.5` → `203-0-113-5.sslip.io`.

Secret üretmek için (her biri için ayrı çalıştır, çıktıyı yapıştır):

```bash
openssl rand -base64 48
```

Doldurulacaklar: `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, `RABBITMQ_PASSWORD`,
`JWT_SECRET`, `INTERNAL_SERVICE_SECRET`, `QR_SECRET`, `ADMIN_INITIAL_PASSWORD`,
`PUBLIC_HOST`, `PUBLIC_ORIGIN`.

> Monolith `ENVIRONMENT=production` ile çalışır ve secret'lar default kalırsa
> **başlamayı reddeder** — bu bilinçli bir güvenlik önlemi.

### 4b. (Sadece 2GB droplet'te) swap ekle — build OOM olmasın

```bash
fallocate -l 2G /swapfile && chmod 600 /swapfile
mkswap /swapfile && swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab
```

---

## 5. Build + ayağa kaldır

```bash
# hâlâ new-backend/infrastructure/ içindeyken
docker compose build          # ilk sefer 3-6 dk (Go + frontend derlenir)
docker compose up -d
```

Sıra otomatik: infra sağlıklı olunca `migrate` çalışır → bitince `monolith` +
`notification` başlar → `caddy` TLS sertifikasını çeker.

Migration loglarını gör:

```bash
docker compose logs migrate           # ">> migrations complete" görmelisin
docker compose logs -f monolith       # panik/hata var mı
docker compose ps                     # hepsi "running", migrate "exited (0)"
```

---

## 6. Doğrula

Tarayıcıda: **`https://203-0-113-5.sslip.io`** (kendi host'unla).

- Sertifika uyarısı **çıkmamalı** (Caddy Let's Encrypt aldı). Çıkarsa 1-2 dk
  bekle (ACME) ve `docker compose logs caddy` bak.
- Admin ile giriş: `.env`'deki `ADMIN_EMAIL` + `ADMIN_INITIAL_PASSWORD`.

---

## 7. Seed (demo verisi) — otomatik

Manuel bir şey yapman gerekmez:

- **Admin** ilk açılışta otomatik oluşur (`.env`'deki `ADMIN_EMAIL` /
  `ADMIN_INITIAL_PASSWORD`).
- **`seed` servisi** monolith ayağa kalkınca otomatik çalışır; demo öğretmen,
  ders ve öğrencileri **gerçek admin API üzerinden** oluşturur (event zinciri
  düzgün dolsun diye — ham SQL değil). Tekrar çalıştırmak güvenli (idempotent).
  Kapatmak istersen `.env`'de `SEED_DEMO=false`.

Seed loglarını gör:

```bash
docker compose logs seed        # ">> seed complete" görmelisin
```

### Demo giriş bilgileri

| Rol | E-posta | Şifre |
|---|---|---|
| Admin | `.env`'deki `ADMIN_EMAIL` | `.env`'deki `ADMIN_INITIAL_PASSWORD` |
| Öğretmen | `ahmet.yilmaz@uni.edu.tr` | `ahmet.yilmaz@uni.edu.tr` |
| Öğrenci | `zeynep.sahin@uni.edu.tr` | `zeynep.sahin@uni.edu.tr` |

> Provisioned kullanıcıların şifresi **e-posta adreslerinin aynısıdır**. Seed,
> demo hesaplarında "ilk girişte şifre değiştir" bayrağını kapatır, böylece
> giriş kesintisiz olur. Tüm demo hesapları
> [seed/data/](new-backend/infrastructure/seed/data/) altında — düzenleyip
> `docker compose up -d --build seed` ile yeniden çalıştırabilirsin.

> Örnek ders kataloğunu genişletmek istersen elle de yükleyebilirsin:
> `docker compose exec -T postgres psql -U postgres -d mydreamcampus < ../monolith/seed_courses.sql`

---

## Günlük komutlar

Hepsi **repo kökünden** çalışır — dizin değiştirmene gerek yok:

```bash
make deploy-ps       # durum
make deploy-logs     # canlı log (monolith + caddy)
make deploy-update   # git pull + değişenleri derle + yeniden başlat
make deploy-down     # durdur (veriyi korur — volume'lar kalır)
make clean           # DİKKAT: veriyi de siler
```

Tek bir servise müdahale gerekirse compose'a doğrudan da geçebilirsin:

```bash
docker compose -f new-backend/infrastructure/docker-compose.yml restart monolith
docker compose -f new-backend/infrastructure/docker-compose.yml logs migrate
```

---

## Sorun giderme

| Belirti | Sebep / çözüm |
|---|---|
| Her komutta sudo şifresi soruyor | `sudo usermod -aG docker $USER` + yeniden giriş (adım A1). Makefile sonrasında sudo'yu tamamen atlar. |
| `permission denied ... docker.sock` | Gruba eklendin ama oturum tazelenmedi: `newgrp docker` ya da çıkış/giriş. |
| Ev ağındaki telefondan açılmıyor | `PUBLIC_HOST` `localhost` kalmış olabilir — LAN IP olmalı. Ayrıca `sudo ufw allow 80/tcp`. |
| `monolith` sürekli restart | `make deploy-logs` → genelde `.env`'de eksik/default secret. Düzelt, `make deploy`. |
| Sertifika uyarısı | Caddy henüz cert almadı: `logs caddy`. 80/443 firewall'da açık mı? `PUBLIC_HOST` gerçekten IP'ye çözülüyor mu (`dig 203-0-113-5.sslip.io`)? |
| `migrate` exit code ≠ 0 | `logs migrate`. DB henüz hazır değilse tekrar: `docker compose up -d migrate`. |
| Login 500 / CORS | `.env`'de `PUBLIC_ORIGIN` tam `https://<host>` mi (sonda `/` yok)? |
| Build OOM (2GB) | Adım 4b swap ekle veya droplet'i 4GB'a resize et. |

## Domain alınca (opsiyonel, sonra)

`.env`'de `PUBLIC_HOST` ve `PUBLIC_ORIGIN`'i gerçek domain'e çevir, DNS A
kaydını droplet IP'sine yönlendir, `docker compose up -d caddy`. Caddy yeni
domain için otomatik sertifika alır.
