# Openship ile Deploy Denemesi — Bulunan Kusurlar ve Durum

**Tarih:** 2026-07-31 · **Openship:** 0.4.8 (self-hosted, `deployMode: docker`, sunucu hedefi)
**Proje:** MyDreamCampus — 10 servisli compose stack, compose dosyası repo kökünde değil
(`new-backend/infrastructure/docker-compose.yml`), build context'leri compose dosyasına göreli.

Bu dosya bir günlük deneme sürecinin kaydı. Amaç ikili: (1) aynı duvara çarpan başkası
zaman kaybetmesin, (2) Openship'e issue/PR açarken elde somut kanıt olsun.

**Sonuç:** Stack 0.4.8 ile deploy **edilemedi**. Altı kusur aşıldı, yedincisi (build
context) repo tarafında çözülemez — Openship'in kendi kodunda düzeltilmesi gerekiyor.
Alternatif yol (`make deploy` + Cloudflare Tunnel) çalışıyor, bkz. [DEPLOY.md §A](DEPLOY.md).

---

## Neden başka projelerde çalışıyor da bizde çalışmıyor

Openship'in compose build'lerinde **context her zaman checkout kökü**. Bu varsayım şu
yaygın yerleşimde doğru sonuç verir:

```
repo/
  docker-compose.yml        # kökte
  Dockerfile
  services:
    api: { build: . }       # context = repo kökü = checkout kökü ✓
```

Tek servisli ya da "hepsi kökten build edilen" stack'lerde `context` ile checkout kökü
zaten çakışır, dolayısıyla context'in yok sayılması **görünmez** kalır.

Bizim yerleşimimizde çakışmıyor:

| Servis | compose'daki `context` | Gerçek context | Openship'in kullandığı |
|---|---|---|---|
| `monolith` | `..` | `new-backend/` | repo kökü ✗ |
| `notification` | `..` | `new-backend/` | repo kökü ✗ |
| `migrate` | `..` | `new-backend/` | repo kökü ✗ |
| `seed` | `./seed` | `new-backend/infrastructure/seed/` | repo kökü ✗ |
| `caddy` | `../../frontend` | `frontend/` | repo kökü ✗ |

Yani kusur "compose dosyası kökte değilse" veya "servisler alt dizinlerden build
ediliyorsa" ortaya çıkıyor — ikisi de compose'un tamamen normal kullanımı.

---

## Kusurlar

Aşağıdakiler yaşandıkları sırayla. "Aşıldı" = geçici çözüm bulundu, "Açık" = çözülemedi.

### 1. Geçersiz `openship.json` sessizce yutuluyor — Aşıldı

**Belirti:** `openship.json` hiç uygulanmıyor, hata veya uyarı yok, proje
auto-detection'a düşüyor.

**Kök neden:** Şemanın kökünde `additionalProperties: false` var. Tanımsız tek bir alan
(bizde `composePath`) dosyanın tamamını geçersiz kılıyor ve sessizce yok sayılıyor.

