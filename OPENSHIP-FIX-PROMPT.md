# Openship kaynak koduna verilecek prompt

`oblien/openship` deposunda açılan bir Claude Code oturumuna yapıştırılmak üzere
yazılmıştır. Aşağıdaki `---` çizgisinden sonrası promptun kendisidir.

> **Bu promptun tasarım ilkesi:** ajandan *onay* değil *çürütme* isteniyor. Aşağıdaki
> iddiaların bir kısmı yanlış olabilir — nitekim daha önce bir iddiamız (kusur C) kod
> okunduğunda yanlış çıktı. Yanlış bir iddiayla issue açmak, doğru iddiaların da
> güvenilirliğini düşürür. O yüzden prompt açıkça "hangi iddialarım yanlış" diye soruyor.

---

# Görev: bir kullanıcı bug raporunu kaynak kodda doğrula, çürüt ve düzelt

Bu depo Openship (self-hosted deployment platform). Ben bir kullanıcıyım; 0.4.8 ile
10 servisli bir compose stack'i deploy etmeye çalıştım ve bir dizi kusura çarptım.
Aşağıda gözlemlerim ve kök neden **hipotezlerim** var.

## Nasıl çalışmanı istiyorum

1. **Önce çürütmeye çalış.** Her hipotez için önce "bu yanlış olabilir mi?" diye sor.
   Yanlış olanı yanlış olarak raporla — bu benim için doğrulanmış olandan daha değerli,
   çünkü upstream'e yanlış iddia göndermekten kurtarır.
2. **Tahmin yürütme.** Her iddiayı `dosya:satır` ile kanıtla. Bulamadığın şeye "kodda
   bulamadım" de; makul görünen bir açıklama uydurma.
3. Gözlemlerim (log çıktıları, hata mesajları) **veridir** — onlar gerçekten oldu.
   Kök neden açıklamalarım **hipotezdir** — onlar yanlış olabilir.

---

## A. Ana bug — compose build context'i yok sayılıyor

### Gözlem (veri)

Compose dosyam repo kökünde değil (`new-backend/infrastructure/docker-compose.yml`),
servisler alt dizinlerden build ediliyordu. Beş imajın beşi de kendi Dockerfile'ının
**ilk COPY** satırında öldü:

```
caddy         COPY failed: stat package.json: file does not exist
migrate       COPY failed: stat monolith/internal/modules/auth/sql/migrations: file does not exist
monolith      COPY failed: stat shared/: file does not exist
notification  COPY failed: stat shared/: file does not exist
seed          COPY failed: stat seed.sh: file does not exist
```

Patlayan yolların hepsi repo kökünden bakınca yok, servisin kendi context'inden bakınca
var (`shared/` kökte yok, `new-backend/shared/` var).

**Dockerfile'lar bulunuyordu** — her biri `FROM`/`RUN` adımlarını çalıştırdı; `migrate`
`go install` ve `apk add postgresql-client` adımlarını bitirip sonra ilk `COPY`'de düştü.
Yani `build` + `dockerfile` birleşimi Dockerfile'ı doğru buluyor.

**En kritik veri — build log'u mekanizmayı açıkça yazıyor:**

```
Building compose service "caddy" from frontend using Dockerfile...
Building compose service "monolith" from new-backend using monolith/Dockerfile...
Preparing shared build context...
Cloning into '/tmp/openship-docker-context-tpe6R7'...
Shared build context ready (4.7 MB)
```

Servis başına context hazırlanmıyor; **tek bir ortak context** kuruluyor ve hepsine o
veriliyor. 4.7 MB = checkout kökünün tamamı.

Üç ayarın hiçbiri context'i değiştirmedi (hata mesajları **birebir** aynı kaldı):
servis `build`, servis `rootDirectory`, proje `rootDirectory`.

### Hipotez

`build` alanı Docker'a context olarak geçirilmiyor; yalnızca Dockerfile arama öneki
olarak tüketiliyor. Gerçekleşen: `docker build -f <checkoutRoot>/<build>/<dockerfile> <checkoutRoot>`.
Olması gereken: `docker build -f <context>/<dockerfile> <context>`.

### Neden bunun bir bug olduğunu düşünüyorum — üç bağımsız gerekçe

1. **Compose spesifikasyonu.** <https://docs.docker.com/reference/compose-file/build/>:
   "`context` defines either a path to a directory containing a Dockerfile" ve dockerfile
   için "A relative path is resolved from the build context."
2. **Openship'in kendi DB şeması aynı şeyi söylüyor.** `packages/db/src/schema/service.ts`
   civarında `build` alanının yorumu: *"Build context path relative to repo root"*,
   `dockerfile` için *"relative to build context"*. Şema sözleşmesi ile runtime çelişiyor.
