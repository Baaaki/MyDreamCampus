# Faz 8 — Ev sunucusu ve Cloudflare Tunnel

## Başlamadan
- Oku: `DEPLOY.md` (özellikle "A8. Cloudflare Tunnel ile internete açmak" ve
  "Otomatik deploy").
- README §4 **K4** cevabı.
- Faz 7 bitmiş olmalı.

## Bağlam
- Base compose Caddy'nin portunu tüm arayüzlere açıyor
  (`new-backend/infrastructure/docker-compose.yml` ≈429,
  `"${HTTP_PORT:-80}:80"`). Tunnel kurulumunda bu, Cloudflare'i atlayan
  doğrudan bir HTTP kapısı demek.
- `scripts/auto-deploy.sh` (≈34-35), `main` her ilerlediğinde CI sonucuna
  bakmadan deploy ediyor.
- Demo HTTPS'le (Cloudflare) sunulacağı için Secure çerezler sorun
  çıkarmaz.

## Görevler

- [x] **8.1 Tunnel compose katmanı** (gemini ile yapıldı)
  - Yeni `new-backend/infrastructure/docker-compose.tunnel.yml`:
    - `cloudflared` servisi: `cloudflare/cloudflared`, **sabit sürüm
      etiketi**; `command: tunnel --no-autoupdate run`; env `TUNNEL_TOKEN`;
      `restart: unless-stopped`.
    - Caddy'nin host portlarını kaldır (`ports: !reset []`). Cloudflared
      Caddy'ye compose ağı üzerinden (`http://caddy:80`) bağlanır; sunucuda
      dışarıya açık port kalmaz.
  - Compose ağının alt ağını sabitle (örn. `172.28.0.0/16`). `frontend/Caddyfile`
    `trusted_proxies 172.16.0.0/12` bunu kapsamalı: böylece cloudflared'ın
    gönderdiği X-Forwarded-For'a güvenilir ve gerçek istemci IP'si servislere
    ulaşır. Rootless Docker'daki IP kaybı da bu yolla ortadan kalkar.
  - Makefile: `EDGE=tunnel` iken compose dosyaları base + tunnel olsun
    (standalone yüklenmesin). `EDGE` verilmezse davranış değişmesin.
  - **Alternatif:** Kullanıcı host'ta zaten cloudflared çalıştırıyorsa bu
    dosya yerine Caddy portu `127.0.0.1:${HTTP_PORT}:80`'e bağlanır. Bunu
    DEPLOY.md'de anlat.
  - **Commit:** `feat(infra): add a Cloudflare Tunnel compose layer`

- [ ] **8.2 Ortam değişkenleri**
  - `new-backend/infrastructure/.env.example`'a ekle:
    - `DEMO_MODE=true`, `DEMO_ADMIN_EMAIL`, `DEMO_TEACHER_EMAIL`,
      `DEMO_STUDENT_EMAIL`
    - `TUNNEL_TOKEN`
    - `SEED_DEMO=true`
    - `NIGHTLY_RESET_AT=04:00`, `BASELINE_KEEP=7`, `BASELINE_OFFSITE_CMD=`
    - `DB_MAX_CONNS`
    - Tunnel örneği: `PUBLIC_HOST=:80`, `PUBLIC_ORIGIN=https://<alan-adı>`
  - `PROTECTED_ACCOUNT_EMAILS`'i compose'da `ADMIN_EMAIL` ve `DEMO_*`
    değerlerinden türet; kullanıcı elle yazmasın.
  - `make check-env`: `DEMO_MODE=true` ise gerekli değişkenlerin dolu olduğunu
    ve `EDGE=tunnel` ise `TUNNEL_TOKEN`'ın olduğunu kontrol etsin.
  - `frontend/Caddyfile`: `/assets/*` için uzun süreli
    `Cache-Control: public, max-age=31536000, immutable` (dosya adları hash'li;
    Cloudflare önbelleği için).
  - **Commit:** `chore(infra): add demo environment settings`

