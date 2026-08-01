# Openship Deploy Playbook

Bir sonraki projeni Openship'e (self-hosted, v0.4.8) deploy ederken izleyeceğin
adımlar. Bu dosya **genel** bir rehber — MyDreamCampus'a özel kusur kayıtları
[OPENSHIP-PROBLEMS.md](OPENSHIP-PROBLEMS.md)'de, bu projenin kendi deploy
seçenekleri [DEPLOY.md](DEPLOY.md)'de.

Buradaki her madde 2026-07-31/08-01'de gerçekten yaşanmış bir duvardan çıktı.

---

## 0. Önce şunu oku: Openship'in zihin modeli

Bu üç şeyi bilmezsen saatlerini kaybedersin. Openship'in davranışı compose
semantiğinden **üç noktada** ayrılıyor.

### 0.1 Build context her zaman repo köküdür

Openship servis başına context kurmaz. Repoyu bir kez klonlar ve o klonun kökünü
**tek bir "shared build context"** olarak bütün servislere verir. Compose'daki
`context:` alanı **yok sayılır**; servisin `build` alanı yalnızca Dockerfile'ı
bulmak için kullanılır (`<build>/<dockerfile>`).

```
Preparing shared build context...
Shared build context ready (4.7 MB)
```

**Sonuç:** Dockerfile'larındaki bütün `COPY` yolları **repo köküne göre** olmalı.
Değilse ilk `COPY`'de `file does not exist` alırsın.

Bu kusur repo kökünde tek compose + `context: .` olan projelerde görünmez —
orada iki taban zaten çakışır. Alt dizinden build eden her stack'te patlar.

### 0.2 `depends_on` koşulları düşer