3. **Cloud runtime'ın bunu doğru yaptığından şüpheleniyorum.** `cloud.ts` içinde context'in
   `rootDirectory` ile daraltıldığını (bir `join(contextDir, ...)` benzeri) gördüğüm
   söylendi, docker runtime'da böyle bir daraltma yok. **Bunu özellikle doğrula veya
   çürüt** — eğer doğruysa bu, "tasarım tercihi" savunmasını imkânsız kılar: aynı sistemde
   iki runtime aynı alanı farklı yorumluyor demektir.

### Senden istediklerim

1. Docker build çağrısında **context argümanının** nereden geldiğini bul. `dosya:satır`.
   Kaç ayrı build çağrı yolu var (SSH/remote, dockerode tekil, dockerode batch)? Hepsini
   listele.
2. `build` alanının context'e neden ulaşmadığını zincirle göster: DB satırı → servis
   nesnesi → build config → docker çağrısı. Hangi adımda düşüyor?
3. Cloud runtime ile docker runtime'ı **karşılaştır**. Aynı işi yapıyorlar mı? Değilse
   farkı `dosya:satır` ile göster.
4. Düzeltmeyi yaz.
   - **Dikkat:** `rootDirectory`'yi context yapmak muhtemelen yanlış çözüm. Monorepo
     alt-uygulamaları `rootDirectory` kullanıp bilerek kök context'e ihtiyaç duyuyor
     (kök lockfile + paylaşılan paketler). Context'i `rootDirectory` ile daraltırsan her
     monorepo build'ini kırarsın. **Önce bu riski doğrula**, sonra muhtemelen ayrı bir
     `buildContext` alanı gerektiğine karar ver — ama kendi incelemene göre karar ver,
     benim önerime göre değil.
5. **Geriye uyumluluk:** `build` boş veya `.` olan projelerin davranışı **birebir**
   aynı kalmalı. Bunu bir testle kanıtla.
6. Regresyon testi: compose dosyası alt dizinde + servis alt dizinden build ediliyor.
7. Context yoksa bugün hata `COPY failed: stat ...` oluyor — anlamsız. Önden kontrol +
   `Build context "X" does not exist` mesajı ekle.

---

## B. İkincil kusurlar

Her biri için: kök nedeni `dosya:satır` ile **doğrula veya çürüt**, düzeltmeyi yaz,
ayrı commit yap.

**B1. `openship deploy --watch` deployment ID alamıyor.**
Belirti: `✔ Deployment queued` / `No deployment id returned; nothing to watch.`
Hipotez: CLI `res.data?.deployment_id` okuyor, API `{ data: { deployment: { id } } }`
döndürüyor — anahtar uyuşmazlığı. `deployment_id` snake_case'i muhtemelen başka bir
endpoint'e (`POST /deployments/:id/build`) ait. Doğrula. **Bunu ilk düzelt** — tek satır
ve `--watch` çalışmadan ana bug'ı doğrulamak gereksiz zor.

**B2. `service sync` build context'ini yanlış tabana göre relatifleştiriyor.**
CLI tarafında `baseDir = path.dirname(<compose dosyasi>)` kullanılıyor ve mutlak context
buna göre relatifleştiriliyor; oysa `service create --build` yardımı "relative to repo
root" diyor. Kök dışı compose'da `..` / `../../frontend` gibi checkout dışına çıkan
değerler üretiliyor. Taban repo kökü olmalı; sonuç `..` ile başlıyorsa sync sessizce
yazmak yerine hata vermeli. **Ayrıca kontrol et:** API tarafındaki compose parser da ham
`build.context` değerini yazıyorsa, CLI ile UI import'u farklı taban kullanıyor demektir —
sözleşme iki yerde birden uygulanmalı.

**B3. `openship.json` parse hataları sessizce yutuluyor.**
Belirti: geçersiz bir `openship.json` hiçbir uyarı üretmeden yok sayılıyor, proje
auto-detection'a düşüyor.
> **Bu maddede daha önce yanlış teşhis koydum, tekrarlama.** İlk hipotezim "şemanın
> kökündeki `additionalProperties: false` dosyayı geçersiz kılıyor" idi. Yayınlanan
> şemada (`openship.io/openship.schema.json`) bu bayrak gerçekten var — ama o dosyanın
> editör autocomplete'i için olduğunu, deploy yolundaki validator'ın ayrı olduğunu ve
> orada bilinmeyen alanların uyarı olduğunu düşünüyorum. **Hangisinin doğru olduğunu
> koddan belirle.** Asıl şüphem: parse sonucu `errors`/`warnings` üretiyor ama çağıran
> taraf yalnızca `.config` alıp gerisini atıyor.