**Nasıl geçtik:** [openship.schema.json](https://openship.io/openship.schema.json) ile
alan adlarını doğrulayıp `composePath`'i attık.

**Olması gereken:** Geçersiz config build log'unda görünür bir uyarı üretmeli
("openship.json ignored: unknown key 'composePath' at line N"). Sessiz yutma, kullanıcıyı
saatlerce yanlış yere baktırıyor.

---

### 2. `docker-compose` framework'ü UI'dan seçilemiyor — Aşıldı

**Belirti:** Dashboard'da Framework → Değiştir listesinde Docker Compose yok. Proje ne
yapılırsa yapılsın `runtime: static` olarak deploy ediliyor.

**Kök neden:** `Frameworks.tsx` içindeki `EXCLUDED_STACKS` listesi `docker`,
`docker-compose` ve `unknown`'ı seçiciden kasıtlı olarak çıkarıyor.

**Nasıl geçtik:** CLI: `openship project create --type services`, ya da mevcut projeye
`PATCH /api/projects/<id>` ile `{"framework":"docker-compose"}`.

**Olması gereken:** Ya seçiciye eklenmeli, ya da repoda compose dosyası tespit edildiğinde
UI kullanıcıyı doğru akışa yönlendirmeli. Şu an dashboard tek başına compose projesi
oluşturamıyor ve bunu hiçbir yerde söylemiyor.

---

### 3. `openship.json`'daki `framework` alanı hiç uygulanmıyor — Aşıldı

**Belirti:** Aynı dosyadaki `rootDirectory` uygulanıyor (build log'unda görülüyor), ama
`framework` uygulanmıyor — UI hâlâ "Gin · Tespit Edildi" gösteriyor.

**Kök neden:** İki ayrı kod yolu var. `prepare.service.ts` içindeki `applyOpenshipOverlay`
`packageManager`, `rootDirectory`, `buildImage`, `port`, `productionMode`, `runtime`,
`domains`, `env`, `resources`, `services` alanlarını uyguluyor — **`framework` bu listede
yok**. `framework` yalnızca metadata fold'undan (`metadata/openship.ts` →
`applyFrameworkOverride`) akıyor, o da **detection'ın seçtiği alt dizinin** snapshot'ı
üzerinde çalışıyor. Compose dosyası kök dışındaysa detector başka bir dizin seçiyor, orada
`openship.json` olmadığı için `framework` düşüyor.

**Nasıl geçtik:** API'den PATCH.

**Olması gereken:** `framework` de overlay'e eklenmeli. Overlay repo kökündeki dosyayı
okuduğu için asimetri kendiliğinden kalkar.

---

### 4. `openship.json` yalnızca import anında okunuyor — Aşıldı

**Belirti:** Mevcut projeye `openship.json` eklemek/düzeltmek hiçbir şeyi değiştirmiyor.
Env bloğu dashboard'da hiç görünmüyor.

**Kök neden:** Config `prepare`/`scan` sırasında bir kez parse edilip DB'ye yazılıyor,
sonrasında servis kayıtları tek doğruluk kaynağı oluyor. Deploy sırasında
`resolveProjectInfo` yalnızca compose-drift uzlaştırması için, o da proje zaten compose
satırlarına sahipse çağrılıyor.

**Nasıl geçtik:** Projeyi silip yeniden oluşturmak — ama silme de patladı (kusur 9).

**Olması gereken:** `openship deploy --reconcile` gibi bir bayrak ya da en azından
"repodaki openship.json bu projede kullanılmıyor" uyarısı. Config-as-code vaadi, config'in
sonradan okunamamasıyla çelişiyor.

---

### 5. `service sync` build context'lerini yanlış tabana göre relatifleştiriyor — Aşıldı

**Belirti:** Sync sonrası servislerin `build` alanları checkout dizininin dışını gösteriyor.

**Kök neden:** `service.ts` içinde `baseDir = path.dirname(abs)` — yani **compose
dosyasının dizini**. `relativizeContext` mutlak context'i repo köküne değil bu tabana göre
relatifleştiriyor. Build tarafı ise bu değeri repo köküne göre bekliyor
(`service create --build` yardımı: *"Build context path (relative to repo root)"*). Ayrıca
`normalizeProjectRootDirectory` `..` segmentlerini temizlemiyor.

| Servis | Sync'in ürettiği | Doğrusu |
|---|---|---|
| `monolith` | `..` | `new-backend` |
| `notification` | `..` | `new-backend` |
| `migrate` | `..` | `new-backend` |
| `seed` | `seed` | `new-backend/infrastructure/seed` |
| `caddy` | `../../frontend` | `frontend` |

Compose dosyası repo kökündeyse iki taban çakıştığı için bu bug hiç görünmüyor.

**Nasıl geçtik:** Sync sonrası her servise `PATCH .../services/<id>` ile doğru değer.

**Olması gereken:** `relativizeContext` tabanı repo kökü olmalı; `..` içeren bir sonuç
üretiliyorsa sync hata vermeli.

---

### 6. `project connect` domain'i exposed servise inmiyor — Aşıldı

**Belirti:** Ön deploy kontrolü:
`Free subdomain "project-caddy.opsh.io" requires Openship Cloud`.
Oysa projeye custom domain bağlanmıştı.

**Kök neden:** `project connect` domain'i proje seviyesine yazıyor; exposed servis
`domainType: "free"`, `customDomain: null` olarak kalıyor ve ön kontrol servise bakıyor.

**Nasıl geçtik:**
`PATCH .../services/<id>` → `{"domainType":"custom","customDomain":"<host>"}`.

**Olması gereken:** `project connect` tek exposed servis varsa domain'i ona da yazmalı;
yazmıyorsa hata mesajı "servise custom domain bağla" demeli, "Cloud'a bağlan" değil —
self-hosted kullanıcı için ikincisi yanlış yönlendirme.

---

### 7. Compose build context'i tamamen yok sayılıyor — **AÇIK, tıkandığımız yer**

**Belirti:** Beş servisin beşi de kendi Dockerfile'ının **ilk COPY**'sinde patlıyor:

```
caddy        COPY failed: stat package.json: file does not exist
migrate      COPY failed: stat monolith/internal/modules/auth/sql/migrations: ...
monolith     COPY failed: stat shared/: file does not exist
notification COPY failed: stat shared/: file does not exist
seed         COPY failed: stat seed.sh: file does not exist
```

Patlayan yolların hepsi repo kökünden bakınca yok, doğru context'ten bakınca var.
Dockerfile'lar **bulunuyor** — her biri `FROM`/`RUN` adımlarını çalıştırıyor, `migrate`
`apk add postgresql-client`'ı bitiriyor, sonra ilk `COPY`'de düşüyor.

**Kök neden (kanıta dayalı çıkarım):** `build` ve `dockerfile` alanları yalnızca
Dockerfile'ın **yerini** belirlemek için birleştiriliyor; build **context**'i ise checkout
kökü olarak sabit kalıyor. `context` compose semantiğinde tam olarak "COPY'lerin tabanı"
demektir — bu davranış onu yok sayıyor.

**Denenen ve işe yaramayan üç şey:**

1. Servis `build` = doğru context (`new-backend`, `frontend`, …) → aynı hata
2. Servis `rootDirectory` = doğru context → aynı hata
3. Proje `rootDirectory` = `""` (kök) → aynı hata

Üçü de context'i etkilemedi; hata mesajları birebir aynı kaldı.

**Olması gereken:** Compose servisinin `build` değeri Docker'a **context** olarak
verilmeli, `dockerfile` da o context'e göre çözülmeli — yani `docker build -f
<context>/<dockerfile> <context>`. Bugünkü davranış `docker build -f
<root>/<build>/<dockerfile> <root>`.

**Repo tarafında geçici çözüm (uygulanmadı):** Beş Dockerfile'ın tüm COPY yollarını repo
köküne göre yeniden yazmak ve compose'daki `context:` satırlarını köke çekmek. Çalışır ama
her build'de tüm repo daemon'a gider ve compose'un standart semantiğinden sapar. Bu yüzden
tercih edilmedi.

---

### 8. `/opt/openship/static` hedef sunucuda oluşturulmuyor — Aşıldı

**Belirti:** Build başarılı, imaj hazır, sonra:
`Deploy failed: mkdir: Permission denied`

**Kök neden:** `promoteBuildArtifact` (`bare.ts`) `${workDir}/releases` için `mkdir`
çağırıyor ve bu SSH kullanıcısı olarak çalışıyor. `/opt/openship/static` repoda yalnızca
docker volume mount'u olarak tanımlı; uzak sunucuda bu dizini oluşturan veya sahipliğini
veren hiçbir kod yok. Stok Linux'ta `/opt` root:root 0755 → EACCES.

**Nasıl geçtik:**
```bash
sudo mkdir -p /opt/openship/static/{releases,.builds}
sudo chown -R $USER:$USER /opt/openship
```

**Olması gereken:** Sunucu bağlanırken (Servers → add) bu dizin oluşturulup chown
edilmeli, ya da kurulum dokümanında adım olarak yer almalı. Git geçmişinde bunu yapan
commit yok; en yakını `7b840a99 fix(core): classify EACCES as permission_denied` — sadece
hata sınıflandırması.

---

### 9. `project delete` bayat static imajlarda HTTP 301 ile patlıyor — Aşıldı (dolanarak)

**Belirti:**
```
image /opt/openship/static/.bu: (HTTP code 301) unexpected -
```
dört kez, ardından proje silinmiyor; sonraki `project create` "already exists" diyor.

**Not:** İmaj adı `/opt/openship/static/.bu` diye kesilmiş görünüyor — muhtemelen ayrı bir
string truncation bug'ı.

**Nasıl geçtik:** Silmeden devam ettik; compose servisleri eski projeye sync edildi ve
pipeline yine de compose yolundan gitti (servis satırlarının varlığı framework'ten bağımsız
olarak yeterli).

---

### 10. `openship deploy --watch` deployment ID döndürmüyor — Aşıldı

**Belirti:** `✔ Deployment queued` / `No deployment id returned; nothing to watch.`

**Nasıl geçtik:** `openship deployment list --project <id>` ile ID'yi alıp
`openship logs <id> --follow`.

---

## Kusurların özeti

| # | Kusur | Durum | Etki |
|---|---|---|---|
| 1 | Geçersiz `openship.json` sessizce yutuluyor | Aşıldı | Yanlış teşhise saatler |
| 2 | `docker-compose` UI'dan seçilemiyor | Aşıldı | Dashboard tek başına yetersiz |
| 3 | `openship.json` `framework` uygulanmıyor | Aşıldı | Config-as-code kısmen ölü |
| 4 | Config yalnızca import'ta okunuyor | Aşıldı | Mevcut proje düzeltilemiyor |
| 5 | `service sync` context tabanı yanlış | Aşıldı | Kök dışı compose'da her context bozuk |
| 6 | `project connect` servise inmiyor | Aşıldı | Yanlış yönlendiren hata mesajı |
| 7 | **Build context yok sayılıyor** | **AÇIK** | **Alt dizinden build eden her stack ölü** |
| 8 | `/opt/openship/static` oluşturulmuyor | Aşıldı | Static deploy sunucuda hiç çalışmıyor |
| 9 | `project delete` 301 ile patlıyor | Aşıldı | Proje silinemiyor |
| 10 | `deploy --watch` ID döndürmüyor | Aşıldı | Kozmetik |

Kusur 7 kritik: diğer dokuzu geçici çözümle aşılabiliyor, bu aşılamıyor.

---

## Ek: kaynak kodda araştırma promptu

Openship kaynak kodunu Claude Code ile açıp aşağıdaki promptu ver. Prompt kendi kendine
yeterli — bu dosyayı okumasına gerek yok.

Prompt: [`OPENSHIP-FIX-PROMPT.md`](OPENSHIP-FIX-PROMPT.md)
