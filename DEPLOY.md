# Deploy Rehberi

Proje tek bir makinede `docker compose` ile **tek komutta** ayağa kalkar:
Caddy (SPA + `/api` proxy) → monolith → Postgres/Redis/RabbitMQ.

> **Ayrı bir "frontend sunucusu" yok.** SPA `bun run build` ile statik dosyalara
> derlenip Caddy imajının içine kopyalanıyor ([frontend/Dockerfile](frontend/Dockerfile)).
> Caddy 80/443'ü dinler: `/api/*` → monolith:8080, geri kalan her şey SPA.
> Tarayıcı tek origin görür — CORS yok, ayrı port yok.

Üç senaryo var:

| | **A. Ev sunucusu (LAN)** | **B. Public VPS** | **C. Openship (PaaS)** |
|---|---|---|---|
| Erişim | Ev ağındaki cihazlar | İnternetten herkes | İnternetten herkes |
| Adres | `http://192.168.1.50:8080` | `https://203-0-113-5.sslip.io` | Openship'in verdiği domain |
| HTTPS | Yok (özel IP'ye sertifika verilmez) | Let's Encrypt, Caddy alır | Let's Encrypt, **Openship** alır |
| Docker | Rootless — root yetkisi gerekmez | Root daemon (droplet'te zaten root'sun) | Openship yönetir |
| Deploy | `make deploy` | `make deploy` | Push → Openship build eder |
| Kurulum | Aşağıdaki **A** bölümü | **B** bölümü | **C** bölümü |

### Compose dosyaları — hangisi ne zaman

| Dosya | Ne yapar |
|---|---|
| `docker-compose.yml` | Temel stack. Host'ta **sadece** Caddy'nin `:80`'ini publish eder. |
| `docker-compose.standalone.yml` | Caddy'nin `:443`'ünü + infra portlarını (`127.0.0.1:5432`, `6379`, `15672`, `8025`…) ekler. |
| `docker-compose.override.yml` | Sadece mobil geliştirme: monolith `:8080`'i LAN'a açar. |

**A ve B'de ikisi de gerekir** — `make deploy` temel + standalone'u birlikte yükler,
elle bir şey yapman gerekmez. **C'de sadece temel dosya** kullanılır: Openship'in
kendi edge proxy'si host'un `:80/:443`'ünü zaten tutuyor, Caddy de onu publish
etmeye kalkarsa deploy port çakışmasından patlar.

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

Telefon Caddy'yi es geçip doğrudan monolith'e gitmek isterse override dosyasıyla
kaldır (base compose 8080'i dışarı açmaz). Üç dosyanın da verilmesi gerekir —
`-f` kullandığın anda compose `override.yml`'ı otomatik yüklemez:

```bash
docker compose \
  -f new-backend/infrastructure/docker-compose.yml \
  -f new-backend/infrastructure/docker-compose.standalone.yml \
  -f new-backend/infrastructure/docker-compose.override.yml up -d
```

Sonra `mobile/.env` içinde API adresini `http://192.168.1.50:8080` yap.

> **Dikkat:** rootless kurulumda Caddy zaten `HTTP_PORT=8080`'de. Override da
> monolith'i 8080'e bağlamaya çalışır ve compose `port is already allocated`
> hatası verir. Birini değiştir — en kolayı `.env`'de `HTTP_PORT=8090` yapıp
> `PUBLIC_ORIGIN`'i de `http://192.168.1.50:8090` olarak güncellemek.

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

## C. Openship (self-hosted PaaS)

[Openship](https://openship.io) repoyu kendisi klonlar, compose servislerini
build eder, container'ları ayağa kaldırır ve kendi **OpenResty edge**'i ile
domain + Let's Encrypt sertifikasını yönetir. Bu senaryoda A/B'deki
`make deploy` ve `.env` dosyası **kullanılmaz** — o işi Openship yapar.

> **Bu bölüm dashboard'dan tamamlanamaz.** Openship'in compose pipeline'ı çalışır
> durumda, ama `docker-compose` framework'ü dashboard'un seçicisinden kasıtlı
> olarak çıkarılmış (`Frameworks.tsx` → `EXCLUDED_STACKS`). Projeyi UI'dan
> açarsan stack tek bir Go/statik uygulama sanılır ve deploy static pipeline'ına
> düşer. Doğru kapı `openship service sync`. Aşağıdaki adımlar Openship v0.4.5
> kaynak kodu incelenerek çıkarıldı.

### C0. Sunucuda bir kerelik izin düzeltmesi

```bash
sudo mkdir -p /opt/openship/static/{releases,.builds}
sudo chown -R $USER:$USER /opt/openship
```

> Openship `/opt/openship/static`'i **uzak sunucuda oluşturmuyor** — repoda bu
> dizini açan ya da sahipliğini veren kod yok, yalnızca docker volume mount'u
> olarak tanımlı. Deploy'un son adımı (`promoteBuildArtifact` → `mkdir`) SSH
> kullanıcısı olarak çalıştığı için stok `/opt` (root:root 0755) altında
> `Permission denied` alır. Compose yoluna geçince bu kod yolu kullanılmaz ama
> ilk denemede static'e düşersen duvara çarpmamak için önden aç.

### C1. Repodaki hazırlık — zaten yapıldı

Kökteki [openship.json](openship.json):

```json
{
  "framework": "docker-compose",
  "rootDirectory": "new-backend/infrastructure",
  "env": { "PUBLIC_HOST": ":80", ... }
}
```

- **`rootDirectory`** — compose dosyası repo kökünde değil, Openship'in nereye
  bakacağı buradan gelir. Overlay bu alanı uyguluyor.
- **`framework`** — **overlay bu alanı uygulamıyor** (`prepare.service.ts`
  içindeki `applyOpenshipOverlay` listesinde yok). Yalnızca detection'ın seçtiği
  dizinde bir `openship.json` varsa metadata fold'undan geçer; bizim compose
  dosyamız kök dışında olduğu için geçmez. Dosyada yine de duruyor çünkü C2'de
  aynı değeri API'ye vereceğiz.
- **`env`** — sadece **yeni import**ta okunur; mevcut bir projeye sonradan
  eklemek DB kaydını değiştirmez.

> Şemanın kökünde `additionalProperties: false` var — tanımsız bir alan
> (eskiden burada `composePath` yazıyordu) dosyanın tamamını geçersiz kılar ve
> Openship sessizce auto-detection'a düşer. Alan adlarını
> [openship.schema.json](https://openship.io/openship.schema.json) ile doğrula.

### C2. Projeyi `services` tipiyle oluştur

```bash
openship project create --name my-dream-campus \
  --git-owner <owner> --git-repo <repo> --git-branch main --type services
```

Proje zaten varsa tipini API'den çevir (CLI'de `update --framework` bayrağı yok):

```bash
curl -X PATCH https://<openship-host>/api/projects/<projectId> \
  -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' \
  -d '{"framework": "docker-compose"}'
```

### C3. Compose servislerini senkronla

```bash
openship service sync new-backend/infrastructure/docker-compose.yml \
  --project <projectId>
```

Bu komut **yerelde** `docker compose config --format json` çalıştırıp sonucu
gönderir; yani `${VAR}` interpolasyonunu Docker Compose çözer, Openship'e somut
değerler gider. Oluşan `kind="compose"` satırları projeyi compose pipeline'ına
sokar.

**Sync sonrası build context'lerini düzelt — zorunlu.** `service sync` mutlak
context'i repo köküne değil **compose dosyasının dizinine** göre relatifleştirir
(`service.ts` → `relativizeContext`) ve `..` segmentlerini temizlemez, dolayısıyla
ürettiği değerler checkout dizininin dışını gösterir.

Ayrıca Openship build fazında **servis başına context kurmuyor**: tek bir ortak
context'i checkout kökünde açıp beşine de veriyor (`Preparing shared build
context...`). `build` alanı yalnızca Dockerfile'ı bulmaya yarıyor
(`<build>/<dockerfile>`). Bu yüzden repo, tüm Dockerfile'ları **repo kökü göreli**
COPY yollarına taşıdı (commit `401397e`) — ayrıntı ve gerekçe:
[OPENSHIP-PROBLEMS.md kusur 7](OPENSHIP-PROBLEMS.md).

Doğru değerler bu yüzden artık şunlar:

| Servis | `build` | `dockerfile` |
|---|---|---|
| `monolith` | `.` | `new-backend/monolith/Dockerfile` |
| `notification` | `.` | `new-backend/services/notification/Dockerfile` |
| `migrate` | `.` | `new-backend/infrastructure/migrate/Dockerfile` |
| `seed` | `.` | `new-backend/infrastructure/seed/Dockerfile` |
| `caddy` | `.` | `frontend/Dockerfile` |

Her biri için:

```bash
curl -X PATCH https://<openship-host>/api/projects/<projectId>/services/<serviceId> \
  -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' \
  -d '{"build": ".", "dockerfile": "new-backend/monolith/Dockerfile"}'
```

Kalan 5 servis (`postgres`, `notification-postgres`, `rabbitmq`, `redis`,
`mailhog`) hazır imaj kullanıyor, `build` alanları yok — dokunma.

> **MCP üzerinden gidiyorsan:** `POST /projects/:id/services/sync` ve
> `GET /projects/:id/services` MCP sunucusunda `service '*' not found` ile patlıyor
> (kusur 11). Servisler zaten kayıtlıysa sync'e gerek yok — her birini
> `PATCH .../services/<serviceId>` ile güncelle. Servis ID'lerini
> `GET /projects/:id/services/containers` verir.

### C4. Sadece `caddy`'yi dışa aç

```bash
openship service update caddy --expose --exposed-port 80 --domain <label>
```

`exposed` varsayılanı `false`, diğer servislere dokunmana gerek yok.

> **`caddy`'yi public işaretlemek zorunlu.** Openship yalnızca route ettiği
> servisin portunu `127.0.0.1:<pinned>:80` olarak yeniden bağlar; işaretlemezsen
> compose'daki `80:80` olduğu gibi host'a publish edilir ve Openship'in kendi
> edge'i (host network'te, `:80`/`:443`) ile çakışır.

### C5. Environment variables

Dashboard → **Environment**. Secret'ları doldur — her biri için ayrı
`openssl rand -base64 48`:

```
POSTGRES_PASSWORD  REDIS_PASSWORD  RABBITMQ_PASSWORD
JWT_SECRET  INTERNAL_SERVICE_SECRET  QR_SECRET  ADMIN_INITIAL_PASSWORD
```

Bir de domain'e bağlı olan tek değişken:

```
PUBLIC_ORIGIN=https://campus.example.com     # sonda / YOK
```

> `PUBLIC_ORIGIN` tarayıcının gördüğü tam URL — CORS ve e-posta linkleri buradan
> üretiliyor. `PUBLIC_HOST` ise `:80` olarak sabit: "hangi Host header gelirse
> gelsin 80'de cevap ver". Host doğrulamasını ve TLS'i zaten Openship'in edge'i
> yapıyor; Caddy'ye sabit hostname yazarsan eşleşmeyen istekler 404 döner.
>
> Monolith `ENVIRONMENT=production` ile çalışır ve secret'lar boş/default kalırsa
> **başlamayı reddeder** — bilinçli bir önlem, sessiz kırık deploy olmaz.

> **Parola rotasyonunda iki tuzak var.**
>
> 1. `POSTGRES_PASSWORD` ve `RABBITMQ_DEFAULT_PASS` yalnızca **boş data dizininde**
>    uygulanır. Çalışan bir stack'te env'i değiştirmek DB kullanıcısının parolasını
>    değiştirmez; uygulama yeni parolayla bağlanmaya çalışır ve auth hatası alır.
>    Gerçekten rotate etmek istiyorsan volume'u boşalt (ya da yeni bir volume adı ver)
>    — `seed` demo verisini zaten yeniden üretir.
> 2. `redis` parolası `command:` içindeki `--requirepass`'ten geliyor. API `command`
>    alanını **maskelemiyor ve PATCH'te korumuyor**: yalnızca `REDIS_PASSWORD` env'ini
>    güncellersen redis eski parolayla ayağa kalkar. İkisini birlikte güncelle.
>    Ayrıntı: [OPENSHIP-PROBLEMS.md kusur 12](OPENSHIP-PROBLEMS.md).

### C6. Deploy

Deploy'a bas. Sıra: imajlar build edilir → `migrate` şemayı kurar → `monolith` +
`notification` başlar → `caddy` SPA'yı servis eder → edge domain'i bağlar.

Sonrası **push-to-deploy**: `main`'e her push Openship'in webhook'unu tetikler,
sadece değişen servisler yeniden build edilir. A bölümündeki
`make autodeploy-install` (systemd poll timer) burada **gereksiz** — kurma.

### C7. Bilmen gereken iki davranış farkı

**1. `depends_on` koşulları düşer.** Openship compose'un `depends_on`
*bağlantısını* okur ama `condition: service_healthy` /
`service_completed_successfully` kısmını okumaz. Pratikte:

- `migrate` bu yüzden şemayı kurmadan önce DB'nin bağlantı kabul etmesini
  [kendi içinde bekler](new-backend/infrastructure/migrate/entrypoint.sh)
  (`pg_isready`, 120 sn). `restart: "no"` olduğu için orada patlamak kalıcı
  olurdu.
- `monolith` migration'lardan önce başlarsa DB'ye bağlanamayıp ölür ve
  `restart: unless-stopped` ile geri gelir. İlk deploy'da loglarda birkaç
  restart görmek **normal**; birkaç saniyede oturur.

**2. Dashboard düzenlemesi repoyu ezmez.** Openship bir alanı dashboard'dan
değiştirdiğinde repo dosyası ile ayrıştığını "drift" olarak işaretler ve seni
seçim yapmaya çağırır. Kalıcı değişiklikler için compose dosyasını düzenleyip
push et — tek doğruluk kaynağı repo kalsın.

### C8. Sorun giderme

| Belirti | Sebep / çözüm |
|---|---|
| Log'da `runtime: static` ve jenerik 6 adımlık Dockerfile | Proje compose olarak tanınmamış — C2/C3 yapılmamış. Dashboard'dan düzeltilemez |
| `Deploy failed: mkdir: Permission denied` | C0 atlanmış; `/opt/openship/static` sunucuda yok veya SSH kullanıcısının değil |
| `COPY failed: ... package.json: file does not exist` | Build context yanlış — C3'teki tablo ile `build` alanlarını karşılaştır |
| `port is already allocated` (80 veya 443) | `caddy` public işaretlenmemiş (C4), ya da yanlışlıkla `docker-compose.standalone.yml` de yüklenmiş — Openship sadece base dosyayı kullanmalı |
| `monolith` sürekli restart | Loglara bak: genelde boş bırakılmış secret (C5) |
| Login 500 / CORS hatası | `PUBLIC_ORIGIN` tam `https://<domain>` mi, sonda `/` var mı |
| Sayfa açılıyor ama `/api` 404 | Domain `caddy`'ye değil başka bir servise bağlanmış (C4) |

---

## Aynı sunucuda birden fazla proje

> Openship kullanıyorsan (C) bu bölümü atla — çoklu proje ve hostname
> yönlendirmesi zaten onun işi.

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
hiçbir şey yapmadan çıkar, varsa `git pull --ff-only` + `make deploy` yapar.

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
cd ~/mydreamcampus       # repo köküne dön
make deploy              # ilk sefer 3-6 dk (Go + frontend derlenir)
```

> Çıplak `docker compose up -d` **çalıştırma**: `-f` vermeden çağırdığında
> standalone overlay'i yüklemez, Caddy `:443`'ü açmaz ve HTTPS gelmez.
> `make deploy` doğru dosya setini kendisi veriyor.

Sıra otomatik: infra sağlıklı olunca `migrate` çalışır → bitince `monolith` +
`notification` başlar → `caddy` TLS sertifikasını çeker.

Migration loglarını gör:

```bash
make deploy-ps                        # hepsi "running", migrate "exited (0)"
make deploy-logs                      # monolith + caddy, canlı
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

Tek bir servise müdahale gerekirse compose'a doğrudan da geçebilirsin. Stack iki
dosyaya bölündüğü için ikisini de vermen gerekir; `COMPOSE_FILE` (compose'un
kendi değişkeni, `:` ile ayrılır) bunu bir kez ayarlamanı sağlar:

```bash
cd ~/mydreamcampus
export COMPOSE_FILE=new-backend/infrastructure/docker-compose.yml:new-backend/infrastructure/docker-compose.standalone.yml

docker compose restart monolith
docker compose logs migrate
docker compose logs seed
docker compose up -d --build seed
docker compose exec -T postgres psql -U postgres -d mydreamcampus \
  < new-backend/monolith/seed_courses.sql
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
| `port is already allocated` | `HTTP_PORT` ile mobil override'ın 8080'i çakışıyor — birini değiştir. |
| Ev ağındaki telefondan açılmıyor | `PUBLIC_ORIGIN` `localhost` kalmış olabilir — LAN IP + port olmalı. Ayrıca `sudo ufw allow <HTTP_PORT>/tcp`. |
| `monolith` sürekli restart | `make deploy-logs` → genelde `.env`'de eksik/default secret. Düzelt, `make deploy`. |
| Sertifika uyarısı | Caddy henüz cert almadı: `logs caddy`. 80/443 firewall'da açık mı? `PUBLIC_HOST` gerçekten IP'ye çözülüyor mu (`dig 203-0-113-5.sslip.io`)? |
| `migrate` exit code ≠ 0 | `logs migrate`. DB henüz hazır değilse tekrar: `docker compose up -d migrate`. |
| Login 500 / CORS | `.env`'de `PUBLIC_ORIGIN` tam `https://<host>` mi (sonda `/` yok)? |
| Build OOM (2GB) | Adım 4b swap ekle veya droplet'i 4GB'a resize et. |

## Domain alınca (opsiyonel, sonra)

`.env`'de `PUBLIC_HOST` ve `PUBLIC_ORIGIN`'i gerçek domain'e çevir, DNS A
kaydını droplet IP'sine yönlendir, `docker compose up -d caddy`. Caddy yeni
domain için otomatik sertifika alır.