**B4. `framework` alanı config overlay'de uygulanmıyor.**
Overlay `packageManager`, `rootDirectory`, `buildImage`, `port`, `env`, `services` vb.
uyguluyor ama `framework`'ü uygulamıyor; o yalnızca metadata fold'undan akıyor ve o da
detection'ın seçtiği snapshot'ta çalıştığı için kök dışı compose'da hiç geçmiyor.
**Kontrol et:** `ProjectInfo` üzerinde alan adı `framework` mi yoksa `stack` mi?
`installCommand` / `buildCommand` / `startCommand` / `outputDirectory` da aynı boşlukta mı?

**B5. `/opt/openship/static` hedef sunucuda hiç oluşturulmuyor.**
`promoteBuildArtifact` `${workDir}/releases` için `mkdir` çağırıyor, SSH kullanıcısı
olarak. Stok `/opt` (root:root 0755) altında EACCES. Depoda bu dizini uzak sunucuda
oluşturan/chown'layan kod var mı? **Yoksa** sunucu bağlama akışına provisioning adımı
eklenmeli (elevated executor mekanizması zaten varsa onu kullan).

**B6. `project connect` domain'i exposed servise inmiyor.**
Projeye custom domain bağlıyken exposed servis `domainType: "free"` kalıyor; ön deploy
kontrolü servise bakıp `Free subdomain requires Openship Cloud` diyor. Self-hosted
kullanıcı için bu yanlış yönlendirme. Tek exposed servis varsa domain ona da yazılmalı.
Yazılamıyorsa mesaj "servise custom domain bağla" demeli.

**B7. `project delete` bayat static imajlarda patlıyor.**
`image /opt/openship/static/.bu: (HTTP code 301) unexpected` — dört kez, sonra proje
silinmiyor.
Hipotez (iki ayrı sorun): (a) static deploy'un `imageRef`'i bir imaj değil host dizin
yolu, ama cleanup manifest'i onu `type: "image"` sayıp Docker'dan silmeye çalışıyor;
daemon 301 dönüyor, kod bunu "not found" saymadığı için fırlatıyor. (b) `.bu` bir string
truncation — `/opt/openship/static/` tam 21 karakter, `.slice(0, 24)` benzeri bir kesme
olabilir. Ayrıca temizlenemeyen tek bir imaj tüm teardown'ı durdurmamalı.

**B8. `docker-compose` UI framework seçicisinden çıkarılmış.**
`Frameworks.tsx` → `EXCLUDED_STACKS`. Sonuç: dashboard tek başına compose projesi
oluşturamıyor ve bunu kullanıcıya söylemiyor, proje sessizce static pipeline'a düşüyor.
**Not:** Çözüm muhtemelen seçiciye eklemek değil — compose projesi framework seçimiyle
değil compose dosyası tespitiyle kurulmalı. Asıl kusur sessiz düşüş.

**B9. MCP sunucusunda koleksiyon uçları çalışmıyor.**
`GET /projects/:id/services` ve `POST /projects/:id/services/sync` MCP üzerinden
`{"error":"service '*' not found","code":"NOT_FOUND"}` döndürüyor. Servis bazlı uçlar
(`GET`/`PATCH .../services/<id>`) ve `.../services/containers` sorunsuz. Hipotez: yol
şablonu `/services/*` gibi bir kalıba eşlenip `*` `:serviceId` olarak gönderiliyor.
Etki: MCP üzerinden yeni bir compose projesi servislerle donatılamıyor.

**B10. `command` alanı secret'ı maskelemiyor (güvenlik).**
Servis kaydı dönerken `environment` içindeki her değer maskeleniyor (`"••••••••"`), ama
`command` düz metin dönüyor:

```json
"environment": { "REDIS_PASSWORD": "••••••••" },
"command": "redis-server /etc/redis.conf --requirepass 9Jiq21g3O2XZ..."
```

Compose'da `command:` içine `${SECRET}` yazmak olağan; interpolasyon sync anında
çözüldüğü için secret komut satırına gömülüyor.
**İkinci etki:** `command`'ı göndermeyen bir PATCH eski değeri koruyor. Yalnızca env'i
güncellersen servis eski parolayla ayağa kalkar ve sessizce auth hatası alırsın.
Maskeleme `command`/`commandArgv` alanlarını da kapsamalı.

---

## Çıktı formatı

Her kusur için:

- **Hipotez doğru mu?** — DOĞRULANDI / ÇÜRÜTÜLDÜ / KISMEN. Çürütüldüyse gerçek mekanizma ne?
- **Kök neden** — `dosya:satır` + kod alıntısı
- **Düzeltme** — uygulanan diff
- **Test** — eklenen/güncellenen test
- **Risk** — geriye uyumluluk etkisi, kimin build'i değişir

Sonda:

1. **Hangi iddialarım yanlıştı** — ayrı bir bölüm. Bunu atlamanı istemiyorum.
2. Ana bug için PR açıklaması (başlık + gövde), spec ve şema sözleşmesi referanslarıyla.
3. Önerilen merge sırası ve gerekçesi.
