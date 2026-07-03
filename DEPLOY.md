# DigitalOcean Deploy Rehberi

Bu proje tek bir droplet üzerinde `docker compose` ile çalışır: Caddy (HTTPS +
SPA + `/api` proxy) → monolith → Postgres/Redis/RabbitMQ. Domain almadan
**gerçek HTTPS** için `sslip.io` kullanıyoruz.

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

```bash
docker compose ps                     # durum
docker compose logs -f monolith       # canlı log
docker compose restart monolith       # tek servis yeniden başlat
docker compose down                   # durdur (veriyi korur — volume'lar kalır)
docker compose down -v                # DİKKAT: veriyi de siler
```

## Kod güncelleyince yeniden deploy

```bash
cd ~/mydreamcampus && git pull
cd new-backend/infrastructure
docker compose build monolith caddy   # değişen servisleri derle
docker compose up -d                  # farkı uygula
```

---

## Sorun giderme

| Belirti | Sebep / çözüm |
|---|---|
| `monolith` sürekli restart | `docker compose logs monolith` → genelde `.env`'de eksik/default secret. Düzelt, `up -d`. |
| Sertifika uyarısı | Caddy henüz cert almadı: `logs caddy`. 80/443 firewall'da açık mı? `PUBLIC_HOST` gerçekten IP'ye çözülüyor mu (`dig 203-0-113-5.sslip.io`)? |
| `migrate` exit code ≠ 0 | `logs migrate`. DB henüz hazır değilse tekrar: `docker compose up -d migrate`. |
| Login 500 / CORS | `.env`'de `PUBLIC_ORIGIN` tam `https://<host>` mi (sonda `/` yok)? |
| Build OOM (2GB) | Adım 4b swap ekle veya droplet'i 4GB'a resize et. |

## Domain alınca (opsiyonel, sonra)

`.env`'de `PUBLIC_HOST` ve `PUBLIC_ORIGIN`'i gerçek domain'e çevir, DNS A
kaydını droplet IP'sine yönlendir, `docker compose up -d caddy`. Caddy yeni
domain için otomatik sertifika alır.