- [ ] **8.3 Otomatik deploy için CI kapısı**
  - `scripts/auto-deploy.sh`: `git pull` + `make deploy` öncesinde
    `origin/main` commit'inin `ci-passed` check-run'ının başarılı olduğunu
    kontrol et (GitHub API; repo public ise token'sız, değilse
    `GITHUB_TOKEN` env). Başarılı değilse çık ve sonraki turda tekrar dene.
    "Bekliyor" ve "başarısız" durumlarını logla.
  - auto-deploy `EDGE=tunnel` ile çalışsın (systemd unit veya `.env`).
  - **Commit:** `fix(infra): deploy only commits that passed CI`

- [ ] **8.4 Mobil yayın ayarları**
  - `mobile/app.json`: `name: "MyDreamCampus"`, anlamlı bir `slug` ve
    `scheme`. `extra.eas.projectId` kullanıcıdan gelecek; şimdilik yer tutucu
    kalsın ve not düş.
  - `mobile/eas.json` `preview` profili:
    `env.EXPO_PUBLIC_API_URL = https://<alan-adı>/api` (alan adını
    kullanıcıdan al).
  - **Commit:** `chore(mobile): prepare release build settings`

- [ ] **8.5 DEPLOY.md'ye "Demo kurulumu" bölümü**
  - Sunucuda adım adım:
    1. repoyu al,
    2. `.env`'i doldur,
    3. `EDGE=tunnel make deploy`,
    4. seed'in ve ilk kalıcı durumun oluştuğunu kontrol et,
    5. süper admin ilk giriş ve şifre değişimi,
    6. `make autodeploy-install`.
  - 8.6'daki Cloudflare adımları.
  - Sorun giderme:
    - tunnel bağlanmıyor,
    - loglarda gerçek IP görünmüyor,
    - geri dönüş (restore) hatası,
    - kilit takılı kaldı.
  - **Commit:** `docs(infra): add the demo deployment guide`

- [ ] **8.6 Kullanıcının yapacakları**
  Hesap veya erişim gerektirir; kullanıcı yapar, sen işaretlersin.
  - [ ] Cloudflare'de alan adı. Zero Trust → Networks → Tunnels → yeni tunnel
    (Docker). Token → `.env` `TUNNEL_TOKEN`. Public hostname:
    `<alan-adı>` → `http://caddy:80`.
  - [ ] SSL/TLS → Edge Certificates → "Always Use HTTPS" açık.
  - [ ] Security → WAF → Rate limiting rule: `/api/auth/login` için IP
    başına dakikada 10 istek.
  - [ ] (K4) Zero Trust → Access → Applications → Self-hosted:
    `<alan-adı>/api/catalog/admin/ops*`. Policy: yalnız kullanıcının
    e-postası (One-time PIN).
  - [ ] GitHub → Settings → Branches: `main` koruması (PR zorunlu ve
    `ci-passed` status check).
  - [ ] UptimeRobot (ücretsiz): `https://<alan-adı>/health`, 5 dakikada bir.
  - [ ] Sunucuda `.env`:
    - her secret `openssl rand -base64 48` ile üretilmiş;
    - `ADMIN_EMAIL` tahmin edilemez bir adres;
    - `ADMIN_INITIAL_PASSWORD` güçlü.

    Ardından `EDGE=tunnel make deploy` ve `make autodeploy-install`.
  - [ ] Süper admin ilk giriş: şifre değişimi, sonra "Kalıcı Veri" sayfasında
    ilk kalıcı durumun göründüğünü kontrol et.
  - [ ] Expo hesabı: `eas init` (projectId), ardından
    `eas build -p android --profile preview`. APK linkini README'ye ekle.
  - [ ] (İsteğe bağlı) Sunucu dışı yedek: rclone kur ve
    `BASELINE_OFFSITE_CMD`'yi ayarla.

## Faz sonu
- README §5 yeşil olmalı.
- README §1 madde 6.