Openship `depends_on` *bağlantısını* okur ama `condition: service_healthy` /
`service_completed_successfully` kısmını okumaz. Migration container'ın DB'nin
hazır olmasını **kendi** beklemeli (entrypoint'te `pg_isready` döngüsü gibi).

### 0.3 Public servisi işaretlemezsen edge ile çakışır

Openship yalnızca `exposed: true` işaretli servisin portunu
`127.0.0.1:<pinned>:80` olarak yeniden bağlar. İşaretlemezsen compose'daki
`80:80` olduğu gibi host'a publish edilir ve Openship'in kendi edge'i
(OpenResty, host network'te `:80`/`:443`) ile çakışır.

---

## 1. Deploy öncesi repo hazırlığı

Bunları deploy'a basmadan **önce** yap; sonradan düzeltmek bir tur build demek.

### 1.1 Dockerfile'ları repo kökü göreli yap

Her `COPY` kaynağı repo kökünden çözülebilmeli. Hedefleri değiştirme, sadece
kaynaklara önek ekle:

```dockerfile
# ONCE (context: new-backend/ varsayimiyla)
COPY shared/ ./shared/
COPY monolith/go.mod monolith/go.sum ./monolith/

# SONRA (context: repo koku)
COPY new-backend/shared/ ./shared/
COPY new-backend/monolith/go.mod new-backend/monolith/go.sum ./monolith/
```

Hedef (`./shared/`) aynı kaldığı için imajın içindeki yerleşim değişmez — Go
`replace` yolları, CWD-göreli glob'lar vs. bozulmaz.

**Doğrula** — her kaynak repo kökünden var mı:

```bash
for f in $(git ls-files '*Dockerfile'); do
  grep -E "^\s*COPY " "$f" | grep -v -- "--from=" | sed -E 's/^\s*COPY\s+//' | while read -r line; do
    set -- $line; n=$#; i=1
    for src in "$@"; do
      [ $i -eq $n ] && break
      [ -e "$src" ] && echo "OK   $f : $src" || echo "FAIL $f : $src"
      i=$((i+1))
    done
  done
done
```

### 1.2 Compose context'lerini repo köküne çek

```yaml
monolith:
  build:
    context: ../..                              # repo koku
    dockerfile: new-backend/monolith/Dockerfile # context'e gore
```

**Doğrula:**

```bash
docker compose -f <compose> config --format json \
  | python3 -c "import json,sys; [print(n, s['build']['context'], s['build']['dockerfile']) for n,s in json.load(sys.stdin)['services'].items() if s.get('build')]"
```

Hepsi aynı (repo kökü) context'i göstermeli.

### 1.3 Köke `.dockerignore` ekle — atlanabilir değil

Context repo kökü olduğu için Docker **yalnızca kökteki** `.dockerignore`'u
okur. Alt dizinlerdekiler artık ölü. Bu dosya olmadan yerel build'de tüm
worktree (node_modules dahil, kolayca 1+ GB) daemon'a gider.

```
.git/
**/node_modules/
mobile/
*.md
**/.env
**/*.log
dist/
```

Temiz bir clone'da zaten node_modules yok — bu dosya asıl **yerel** build'i
korur.

### 1.4 `openship.json` yazıyorsan şemayı doğrula

Şemanın kökünde `additionalProperties: false` var. **Tanımsız tek bir alan tüm
dosyayı geçersiz kılar ve Openship bunu sessizce yutar** — ne hata, ne uyarı,
proje auto-detection'a düşer.

Alan adlarını <https://openship.io/openship.schema.json> ile karşılaştır.

> Not: `framework` alanı config overlay tarafından **uygulanmıyor**. Dosyaya
> yazsan da işe yaramaz; adım 2'de API'den set edeceksin.

### 1.5 Bind-mount kullanıyorsan dikkat

Compose'daki `./config/foo.conf:/etc/foo.conf` gibi göreli bind-mount'lar
**mutlak yola** çözülür ve Openship o mutlak yolu saklar. Senin makinendeki yol
sunucuda yoktur.

İki seçenek:
- Sunucuda repo checkout'u tut ve mount'ları o yola göre yaz
  (`/home/<user>/<repo>/...`)
- Ya da dosyayı imaja `COPY`'le, mount'u tamamen kaldır (tercih edilen)

---

## 2. Adım adım deploy

### Adım 1 — Projeyi `services` tipiyle oluştur

**Dashboard'dan compose projesi oluşturamazsın.** `docker-compose` framework
seçicisinden kasıtlı olarak çıkarılmış (`Frameworks.tsx` → `EXCLUDED_STACKS`);
UI'dan açarsan proje tek bir uygulama sanılır ve static pipeline'a düşer.

```bash
openship project create --name <ad> \
  --git-owner <owner> --git-repo <repo> --git-branch main --type services
```

Proje zaten varsa tipini API'den çevir:

```bash
curl -X PATCH https://<host>/api/projects/<projectId> \
  -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' \
  -d '{"framework": "docker-compose"}'
```

### Adım 2 — Servisleri kaydet

```bash
openship service sync <compose-dosyasi> --project <projectId>
```

Bu komut **yerelde** `docker compose config` çalıştırır, yani `${VAR}`
interpolasyonunu Docker çözer ve Openship'e somut değerler gider. Bu yüzden
**yerelde geçerli bir `.env` olmalı**.

> **MCP üzerinden gidiyorsan sync çalışmaz.** `POST /projects/:id/services/sync`
> ve `GET /projects/:id/services` MCP'de `service '*' not found` ile patlıyor.
> Servisler zaten kayıtlıysa sync'e gerek yok — her birini
> `PATCH /projects/:id/services/<serviceId>` ile güncelle. Servis ID'lerini
> `GET /projects/:id/services/containers` verir (o uç çalışıyor).

### Adım 3 — Build alanlarını düzelt (zorunlu)

`service sync` context'i **compose dosyasının dizinine** göre relatifleştirir,
build tarafı ise repo köküne göre bekler. Kök dışı compose'da ürettiği her değer
yanlış olur. Her build'li servis için:

```bash
curl -X PATCH https://<host>/api/projects/<projectId>/services/<serviceId> \
  -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' \
  -d '{"build": ".", "dockerfile": "<repo-koku-goreli-Dockerfile-yolu>"}'
```

Hazır imaj kullanan servislerin (`postgres`, `redis`, …) `build` alanı yoktur —
dokunma.

### Adım 4 — Environment

Bütün secret'ları doldur. Her biri ayrı:

```bash
openssl rand -hex 32
```

**Parola rotasyonunda iki tuzak var — bunları bilmeden rotate etme:**

1. `POSTGRES_PASSWORD`, `RABBITMQ_DEFAULT_PASS` ve benzerleri yalnızca **boş
   data dizininde** uygulanır. Çalışan bir stack'te env'i değiştirmek DB
   kullanıcısının parolasını değiştirmez; uygulama yeni parolayla bağlanmaya
   çalışır ve auth hatası alır. Gerçekten rotate etmek istiyorsan volume'u
   boşalt — ya da servise **yeni bir volume adı** ver (`pg-data` → `pg-data-v2`).
   İkincisi yıkıcı değil, eski volume durur.

2. Parola `command:` içinde geçiyorsa (örn. redis'in `--requirepass`'i), API o
   alanı **maskelemez ve PATCH'te korur**. Yalnızca env'i güncellersen servis
   eski parolayla ayağa kalkar ve sessizce auth hatası alırsın. `command` ve env
   birlikte güncellenmeli.

> Bonus: `command` maskelenmediği için mevcut parolayı okuyabilirsin —
> `GET /projects/:id/services/<id>` çıktısında düz metin görünür.

### Adım 5 — Public servisi aç

Yalnızca edge servisini (Caddy/nginx/vb.) işaretle:

```bash
openship service update <servis> --expose --exposed-port 80 --domain <label>
```

**Custom domain kullanıyorsan servise de yazmak zorundasın.** `project connect`
domain'i yalnızca proje seviyesine yazar; ön deploy kontrolü servise bakar ve
`Free subdomain requires Openship Cloud` der — self-hosted'da bu yanıltıcı bir
mesaj:

```bash
curl -X PATCH https://<host>/api/projects/<projectId>/services/<serviceId> \
  -H 'Authorization: Bearer <token>' -H 'Content-Type: application/json' \
  -d '{"domainType":"custom","customDomain":"<hostname>"}'
```

TLS'i Openship'in edge'i sonlandırır. Kendi edge container'ında sabit hostname
yazma — `:80` yaz ki eşleşmeyen Host header'ları 404 dönmesin.

### Adım 6 — Deploy ve takip

```bash
openship deploy --project <projectId>
```

`--watch` deployment ID döndürmüyor; ID'yi ayrıca al:

```bash
openship deployment list --project <projectId>
openship logs <deploymentId> --follow
```

---

## 3. Doğrulama — "deploy oldu" yeterli değil

Container'ın `running` olması çalıştığı anlamına gelmez. Sırayla:

```bash
# 1. Butun servisler ayakta mi? (one-shot job'lar 'stopped' olmali)
curl -s https://<host>/api/projects/<id>/services/containers | jq '.containers[] | {serviceName, status}'

# 2. Migration gercekten bitti mi?
#    Servis loglarinda ">> migrations complete" benzeri bir final satiri ara.

# 3. Edge -> backend zinciri
curl -sS -w '\n[%{http_code}]\n' https://<domain>/health

# 4. Gercek bir API ucu (hatali girdiyle 4xx bekle, 404 DEGIL)
curl -sS -X POST https://<domain>/api/auth/login \
  -H 'Content-Type: application/json' -d '{"email":"x@y.z","password":"wrong"}' \
  -w '\n[%{http_code}]\n'
```

4. adımda **404 alırsan yol yanlıştır**, servis bozuk değil. Rota kaydını
koddan doğrula — tahmin etme.

---

## 4. Belirti → sebep tablosu

| Belirti | Gerçek sebep | Çözüm |
|---|---|---|
| İlk `COPY`'de `file does not exist` | Context repo kökü, COPY yolları değil | §1.1 |
| `openship.json` hiç uygulanmıyor | Tanımsız bir alan tüm dosyayı geçersiz kılmış, sessizce yutulmuş | §1.4 |
| UI'da framework `docker-compose` seçilemiyor | Seçiciden kasıtlı çıkarılmış | Adım 1, API'den PATCH |
| Mevcut projeye `openship.json` eklemek işe yaramıyor | Config yalnızca import anında okunur | Alanları API'den PATCH'le |
| `requires Openship Cloud` (custom domain varken) | Domain projeye yazılmış, servise değil | Adım 5 |
| Migration DB hazır olmadan koşuyor | `condition:` düşürülüyor | Entrypoint'te bekleme döngüsü |
| Edge port çakışması | Servis `exposed` işaretlenmemiş | Adım 5 |
| Yeni parola yazdım, auth hatası | Parola boş data dizininde uygulanır / `command`'da kalmış | Adım 4 |
| `mkdir: Permission denied` (static deploy) | `/opt/openship/static` sunucuda yok | `sudo mkdir -p /opt/openship/static/{releases,.builds} && sudo chown -R $USER:$USER /opt/openship` |
| `project delete` 301 ile patlıyor | Bayat static imaj kaydı | Silme; mevcut projeye sync et |

---

## 5. Bir sonraki proje için kısa checklist

```
Repo hazirligi:
- [ ] Butun COPY yollari repo koku goreli (dogrulama script'i temiz)
- [ ] Compose context'leri repo koku, dockerfile yollari ona gore
- [ ] Kokte .dockerignore var (node_modules, .git, build ciktilari)
- [ ] Bind-mount yok (ya da sunucu yoluna gore yazilmis)
- [ ] Migration container DB'yi kendi bekliyor
- [ ] openship.json semaya uygun (ya da hic yok)

Openship:
- [ ] Proje --type services ile olusturuldu
- [ ] Servisler kayitli, build=".", dockerfile repo koku goreli
- [ ] Butun secret'lar dolu
- [ ] Edge servisi exposed + custom domain SERVISE yazildi
- [ ] Deploy ready, 10/10 servis beklenen durumda

Dogrulama:
- [ ] /health 200
- [ ] Gercek bir API ucu dogru statusu donuyor (404 degil)
- [ ] Migration log'u tamamlandigini soyluyor
- [ ] Frontend gercek build (mock degil)
```
