# Ev sunucusunda canlı demo kurulumu — adım adım

Bu rehber Faz 8.6'yı (sana kalan adımlar) sırasıyla uygulatır. Sonunda
`https://mydreamcampus.madebybaki.com` internetten açılır ve sunucuda dışarıya
açık port kalmaz. Ardından Faz 9'daki canlı kontrollere geçilir.

Ayrıntı ve sorun giderme: [DEPLOY.md](../../DEPLOY.md), "C. Canlı Demo Kurulumu".

**Her adımın başında nerede yapıldığı yazıyor:**
- **[Sunucu]** ev sunucusunda, SSH ile. Komutlar repo kökünden çalışır.
- **[Cloudflare]**, **[GitHub]**, **[UptimeRobot]** tarayıcıda.
- **[Geliştirme]** bu bilgisayarda.

`sudo` ile başlayan komutları sen çalıştırıyorsun. Sunucuda `docker` sudo
istiyorsa (rootless değilse) `docker` ile başlayan komutların başına da `sudo`
ekle. `make` hedefleri bunu kendisi ayarlıyor.

> **Önemli — sunucudaki mevcut veri silinecek.** Süper admin hesabı ve demo
> içeriği yalnız **boş** bir veritabanında oluşuyor: auth, `ADMIN_EMAIL`
> hesabını ilk açılışta yaratıyor, seed de içinde öğrenci olan bir sistemi
> atlıyor. Bugünkü LAN kurulumunun verisiyle demo doğru kurulmaz. Adım 3'te
> önce yedek alıp sonra sıfırlıyoruz.

> **LAN adresi kapanacak.** Tunnel modunda Caddy host'a port açmaz;
> `http://192.168.1.106` artık çalışmaz. Evden de
> `https://mydreamcampus.madebybaki.com` kullanılır. Bu bilgisayardaki
> `mobile/.env` dosyasını da buna göre değiştir (adım 13).

---

## 0. Hazırlık — hesaplar

- [ ] **Cloudflare:** hesap ve `madebybaki.com` alan adı. Alan adının DNS'i
  Cloudflare'de olmalı (Dashboard → alan adı "Active" görünmeli). Değilse:
  "Add a site", ardından alan adını aldığın firmada nameserver'ları
  Cloudflare'in verdikleriyle değiştir. Aktif olması birkaç saat sürebilir.
- [ ] **Cloudflare Zero Trust:** Dashboard → Zero Trust. İlk girişte bir
  "team name" seç ve **Free** planı etkinleştir (kart bilgisi istenebilir,
  ücret alınmaz).
