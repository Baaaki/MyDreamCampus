# Deploy Rehberi

Proje tek bir makinede `docker compose` ile **tek komutta** ayağa kalkar:
Caddy (SPA + `/api` proxy) → on servis → Postgres/Redis/RabbitMQ.

### Konteyner listesi (16)

| Grup | Konteynerler |
|---|---|
| Edge | `mydreamcampus-caddy` (:80, standalone'da :443) |
| İş servisleri | `mydreamcampus-{auth,staff,student,catalog,enrollment,attendance,grades,meal,payment}` — 8081-8089, host'a publish **edilmez** |
| Worker | `mydreamcampus-notification` — RabbitMQ tüketicisi, HTTP route'u yok |
| Altyapı | `mydreamcampus-{postgres,rabbitmq,redis,mailhog}` |
| Tek seferlik | `mydreamcampus-migrate`, `mydreamcampus-seed` — işi bitince `exited (0)` kalır, bu normaldir |

Postgres tek konteyner ama **servis başına ayrı veritabanı + ayrı rol**
(`auth` DB'si ↔ `auth_svc`). Ayrı bir notification veritabanı konteyneri yoktur.

**Kaynak beklentisi:** imajlar toplam ~650 MB, çalışırken ölçülen bellek
~400 MB (compose'daki `mem_limit` toplamı ~2.4 GB tavan). 2 GB RAM'li bir
makinede swap ile çalışır; 4 GB rahat eder.

> **Ayrı bir "frontend sunucusu" yok.** SPA `bun run build` ile statik dosyalara
> derlenip Caddy imajının içine kopyalanıyor ([frontend/Dockerfile](frontend/Dockerfile)).
> Caddy 80/443'ü dinler: `/api/<prefix>` path-prefix ile ilgili servise, geri
> kalan her şey SPA. Tarayıcı tek origin görür — CORS yok, ayrı port yok.

İki senaryo var:

| | **A. Ev sunucusu** | **B. Public VPS** |
|---|---|---|
| Erişim | Ev ağındaki cihazlar; **A8** ile internete de açılır | İnternetten herkes |
| Adres | `http://192.168.1.50:8080` (tünelle: `https://<domain>`) | `https://203-0-113-5.sslip.io` |
| HTTPS | LAN'da yok (özel IP'ye sertifika verilmez); tünelde Cloudflare alır | Let's Encrypt, Caddy alır |
| Docker | Rootless — root yetkisi gerekmez | Root daemon (droplet'te zaten root'sun) |
| Deploy | `make deploy` | `make deploy` |
| Kurulum | Aşağıdaki **A** bölümü | **B** bölümü |

Ev bağlantısı CGNAT arkasındaysa port yönlendirme çalışmaz — o durumda A + A8
(Cloudflare Tunnel) B'nin yerini tutar ve VPS kirası gerekmez.

### Compose dosyaları — hangisi ne zaman

| Dosya | Ne yapar |
|---|---|
| `docker-compose.yml` | Temel stack. Host'ta **sadece** Caddy'nin `:80`'ini publish eder. |
| `docker-compose.standalone.yml` | Caddy'nin `:443`'ünü + infra portlarını (`127.0.0.1:5432`, `6379`, `15672`, `8025`…) ekler. |

`make deploy` ikisini birlikte yükler, elle bir şey yapman gerekmez. Stack'in
önüne bir edge koyarsan (Cloudflare Tunnel) sadece temel dosya da yeter — o
edge host'un `:80/:443`'ünü tutuyorsa Caddy'nin `:443`'ü publish etmesi port
çakışması yaratır.

---

## A. Ev sunucusu — tek komut

### A1. Sunucuda Docker — root yetkisi vermeden

Makefile sudo'ya **sadece** docker soketine doğrudan erişemediğinde başvurur:

```make
SUDO := $(shell docker info >/dev/null 2>&1 || echo sudo)
```

Yani aşağıdaki kurulumdan sonra hiçbir `make` komutu şifre sormaz — `ssh sunucu
'cd mydreamcampus && make deploy'` gibi tty'siz uzaktan komutlar da çalışır.

**Rootless Docker (önerilen).** Daemon senin kullanıcın olarak çalışır;
container'daki root, host'ta yetkisiz bir UID'ye map'lenir. Kalıcı root
yetkisi **yok**:

```bash
sudo apt install -y uidmap dbus-user-session   # tek seferlik, paket kurulumu
curl -fsSL https://get.docker.com/rootless | sh

echo 'export PATH=$HOME/bin:$PATH' >> ~/.bashrc
echo 'export DOCKER_HOST=unix:///run/user/'$(id -u)'/docker.sock' >> ~/.bashrc
source ~/.bashrc

systemctl --user enable --now docker
sudo loginctl enable-linger $USER    # SSH oturumu kapanınca daemon ölmesin

docker info >/dev/null && echo "rootless calisiyor, sudo gerekmiyor"
```

Rootless daemon 1024'ün altındaki portlara bağlanamaz, o yüzden `.env`'de
yüksek port kullan (adım A3'te):

```
HTTP_PORT=8080
HTTPS_PORT=8443
PUBLIC_HOST=:80
PUBLIC_ORIGIN=http://192.168.1.50:8080
```

Adres `http://192.168.1.50:8080` olur. Portsuz sade bir URL istersen tek
seferlik şu capability yeter (kullanıcıya root vermez, sadece o binary'ye port
bağlama izni tanır) — sonra `.env`'deki `HTTP_PORT` satırlarını sil:

```bash
sudo setcap cap_net_bind_service=ep $(which rootlesskit)
systemctl --user restart docker
```

<details>
<summary><b>Alternatif:</b> klasik (root) daemon + <code>docker</code> grubu — <b>önerilmez</b></summary>

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER && newgrp docker
```

`docker` grubu pratikte root yetkisine denktir: gruba dahil olan herkes host
diskini bir container'a mount edip root olabilir. Aynı sebeple `sudoers`'a
NOPASSWD ile `docker` eklemek de güvenlik kazancı sağlamaz — riski sadece
gizler. Rootless varken bunu tercih etme.
</details>

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
Adres satırları A1'de seçtiğin kuruluma göre — IP'yi kendi LAN IP'nle değiştir:

**Rootless, yüksek port (varsayılan öneri):**

```
HTTP_PORT=8080
HTTPS_PORT=8443
PUBLIC_HOST=:80
PUBLIC_ORIGIN=http://192.168.1.50:8080
```

**Port 80 kullanabiliyorsan (setcap yaptıysan ya da root daemon):**

```
PUBLIC_HOST=http://192.168.1.50
PUBLIC_ORIGIN=http://192.168.1.50
```

> `PUBLIC_HOST` Caddy'nin **container içinde** dinlediği adres, `PUBLIC_ORIGIN`
> ise **tarayıcının gördüğü** tam URL (CORS + e-posta linkleri buradan üretilir),
> sonda `/` olmadan. Yüksek port kullanırken ikisi bilerek farklı: port
> yönlendirmesini compose yapar, Caddy içeride hep 80'i dinler.
>
> `http://` öneki kritik: Caddy şemayı açıkça görünce otomatik HTTPS'i kapatır.
> Öneksiz bırakırsan Let's Encrypt'ten özel IP için sertifika almaya çalışır ve
> başarısız olur. `localhost` da yazma — o sadece sunucunun kendisi demek,
> telefonundan/laptop'undan erişemezsin.

### A4. Başlat — **tek komut, repo kökünden**

```bash
make deploy
```

Bu komut her şeyi yapar: frontend'i derler, on servisin binary'lerini derler,
infra'yı kaldırır, migration'ları uygular, servisleri + Caddy'yi başlatır ve
demo veriyi seed eder. İlk sefer 5-10 dakika (sonrakiler ~30 sn).

Sunucuda dizin değiştirmene gerek yok; hepsi kökten:

```bash
make deploy-ps       # durum
make deploy-logs     # canlı log (caddy + auth + catalog)
make deploy-update   # git pull + değişenleri derle + yeniden başlat
make deploy-down     # durdur (veri korunur)
```

**Tek servisi güncellemek** — diğer 15 konteynere dokunmadan:

```bash
make deploy-grades   # o servisi rebuild + restart et
make logs-grades     # tek servisin logu
make restart-grades  # sadece yeniden başlat
```

### A5. Firewall (ufw kuruluysa)

`.env`'deki `HTTP_PORT` neyse onu aç:

```bash
sudo ufw allow 8080/tcp     # rootless varsayilani; port 80 kullaniyorsan: 80/tcp
```

### A6. Doğrula

Ev ağındaki **herhangi bir cihazdan** tarayıcı: `PUBLIC_ORIGIN`'e yazdığın adres
— rootless kurulumda `http://192.168.1.50:8080`, port 80 kullanıyorsan
`http://192.168.1.50`.

Giriş bilgileri için aşağıdaki [demo hesaplar](#demo-giriş-bilgileri) tablosuna bak.

### A7. Sunucu yeniden başlarsa

Bir şey yapman gerekmez — tüm servisler `restart: unless-stopped` ile
tanımlı, Docker daemon boot'ta kalkınca stack de kalkar. `make deploy` yalnızca
ilk kurulumda ve kod güncellemesinde gerekir.

### Ek: Mobil (Expo) uygulamayı bağlamak

Telefon da tarayıcı gibi **Caddy'ye** bağlanır — servislerin portları host'a
hiç açılmaz, ve zaten tek bir API adresi yok: on servisin her biri kendi
`/api/<önek>`'inde yaşıyor, onları tek adres altında birleştiren şey Caddy.

`mobile/.env` içinde API adresini `PUBLIC_ORIGIN` ile aynı yap — rootless
kurulumda `http://192.168.1.50:8080`, port 80 kullanıyorsan
`http://192.168.1.50`.

> `PUBLIC_HOST`'ta `http://` öneki **şart**: şemasız bırakılırsa Caddy otomatik
> HTTPS'e geçer ve :80'e gelen her isteğe özel IP için alınamayan bir
> sertifikaya 308 döner. Telefonda bu "ağ hatası" olarak görünür.

### A8. Cloudflare Tunnel ile internete açmak

Ev bağlantılarında port yönlendirme çoğu zaman çalışmaz (ISP CGNAT arkasına
alır, 80/443 kapalı olabilir). **Cloudflare Tunnel** giden bağlantı kurar:
router'da port açmaz, gerçek HTTPS ve domain verir.

Ön koşul: Cloudflare'de yönetilen bir domain (nameserver'ları Cloudflare'e
taşınmış olmalı — ücretsiz plan yeterli).

#### 1. cloudflared kur ve giriş yap

```bash
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg \
  | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null
echo "deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main" \
  | sudo tee /etc/apt/sources.list.d/cloudflared.list
sudo apt update && sudo apt install -y cloudflared

cloudflared tunnel login      # tarayicida domain'i sec
```

#### 2. Tunnel oluştur

```bash
cloudflared tunnel create homelab
cloudflared tunnel list        # ID'yi not al
```

#### 3. Ingress kurallarını yaz

`~/.cloudflared/config.yml` — **routing burada yapılıyor**, her proje için bir
hostname:

```yaml
tunnel: homelab
credentials-file: /home/KULLANICI/.cloudflared/<TUNNEL_ID>.json

ingress:
  - hostname: campus.example.com
    service: http://localhost:8080      # mydreamcampus (HTTP_PORT)
  - hostname: proje2.example.com
    service: http://localhost:8081
  # Zorunlu: eslesmeyen her istek icin catch-all, en sonda olmali.
  - service: http_status:404
```

#### 4. DNS kaydını aç ve servisi başlat

```bash
cloudflared tunnel route dns homelab campus.example.com

sudo cloudflared service install     # boot'ta otomatik baslar
sudo systemctl status cloudflared
```

#### 5. `.env`'i tunnel'a göre ayarla

```
HTTP_PORT=8080
HTTPS_PORT=8443
PUBLIC_HOST=:80
PUBLIC_ORIGIN=https://campus.example.com
```

`make deploy` ile uygula. Artık `https://campus.example.com` çalışıyor.

> **`PUBLIC_HOST=:80` neden?** Tunnel isteği `Host: campus.example.com` ile
> iletir. Caddy'ye sabit bir hostname yazarsan eşleşmeyen her istek 404 döner;
> `:80` "hangi host olursa olsun 80'de cevap ver" demektir — host doğrulamasını
> zaten Cloudflare yapıyor.
>
> **HTTPS'i Cloudflare sonlandırır**, Caddy'ye düz HTTP gelir. Bu yüzden Caddy
> tarafında sertifika ayarı gerekmez. `PUBLIC_ORIGIN` yine de `https://` olmalı
> — tarayıcının gördüğü şema o, CORS ve e-posta linkleri oradan üretiliyor.
>
> `HTTPS_PORT` bu senaryoda kullanılmıyor (TLS Cloudflare'de bitiyor), ama
> compose onu publish ettiği için diğer projelerle çakışmayan bir değer ver.

---

## Aynı sunucuda birden fazla proje

**Ayrı bir reverse proxy kurmana gerek yok.** Tunnel'ın `ingress` bloğu zaten
hostname → port yönlendirmesi yapıyor. Her projeye farklı bir host portu ver,
ingress'e bir satır ekle:

```
~/mydreamcampus/   HTTP_PORT=8080  →  campus.example.com
~/proje2/          HTTP_PORT=8081  →  proje2.example.com
~/proje3/          HTTP_PORT=8082  →  proje3.example.com
```

Projeler birbirini tanımaz; tek ortak nokta port numaralarının çakışmaması.

### nginx'i ne zaman araya koymalı

Tunnel ingress **şunları yapamaz**: aynı hostname altında path bazlı bölme
(`/app1`, `/app2`), merkezi rate limit, tek yerde erişim logu, tunnel'dan
bağımsız LAN erişimi. Bunlardan birine ihtiyacın olduğunda araya bir edge proxy
koy:

```
Tunnel → nginx :80 → :8080 mydreamcampus
                   → :8081 proje 2
```

O zaman ingress'te tek kural kalır (`service: http://localhost:80`) ve dağıtımı
nginx yapar. Sonradan geçmek kolay, baştan kurmak gereksiz.

**nginx bu repoda olmamalı** — ortak altyapı, ayrı bir dizin/repo'da dursun
(`~/infra/`). Aksi halde MyDreamCampus'a her push, otomatik deploy sırasında
tüm projelerin router'ını yeniden başlatır ve `make deploy-down` bütün siteleri
kapatır.

---

## Otomatik deploy — push'ta sunucu kendini güncellesin

`git pull` tek başına yetmez: Go kodu binary'ye, React kodu statik dosyalara
**imajın içine** derleniyor. Kaynağı güncellemek çalışan container'ı değiştirmez;
yeniden build + restart gerekir.

Bunu otomatikleştirmek için sunucu `origin/main`'i yoklar ve hareket edince
kendini günceller:

```bash
make autodeploy-install
```

Kurulan şey bir **systemd user timer**: 2 dakikada bir
[scripts/auto-deploy.sh](scripts/auto-deploy.sh) çalışır, yeni commit yoksa
hiçbir şey yapmadan çıkar. Varsa önce GitHub API'den o commit'in `ci-passed`
kontrolüne bakar; yalnız başarılıysa `git pull --ff-only` + `make deploy`
yapar. CI sürüyorsa ya da kırmızıysa deploy etmez, sonraki turda yeniden
bakar.

Sunucuda `jq` kurulu olmalı (`sudo apt install jq`); yoksa script CI'ın
cevabını okuyamaz ve deploy etmeden hata verir.

```bash
make autodeploy-status   # sonraki kontrol ne zaman, son sonuç ne
make autodeploy-logs     # deploy geçmişi (journalctl)
make autodeploy-now      # timer'ı bekleme, hemen kontrol et
make autodeploy-off      # kapat
```

Artık akış şu: kendi bilgisayarında `git push` → en geç 2 dakika içinde sunucu
kendini günceller. Elle hiçbir komut yok.

### Neden webhook değil de yoklama?

Ev sunucusu NAT arkasında; GitHub Actions oraya SSH ile **bağlanamaz** —
internette bir sunucu için standart olan "CI bitince sunucuya deploy komutu
gönder" yaklaşımı burada çalışmaz. Yoklama ters yönde çalışır: sunucunun
GitHub'a bağlanması her zaman mümkün.

Cloudflare Tunnel kurulu olsa bile yoklama tercih edilir:

| | Yoklama (kurulan bu) | Webhook (tunnel üzerinden) |
|---|---|---|
| Gecikme | ≤ 2 dakika | Anında |
| Saldırı yüzeyi | Yok — dışarıdan tetiklenemez | İnternete açık bir deploy endpoint'i |
| Gereken secret | Yok | HMAC imza doğrulaması şart |
| Tunnel çökerse | Çalışmaya devam eder | Deploy durur |

Portfolio/demo için 2 dakikalık gecikme sorun değil, o yüzden basit ve güvenli
olan seçildi. Anlık deploy'a ihtiyaç olursa tunnel zaten kurulu — webhook
listener'ı sonradan eklemek mümkün.

### Notlar

- **Linger şart.** `sudo loginctl enable-linger $USER` yapılmadıysa timer
  yalnızca sen SSH ile bağlıyken çalışır (adım A1).
- **Sunucuda elle commit atma.** Script `--ff-only` kullanır; sunucudaki local
  değişiklik deploy'u sessizce bozmak yerine yüksek sesle patlatır.
- **`.env` güvende.** `.gitignore`'da ve takipli değil — `git pull` sunucudaki
  secret'larını ezmez.
- **Başarısız build tekrar denenir.** Script en son *başarıyla* deploy edilen
  SHA'yı `.git/last-deployed-sha` içinde tutar, `HEAD`'i değil. Build patlarsa
  sonraki turda aynı commit tekrar denenir.
- **Private repo ise** sunucuya read-only bir GitHub *deploy key* ekle
  (`ssh-keygen -t ed25519` → public key'i repo → Settings → Deploy keys).
  CI kontrolü için de token gerekir: `repo` scope'lu bir classic token üret
  ve `systemctl --user edit mydreamcampus-deploy.service` ile
  `[Service]` altına `Environment=GITHUB_TOKEN=<token>` yaz. Bu override
  `make autodeploy-install` tekrar çalışınca silinmez.
- **Deploy neden olmuyor?** `make autodeploy-logs`: `waiting for CI` CI'ın
  bitmesini, `not deploying, ci-passed concluded failure` kırmızı bir CI'ı
  gösterir. İkisinde de sunucu bir önceki sürümde çalışmaya devam eder.

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

Doldurulacaklar: `POSTGRES_PASSWORD`, `SERVICE_DB_PASSWORD`, `REDIS_PASSWORD`, `RABBITMQ_PASSWORD`,
`JWT_SECRET`, `INTERNAL_SERVICE_SECRET`, `QR_SECRET`, `ADMIN_INITIAL_PASSWORD`,
`PUBLIC_HOST`, `PUBLIC_ORIGIN`.

> On servisin hepsi `ENVIRONMENT=production` ile çalışır ve secret'lar default
> kalırsa **başlamayı reddeder** — bu bilinçli bir güvenlik önlemi.

### 4b. (Sadece 2GB droplet'te) swap ekle — build OOM olmasın

```bash
fallocate -l 2G /swapfile && chmod 600 /swapfile
mkswap /swapfile && swapon /swapfile
echo '/swapfile none swap sw 0 0' >> /etc/fstab
```

---

## 5. Build + ayağa kaldır

```bash
cd ~/mydreamcampus       # repo köküne dön
make deploy              # ilk sefer 3-6 dk (Go + frontend derlenir)
```

> Çıplak `docker compose up -d` **çalıştırma**: `-f` vermeden çağırdığında
> standalone overlay'i yüklemez, Caddy `:443`'ü açmaz ve HTTPS gelmez.
> `make deploy` doğru dosya setini kendisi veriyor.

Sıra otomatik: infra sağlıklı olunca `migrate` çalışır → bitince on servis
başlar → `seed` demo veriyi yazar → `caddy` TLS sertifikasını çeker.

Migration loglarını gör:

```bash
make deploy-ps                        # 14 "running", migrate + seed "exited (0)"
make deploy-logs                      # caddy + auth + catalog, canlı
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
- **`seed` servisi** auth/staff/student/catalog ayağa kalkınca otomatik çalışır;
  öğretmen, ders, öğrenci ve profilleri **gerçek admin API üzerinden** oluşturur
  (event zinciri düzgün dolsun diye — ham SQL değil), fakülte/bölüm listesini
  ve senaryo verisini (dönem, program, not, yoklama, menü, rezervasyon) servis
  veritabanlarına SQL ile yazar. Tarihler seed gününe göredir: aktif dönem o
  günün dönemidir (`YYYY-YYYY-Fall|Spring`), periyotlar o gün açıktır.
- Seed **yalnız boş sistemde** koşar: öğrenci kaydı varsa hiçbir şey yazmadan
  çıkar. Bir API hatasında durur ve `exited (1)` kalır — log hatayı gösterir.
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
> seed'lediği hesaplarda "ilk girişte şifre değiştir" bayrağını kapatır, böylece
> giriş kesintisiz olur. Tüm demo içeriği
> [seed/data/](new-backend/infrastructure/seed/data/) altında.

> Seed içeriğini değiştirmek (örn.
> [seed/data/courses.json](new-backend/infrastructure/seed/data/courses.json))
> yalnız **temiz bir kurulumda** etkili olur — veri dolu bir sistemde seed
> atlanır. Çalışan sisteme ders, öğretmen ya da öğrenci eklemek için admin
> arayüzünü kullan: kayıt API'den geçer ve tetiklediği olaylar diğer
> servislerin projeksiyonlarına da düşer. Doğrudan `psql` ile yazılan satır o
> zincirin dışında kalır. Seed'i baştan koşturmak volume'ları siler:
> `docker compose down -v && make deploy`.

---

## C. Canlı Demo Kurulumu (Ev Sunucusu + Cloudflare Tunnel)

Bu senaryoda proje, kullanıcının ev sunucusunda çalışır ve Cloudflare Tunnel
aracılığıyla dış dünyaya (`https://mydreamcampus.madebybaki.com`) açılır.

- **Dışarıya açık port yok:** Caddy'nin host portları iptal edilir (`docker-compose.tunnel.yml`).
  İstekler Cloudflare'in sunucuya kurduğu tünel konteyneri (`cloudflared`) üzerinden
  dahili Docker ağıyla Caddy'ye (`http://caddy:80`) ulaşır.
- **Güvenli ve İzole:** Statik varlıklar Cloudflare tarafından önbelleğe alınır,
  yönetim uçları Cloudflare Access ile korunur, gece 04:00'te sistem otomatik olarak
  süper adminin kaydettiği kalıcı duruma döner.

---

### Adım Adım Kurulum

#### 1. Repoyu al
```bash
git clone <REPO_URL> mydreamcampus
cd mydreamcampus
```

#### 2. Ortam Değişkenlerini (`.env`) Doldur
```bash
cp new-backend/infrastructure/.env.example new-backend/infrastructure/.env
nano new-backend/infrastructure/.env
```
Gereken temel değerler:
- Tüm secret'ları `openssl rand -base64 48` ile üret (`SERVICE_DB_PASSWORD` için `openssl rand -hex 32`).
- `ADMIN_EMAIL`: Tahmin edilemez gerçek süper admin e-postası.
- `ADMIN_INITIAL_PASSWORD`: Güçlü geçici ilk şifre (ilk girişte değiştirilmesi zorunludur).
- `DEMO_MODE=true`
- `SEED_DEMO=true`
- `DEMO_ADMIN_EMAIL=demo.admin@mydreamcampus.com`
- `DEMO_TEACHER_EMAIL=ahmet.yilmaz@uni.edu.tr`
- `DEMO_STUDENT_EMAIL=zeynep.sahin@uni.edu.tr`
- `NIGHTLY_RESET_AT=04:00`
- `BASELINE_KEEP=7`
- `EDGE=tunnel`: Her `make` hedefi ve otomatik deploy bunu `.env`'den okur. Yazmazsan elle
  çalıştırılan bir `make deploy` standalone katmanına döner ve `cloudflared`'ı kaldırır.
- `TUNNEL_TOKEN`: Cloudflare Zero Trust panelinden alınan tünel token'ı (aşağıdaki Cloudflare adımlarına bak).
- `PUBLIC_HOST=:80`
- `PUBLIC_ORIGIN=https://mydreamcampus.madebybaki.com`

> **Alternatif (Host Seviyesinde Cloudflared):** Eğer sunucuda zaten Docker dışında
> bağımsız çalışan bir `cloudflared` varsa, stack'i `EDGE=tunnel` ile başlatmak yerine
> normal `make deploy` ile başlatabilirsin. Bu durumda `.env`'de `HTTP_PORT=8080` tanımlanır,
> Caddy host'ta `127.0.0.1:8080` dinler ve host'taki `~/.cloudflared/config.yml` ingress kuralına
> `service: http://127.0.0.1:8080` yazılır.

#### 3. Stack'i Başlat
```bash
make deploy
```
`.env`'deki `EDGE=tunnel` sayesinde bu komut `docker-compose.yml` ve `docker-compose.tunnel.yml`
katmanlarını birlikte yükler.
Caddy'nin host portları kalkar, `cloudflared` konteyneri ayağa kalkıp tüneli kurar.

#### 4. Seed ve İlk Kalıcı Durumu Doğrula
```bash
# Seed işlemini kontrol et
docker compose logs seed | grep "seed complete"

# demo-ops konteynerinin ilk kalıcı durumu kaydettiğini kontrol et
docker compose logs demo-ops | grep "baseline"
```

#### 5. Süper Admin İlk Girişi ve Doğrulama
1. Tarayıcıda `https://mydreamcampus.madebybaki.com` adresini aç.
2. `ADMIN_EMAIL` ve `ADMIN_INITIAL_PASSWORD` ile giriş yap.
3. Sunucu seni otomatik olarak `/change-password` ekranına yönlendirecektir; güçlü yeni şifreni belirle.
4. `/system/baseline` ("Kalıcı Veri") sayfasına git ve ilk kalıcı durumun (v1) başarıyla listelendiğini gör.

#### 6. Otomatik Deploy'u Kur
```bash
make autodeploy-install
```
`origin/main` dalını 2 dakikada bir kontrol eden systemd user timer'ı kurulur.
Yeni commit geldiğinde GitHub API üzerinden `ci-passed` kontrolünün başarılı olduğunu
doğrular ve yalnızca CI'dan geçmiş sürümleri deploy eder; katmanı yine `.env`'deki `EDGE` belirler.

---

### Cloudflare ve Yayın Adımları

1. **Cloudflare Tunnel (Zero Trust):**
   - Cloudflare One (Zero Trust) -> Networks -> Tunnels -> **Create a tunnel**.
   - Connector tipi olarak **Cloudflared / Docker** seç.
   - Verilen komuttaki `--token <TOKEN>` değerini kopyalayıp sunucudaki `.env` dosyasındaki `TUNNEL_TOKEN` alanına yapıştır.
   - **Public Hostname:**
     - Subdomain / Domain: `mydreamcampus.madebybaki.com`
     - Service Type: `HTTP`
     - URL: `caddy:80` (DİKKAT: localhost değil! cloudflared konteyneri Caddy'ye Docker ağı üzerinden `caddy` adıyla bağlanır).

2. **SSL / TLS Ayarları:**
   - Cloudflare Dashboard -> SSL/TLS -> Edge Certificates.
   - **"Always Use HTTPS"** özelliğini aktif et.

3. **WAF & Rate Limiting:**
   - Security -> WAF -> Rate limiting rules -> **Create rule**:
     - Kural Adı: `Rate limit login`
     - If incoming requests match: `URI Path equals /api/auth/login`
     - Rate: `10 requests per 1 minute` (IP başına).
     - Action: `Block` (veya Managed Challenge), Süre: `10 minutes`.

4. **Kalıcı Veri Uçlarını Koruma (K4 Kararı — Cloudflare Access):**
   - Zero Trust -> Access -> Applications -> **Add an Application** -> **Self-hosted**:
     - Application Name: `MyDreamCampus Baseline Ops`
     - Application Domain: `mydreamcampus.madebybaki.com`
     - Path: `/api/catalog/admin/ops*`
     - Policy: Rule name: `Superadmin Only`, Action: `Allow`.
     - Include Selector: `Emails` -> Kullanıcının kendi şahsi e-posta adresi (One-time PIN ile doğrulanır).
   - Böylece kalıcı veri alma ve düzenleme moduna geçme uçları internetten gelebilecek yetkisiz taramalara karşı Cloudflare seviyesinde korunur.

5. **GitHub Dal Koruması:**
   - GitHub Repository -> Settings -> Branches -> Add branch protection rule (`main`):
     - `Require a pull request before merging`
     - `Require status checks to pass before merging` -> `ci-passed` seç.

6. **Uptime İzleme (UptimeRobot - Ücretsiz):**
   - Yeni monitor ekle: Type `HTTP(s)`, URL `https://mydreamcampus.madebybaki.com/health`, Interval `5 minutes`.
   - Caddy `/health` isteğini `auth-service`'e iletir; auth-service Postgres, Redis ve RabbitMQ sağlığını doğrulayarak yanıt verir.

7. **Mobil Yayın (EAS Build Preview):**
   ```bash
   cd mobile
   npx eas-cli login
   npx eas-cli init    # Expo projesini oluşturur, app.json'a extra.eas.projectId yazar
   npx eas-cli build -p android --profile preview
   ```
   `app.json`'da yer tutucu bir `projectId` bırakma: `eas init` var olan bir ID'yi
   "zaten bağlı" sayıp yenisini yazmaz, build de olmayan projeyi arar.
   `eas init`'in yazdığı `extra.eas` bloğunu commit'le.
   EAS tarafından üretilen APK indirme bağlantısını projenin dokümantasyonuna veya README'sine ekleyebilirsin.

8. **(İsteğe bağlı) Sunucu Dışı Yedek:**
   Sunucuya `rclone` kurup harici bir bulut depolama (Google Drive, AWS S3 vb.) bağla.
   `.env` dosyasında `BASELINE_OFFSITE_CMD` değişkenine yedekleme komutunu gir:
   ```bash
   BASELINE_OFFSITE_CMD="rclone copy /baselines remote:mydreamcampus-backups"
   ```
   Her yeni kalıcı durum alındığında `demo-ops` bu komutu otomatik çalıştıracaktır.

---

## Günlük komutlar

Hepsi **repo kökünden** çalışır — dizin değiştirmene gerek yok:

```bash
make deploy-ps       # durum
make deploy-logs     # canlı log (caddy + auth + catalog)
make deploy-update   # git pull + değişenleri derle + yeniden başlat
make deploy-down     # durdur (veriyi korur — volume'lar kalır)
make clean           # DİKKAT: veriyi de siler

make deploy-grades   # TEK servisi rebuild + restart (diğer 15'e dokunmaz)
make logs-grades     # tek servisin logu
make restart-grades  # sadece yeniden başlat
```

Tek bir servise müdahale gerekirse compose'a doğrudan da geçebilirsin. Stack iki
dosyaya bölündüğü için ikisini de vermen gerekir; `COMPOSE_FILE` (compose'un
kendi değişkeni, `:` ile ayrılır) bunu bir kez ayarlamanı sağlar:

```bash
cd ~/mydreamcampus
export COMPOSE_FILE=new-backend/infrastructure/docker-compose.yml:new-backend/infrastructure/docker-compose.standalone.yml

docker compose restart grades-service
docker compose logs migrate
docker compose logs seed
docker compose up -d --build seed

# Servis başına ayrı veritabanı: psql'e HANGİ veritabanı olduğunu söylemen
# gerekiyor. Süperuser olarak hepsine bağlanabilirsin.
docker compose exec -T postgres psql -U postgres -d catalog -c '\dt course_catalog.*'
docker compose exec -T postgres psql -U postgres -c '\l'   # veritabanlarını listele
```

Kalıcı olsun istersen `~/.bashrc`'ye ekle.

---

## Sorun giderme

| Belirti | Sebep / çözüm |
|---|---|
| Her komutta sudo şifresi soruyor | Rootless daemon'a erişilemiyor. `echo $DOCKER_HOST` boşsa `source ~/.bashrc`; `systemctl --user status docker` çalışıyor mu? (adım A1) |
| `permission denied ... docker.sock` | Klasik daemon'ın root soketine düşmüşsün. `DOCKER_HOST=unix:///run/user/$(id -u)/docker.sock` ayarlı mı? |
| SSH kapanınca container'lar ölüyor | `sudo loginctl enable-linger $USER` yapılmamış (adım A1). |
| `bind: permission denied` (port 80) | Rootless 1024 altına bağlanamaz: `.env`'de `HTTP_PORT=8080` kullan ya da `setcap` uygula (adım A1). |
| `port is already allocated` | Makinede o portu tutan başka bir şey var. `.env`'de `HTTP_PORT` (veya infra için `POSTGRES_HOST_PORT`, `REDIS_HOST_PORT`…) ile değiştir. |
| Ev ağındaki telefondan açılmıyor | `PUBLIC_ORIGIN` `localhost` kalmış olabilir — LAN IP + port olmalı. Ayrıca `sudo ufw allow <HTTP_PORT>/tcp`. |
| Bir servis sürekli restart | `make logs-<servis>` → genelde `.env`'de eksik/default secret. Düzelt, `make deploy-<servis>`. |
| Sadece bir `/api/<önek>` 502 veriyor | O servis ayakta değil: `make deploy-ps`, sonra `make logs-<servis>`. Diğer önekler etkilenmez — bölünmenin beklenen davranışı. |
| Loglarda `circuit breaker state change ... to: open` | Çağrılan servis düşmüş; onu düzelt. Breaker 30 sn sonra kendini dener, eşiği gevşetme. |
| Sertifika uyarısı | Caddy henüz cert almadı: `logs caddy`. 80/443 firewall'da açık mı? `PUBLIC_HOST` gerçekten IP'ye çözülüyor mu (`dig 203-0-113-5.sslip.io`)? |
| `migrate` exit code ≠ 0 | `logs migrate`. DB henüz hazır değilse tekrar: `docker compose up -d migrate`. |
| `migrate` logunda `cannot open these databases` | Mevcut volume'a sonradan veritabanı eklenmiş (ör. `payment`); `init-databases.sh` yalnız boş volume'da çalışır. Logdaki `PROVISION_ONLY=...` komutunu çalıştır, sonra `make deploy`. |
| Login 500 / CORS | `.env`'de `PUBLIC_ORIGIN` tam `https://<host>` mi (sonda `/` yok)? |
| Build OOM (2GB) | Adım 4b swap ekle veya droplet'i 4GB'a resize et. |
| `rabbitmq` açılmıyor, logda `feature flag` / `incompatible` | Broker yeni bir sürüm serisine, eski sürümde kapalı kalmış feature flag'lerle geçmiş. Aşağıdaki "RabbitMQ sürüm yükseltmesi" bölümüne bak. |
| Tunnel bağlanmıyor | `docker compose logs cloudflared`. `TUNNEL_TOKEN` değerinin `.env`'de doğru olduğunu kontrol et. Cloudflare Zero Trust'ta Public Hostname hedefinin `caddy:80` (HTTP) olarak yazıldığından emin ol (localhost:80 container içinden host'a değil container'ın kendisine bakar). |
| Loglarda gerçek IP görünmüyor | `frontend/Caddyfile`'daki `trusted_proxies` bloğunun Docker ağını (`172.16.0.0/12`) kapsadığından emin ol. Compose ağı `172.28.0.0/16` olarak sabitlenmiştir. |
| Geri dönüş (restore) hatası | `docker compose logs demo-ops`. Veritabanı şifrelerinin `.env` ile uyumunu doğrula. `docker exec mydreamcampus-demo-ops ls -la /baselines` ile geçerli dump dosyalarını kontrol et. |
| Kilit takılı kaldı (503 SYSTEM_EDITING) | Süper admin düzenleme modundayken oturum kapandıysa veya demo-ops zaman aşımından önce çöktüyse: `docker exec mydreamcampus-redis redis-cli -a "$REDIS_PASSWORD" DEL system:editing_lock system:editing_until` çalıştır veya Kalıcı Veri sayfasında "İptal Et" butonuna bas. |


### RabbitMQ sürüm yükseltmesi

RabbitMQ, mevcut veri dizinini yeni bir sürüm serisinde açmadan önce eski
sürümün tüm *stable* feature flag'lerinin açık olmasını ister. Compose'daki
imaj 4.3 ve 4.3 yalnızca 4.2'den yükseltilebiliyor: broker hâlâ 3.13
verisindeyse önce imajı `rabbitmq:4.2-management` yapıp bir kez `make deploy`
çalıştır, sonra 4.3'e dön ve tekrar `make deploy`.

`make deploy` ve `make deploy-update` bunu zaten yapar: `up`'tan önce çalışan
broker'da `rabbitmqctl enable_feature_flag all` çalıştırır (işlem idempotent,
broker yoksa atlanır). Otomatik deploy da `make deploy` kullandığı için ek bir
adım gerekmez.

Compose'u elle çalıştırıyorsan imajı değiştirmeden **önce** kendin yap:

```bash
docker exec mydreamcampus-rabbitmq rabbitmqctl enable_feature_flag all
docker compose up -d rabbitmq
```

Bu adımı atlayıp broker açılmıyorsa: imajı geçici olarak önceki sürüme
(ör. `rabbitmq:4.2-management`) döndür, broker'ı başlat, yukarıdaki komutu
çalıştır, sonra yeni imaja geç. Kuyruktaki mesajlar volume'da durduğu için
bu sırada kaybolmaz.

## Domain alınca (opsiyonel, sonra)

`.env`'de `PUBLIC_HOST` ve `PUBLIC_ORIGIN`'i gerçek domain'e çevir, DNS A
kaydını droplet IP'sine yönlendir, `docker compose up -d caddy`. Caddy yeni
domain için otomatik sertifika alır.