- [ ] **Expo** hesabı (APK için; https://expo.dev).
- [ ] **UptimeRobot** hesabı (ücretsiz).

---

## 1. [Sunucu] Paketler

```bash
sudo apt update
sudo apt install -y git make curl jq openssl
docker info >/dev/null && echo "docker OK"
```

`jq` zorunlu: otomatik deploy, CI'ın sonucunu onunla okuyor. Yoksa deploy
etmez ve hata verir.

Docker rootless kuruluysa (DEPLOY.md A1), SSH oturumu kapanınca da çalışması
için şu komutu bir kez çalıştır:

```bash
sudo loginctl enable-linger $USER
```

## 2. [Sunucu] Otomatik deploy'u durdur, kodu güncelle

Kurulum sırasında zamanlayıcı araya girip deploy etmesin:

```bash
cd ~/mydreamcampus                  # repo neredeyse
systemctl --user list-timers mydreamcampus-deploy.timer --all
make autodeploy-off 2>/dev/null || true
git pull --ff-only origin main
git log -1 --oneline                # son commit GitHub'dakiyle aynı olmalı
```

Repo sunucuda yoksa:
`git clone https://github.com/Baaaki/MyDreamCampus.git ~/mydreamcampus`.

## 3. [Sunucu] Mevcut veriyi yedekle ve sıfırla

Bu adımı `.env`'i değiştirmeden **önce** yap: eski ayarlarla çalışan stack'i
eski ayarlarla kapatmak gerekiyor.

```bash
make deploy-ps          # stack çalışıyor mu?
make backup             # ~/mydreamcampus-backups/all-<tarih>.sql.gz
ls -lh ~/mydreamcampus-backups | tail -3
```

Stack çalışmıyorsa `make backup` hata verir; saklamak istediğin veri varsa
önce `make deploy` ile kaldırıp yedeği öyle al. Yedeğin boyutu 0'dan büyükse
sıfırla. **Bu komut bütün veriyi siler:**

```bash
make clean
docker volume ls | grep mydreamcampus_postgres-data    # boş dönmeli
```

Listede compose'da artık tanımlı olmayan eski volume'lar kalabilir (ör.
`mydreamcampus_notification-postgres-data`). Demo'yu etkilemezler;
istersen `docker volume rm <ad>` ile sil.

## 4. [Cloudflare] Tunnel oluştur ve token'ı al

1. Zero Trust → **Networks → Tunnels** (yeni arayüzde **Networks →
   Connectors → Cloudflare Tunnels**) → **Create a tunnel**.
2. Tür: **Cloudflared**. Ad: `mydreamcampus`.
3. "Install and run a connector" ekranında **Docker**'ı seç. Gösterilen
   komutta `--token` sonrasındaki uzun değeri kopyala (`eyJ...` ile başlar).
   O komutu **çalıştırma**: cloudflared'ı stack kendisi başlatıyor.
4. İleri → **Public Hostname** (yeni arayüzde **Published application
   routes**):
   - Subdomain: `mydreamcampus`, Domain: `madebybaki.com`, Path: boş
   - Service Type: **HTTP**, URL: **`caddy:80`**. `localhost` değil:
     cloudflared, Caddy'ye Docker ağı üzerinden `caddy` adıyla ulaşır.
5. Kaydet. DNS kaydı (CNAME) otomatik oluşur.

Tunnel şimdilik "Inactive" görünür; adım 6'dan sonra "Healthy" olur.

## 5. [Sunucu] `.env`'i yeniden yaz

Eskisini yedekleyip örnekten başla:

```bash
cd new-backend/infrastructure
mv .env .env.lan-backup 2>/dev/null || true
cp .env.example .env
```

Her secret için ayrı değer üret. Hex kullan, base64 değil: base64'teki `/`
karakteri RabbitMQ ve Postgres bağlantı URL'lerini bozar.

```bash
for v in POSTGRES_PASSWORD SERVICE_DB_PASSWORD REDIS_PASSWORD RABBITMQ_PASSWORD JWT_SECRET INTERNAL_SERVICE_SECRET QR_SECRET; do
  echo "$v=$(openssl rand -hex 32)"
done
```

Çıktıyı `nano .env` ile ilgili satırlara yapıştır, sonra şu satırları
düzenle:

```
ADMIN_EMAIL=<tahmin edilemez bir adres; süper admin girişi bu>
ADMIN_INITIAL_PASSWORD=<güçlü bir geçici şifre; ilk girişte değiştirilecek>

SEED_DEMO=true
DEMO_MODE=true
DEMO_ADMIN_EMAIL=demo.admin@mydreamcampus.com
DEMO_TEACHER_EMAIL=ahmet.yilmaz@uni.edu.tr
DEMO_STUDENT_EMAIL=zeynep.sahin@uni.edu.tr

REVERT_AFTER_MINUTES=60
BASELINE_KEEP=7

EDGE=tunnel
TUNNEL_TOKEN=<adım 4'te kopyaladığın token>

PUBLIC_HOST=:80
PUBLIC_ORIGIN=https://mydreamcampus.madebybaki.com
```

- `TUNNEL_TOKEN` örnekte yorum satırı; başındaki `#` ve boşlukları sil.
- Dosyanın sonundaki eski `PUBLIC_HOST=http://192.168.1.50` ve
  `PUBLIC_ORIGIN=...` satırlarını yukarıdakilerle **değiştir** (iki kez
  tanımlı kalmasın).
- `HTTP_PORT` / `HTTPS_PORT` satırları varsa sil; tunnel modunda
  kullanılmıyorlar.
- `ADMIN_INITIAL_PASSWORD`'da `$` ve `#` kullanma. Compose `.env`'de `$`'ı
  değişken, boşluktan sonraki `#`'i yorum sanar ve şifre kırpılır.
- `ADMIN_EMAIL` gerçek bir posta kutusu olmak zorunda değil (SMTP kapalı).
  Ama kimsenin tahmin edemeyeceği bir adres olsun, çünkü demo hesaplar giriş
  ekranında herkese açık.

Kontrol et:

```bash
cd ~/mydreamcampus
make check-env && echo "env OK"
```

## 6. [Sunucu] Yayına al

```bash
make deploy
```

İlk derleme birkaç dakika sürer. Bittiğinde:

```bash
make deploy-ps
```

- `migrate` ve `seed` → **Exited (0)**; diğer hepsi → **Up**.
- `cloudflared` listede olmalı. `caddy`'nin PORTS sütunu boş olmalı.

```bash
docker logs mydreamcampus-seed 2>&1 | grep "seed complete"
docker logs mydreamcampus-cloudflared 2>&1 | grep -m1 "Registered tunnel connection"
```

İlk kalıcı durum seed'den sonra alınır (bir dakika kadar sürebilir):

```bash
docker logs mydreamcampus-demo-ops 2>&1 | grep "Snapshot completed"
```

## 7. [Tarayıcı] İlk kontrol ve süper admin girişi

Telefondan **mobil veriyle** (Wi-Fi kapalı) aç:

- `https://mydreamcampus.madebybaki.com/health` → JSON döner.
- `https://mydreamcampus.madebybaki.com` → giriş ekranı ve demo hesapları
  paneli.

Süper admin olarak gir:

1. `ADMIN_EMAIL` + `ADMIN_INITIAL_PASSWORD` ile giriş yap. Şifre değiştirme
   ekranına yönlendirilirsin; güçlü yeni bir şifre belirle. Bu adım
   zorunlu: şifre değişmeden başka hiçbir admin işlemi çalışmaz.
   - `ADMIN_INITIAL_PASSWORD` yalnız ilk açılışta, süper admin
     oluşturulurken kullanılır. Sonradan `.env`'de değiştirmek şifreyi
     değiştirmez; şifre arayüzden değişir.
   - Yeni şifren ve açık oturumların saatlik geri dönüşten, "Düzenlemeye
     Başla"dan ve "Vazgeç ve Sıfırla"dan etkilenmez; kalıcı durum ilk şifreyi
     taşısa da geri dönüş onu geri getirmez.
2. Menüden **Kalıcı Veri** sayfasını aç (`/system/baseline`).
   `YYYYMMDD-HHMMSS` adlı bir sürüm listelenmeli.

## 8. [Cloudflare] Güvenlik ayarları

**HTTPS zorunlu:** alan adı → SSL/TLS → Edge Certificates →
**Always Use HTTPS**: açık.

**Demonun bağlı olduğu ayarlar** — varsayılanlar çoğunda doğru, ama biri
değişirse uygulama kırılır:
- **Bot Fight Mode kapalı** (Security → Bots): açıkken mobil uygulamanın ve
  sayfanın XHR isteklerini challenge'a sokar, bu istemciler challenge
  çözemez. Free planda WAF kuralıyla istisna yazılamıyor; tek yol kapatmak.
- **Security Level "I'm Under Attack" olmasın** (Security → Settings): bu mod
  her isteğe tarayıcı doğrulaması koyar, API ve mobil aynı sebeple kırılır.
- **Rocket Loader kapalı** (Speed → Optimization → Content Optimization):
  script'leri kendi yükleyicisiyle değiştirir; CSP'deki `script-src 'self'`
  ile çakışır.
- **Web Analytics otomatik enjeksiyonu kapalı** (Analytics & Logs → Web
  Analytics → siteyi otomatik kurulumla ekleme): enjekte ettiği
  `static.cloudflareinsights.com` script'ini CSP engeller, her sayfada konsol
  hatası çıkar.
- **`/api/*` için cache kuralı yazma** (Caching → Cache Rules): API yanıtları
  kullanıcıya özel; önbelleğe alınırsa bir ziyaretçinin verisi başkasına
  gider.

**Giriş için rate limit:** alan adı → Security → WAF → **Rate limiting
rules** → Create rule:
- Ad: `login`
- Eşleşme: URI Path **equals** `/api/auth/login`
- Characteristics: IP
- Free planda sayım ve engel süresi 10 saniyeye sabit. Ör. **10 saniyede
  5 istek → Block**. Planın ücretliyse "dakikada 10 istek" kullan.

**Kalıcı Veri uçlarını kilitle (K4):** Zero Trust → **Access → Applications**
→ Add an application → **Self-hosted**:
- Ad: `MyDreamCampus Ops`
- Domain: `mydreamcampus.madebybaki.com`, Path: `api/catalog/admin/ops*`
- Policy: Action **Allow**, Include → **Emails** → kendi e-posta adresin
- Kimlik doğrulama: **One-time PIN** (Settings → Authentication'da açık
  olmalı; varsayılan olarak açık).

Sonra Kalıcı Veri sayfasını yenile. Sayfa "Sistem durumu okunamadı" diyip
bir doğrulama bağlantısı gösterecek. Bağlantıyı aç, e-postana gelen kodu gir,
sayfaya dönüp **Yenile**'ye bas. Doğrulama 24 saat geçerli; süresi dolunca
aynısını tekrarlarsın.

## 9. [Sunucu] Otomatik deploy'u aç

```bash
make autodeploy-install
make autodeploy-status
```

Artık `main`'e gelen her commit, CI'ı (`ci-passed`) geçince en geç 2 dakika
içinde sunucuya kendiliğinden deploy edilir. Geçmezse deploy edilmez.

```bash
make autodeploy-logs     # "waiting for CI", "ci-passed succeeded", "deploy complete"
```

## 10. [Sunucu] Dışarıya açık kapı kalmadığını doğrula

Sunucuda:

```bash
docker ps --format '{{.Names}}\t{{.Ports}}' | grep caddy     # host portu yok
```

Eski ayarlardan kalan firewall kuralını kapat (varsa):

```bash
sudo ufw status
sudo ufw delete allow 8080/tcp     # hangisi açıksa: 80/tcp, 8080/tcp
```

Evdeki başka bir cihazdan:

```bash
curl -m 5 http://192.168.1.106      # "Connection refused" / zaman aşımı beklenir
```

Router'da bu sunucuya port yönlendirmesi (80/443) varsa artık gerekmez; kaldır.

## 11. [GitHub] Dal koruması

Repo → Settings → **Branches** → Add rule (ya da **Rules → Rulesets**),
dal: `main`:
- **Require status checks to pass** → `ci-passed`
- (Plandaki tercih) **Require a pull request before merging**

> Bu açıkken `main`'e doğrudan push edilemez; her değişiklik PR'la ve CI
> yeşilken girer. Yönetici olarak atlayabilmek istersen "Do not allow
> bypassing" / "Include administrators" kutusunu **işaretleme**. Otomatik
> deploy zaten yalnız CI'ı geçen commit'i aldığı için bu adım ek bir
> emniyet, zorunlu değil.

## 12. [UptimeRobot] Erişilebilirlik takibi

New monitor → Type **HTTP(s)** → URL
`https://mydreamcampus.madebybaki.com/health` → Interval **5 minutes** →
bildirim için e-postanı seç.

## 13. [Geliştirme] Mobil APK

```bash
cd mobile
npx eas-cli login
npx eas-cli init                                   # app.json'a projectId yazar
npx eas-cli build -p android --profile preview     # bulutta derler, ~15 dk
```

- `eas init`'in `app.json`'da yaptığı değişikliği commit'le (ya da bana
  söyle, ben commit'lerim).
- Build bitince verilen APK bağlantısını telefona kur. Giriş panelini,
  rezervasyonu ve `4242 4242 4242 4242` ile ödemeyi dene. Bağlantıyı README'ye
  ekleyelim.
- Bu bilgisayarda Expo ile geliştirirken `mobile/.env` içindeki
  `EXPO_PUBLIC_API_URL`'i `https://mydreamcampus.madebybaki.com/api` yap.
  LAN adresi artık cevap vermiyor.

## 14. (İsteğe bağlı) [Sunucu] Kalıcı durumların sunucu dışı yedeği

```bash
sudo apt install -y rclone
rclone config                      # ör. Google Drive → uzak ad: gdrive
docker volume inspect mydreamcampus_demo-baselines --format '{{ .Mountpoint }}'
crontab -e
```

Crontab'a ekle (`<Mountpoint>` yerine bir üstteki çıktıyı yaz):

```
30 4 * * * rclone copy <Mountpoint> gdrive:mydreamcampus-backups
```

`.env`'deki `BASELINE_OFFSITE_CMD` boş kalır: o komut demo-ops
konteynerinin içinde çalışıyor ve orada rclone yok.

## 15. Faz 9 — canlı kontroller

Her şey ayaktayken
[phase-9-final-check.md](phase-9-final-check.md) listesini tarayıcıda ve
APK'da sırayla dene. Saatlik geri dönüş maddesi için:
`.env`'de `REVERT_AFTER_MINUTES=2` yap, `make deploy`, demo admin ile bir
kayıt ekle, 2-3 dakika sonra kaydın gittiğini gözle, değeri 60'a döndür,
yine `make deploy`. Geri dönüşten sonra süper admin oturumun açık kalmalı ve
adım 7'de belirlediğin şifreyle giriş yapabilmelisin; `ADMIN_INITIAL_PASSWORD`
artık reddedilir.

Sonuçları bana madde madde yaz (geçti / şu hata). Faz 9'u birlikte
işaretleyip kapatalım; hata çıkan maddeleri düzeltirim.

---

## Takılırsan

| Belirti | Bak |
|---|---|
| `make check-env` hata veriyor | Mesajdaki değişken: `CHANGE_ME` kalmış, `TUNNEL_TOKEN` boş ya da `EDGE` yanlış yazılmış |
| `Pool overlaps with other one on this address space` | `.env`'e `STACK_SUBNET=172.16.239.0/24` ekle, `make deploy` |
| Tunnel "Inactive" / site 502 | `docker logs mydreamcampus-cloudflared`; Public Hostname URL'si `caddy:80` mi |
| Site açılıyor ama giriş paneli boş | `docker logs mydreamcampus-seed` — seed "already holds data" dediyse adım 3 atlanmış |
| Kalıcı Veri "okunamadı" | Adım 8'deki Access doğrulaması |
| Deploy olmuyor | `make autodeploy-logs` — `jq` eksik mi, CI kırmızı mı |
| Diğerleri | DEPLOY.md → "Sorun giderme" |
