# Openship kaynak koduna verilecek prompt

Bu dosyanın tamamı, `oblien/openship` deposunda açılan Claude Code oturumuna
yapıştırılmak üzere yazılmıştır. Aşağıdaki `---` çizgisinden sonrası promptun kendisidir.

---

# Görev: Compose build context'inin yok sayılması bug'ını bul ve düzelt

Bu depo Openship (self-hosted deployment platform). Ben bir kullanıcıyım ve 0.4.8 ile
compose stack'imi deploy edemiyorum. Kök nedeni **kaynak kodda doğrulamanı**, sonra
**düzeltmeni** ve **PR'a hazır hale getirmeni** istiyorum.

Tahmin yürütme. Her iddianı `dosya:satır` ile kanıtla. Bulamadığın şey için "kodda yok" de.

## Ana bug (öncelik 1)

Compose servisinin `build` alanı Docker'a **build context** olarak verilmiyor; context
her zaman checkout kökü kalıyor. `build` yalnızca Dockerfile'ın yerini bulmak için
kullanılıyor gibi görünüyor.

### Kanıt

Compose dosyam repo kökünde değil (`new-backend/infrastructure/docker-compose.yml`) ve
servisler alt dizinlerden build ediliyor. Servis kayıtlarına doğru context'leri yazdım:

| Servis | `build` (DB'de) | `dockerfile` (DB'de) |
|---|---|---|
| `monolith` | `new-backend` | `monolith/Dockerfile` |
| `notification` | `new-backend` | `services/notification/Dockerfile` |
| `migrate` | `new-backend` | `infrastructure/migrate/Dockerfile` |
| `seed` | `new-backend/infrastructure/seed` | `Dockerfile` |
| `caddy` | `frontend` | `Dockerfile` |

Deploy sonucu — beşi de kendi Dockerfile'ının **ilk COPY**'sinde patlıyor:

```
caddy         COPY failed: stat package.json: file does not exist
migrate       COPY failed: stat monolith/internal/modules/auth/sql/migrations: file does not exist
monolith      COPY failed: stat shared/: file does not exist
notification  COPY failed: stat shared/: file does not exist
seed          COPY failed: stat seed.sh: file does not exist
```

Kritik ayrıntılar:

- **Dockerfile'lar bulunuyor.** Her biri `FROM`/`RUN` adımlarını çalıştırıyor; `migrate`
  `go install` ve `apk add postgresql-client` adımlarını bitirip sonra ilk `COPY`'de
  düşüyor. Yani `build` + `dockerfile` birleşimi Dockerfile'ı doğru buluyor.
- **Patlayan yolların hepsi**, repo kökünden bakınca yok; ilgili servisin `build`
  dizininden bakınca var. Örnek: `shared/` diye bir dizin kökte yok, `new-backend/shared/`
  var.
- Bu üç ayarın **hiçbiri** context'i değiştirmedi (hata mesajları birebir aynı kaldı):
  1. servis `build` = doğru context
  2. servis `rootDirectory` = doğru context
  3. proje `rootDirectory` = `""` (kök)

### Beklenen davranış

`docker build -f <context>/<dockerfile> <context>` — yani `build` context olarak
verilmeli, `dockerfile` o context'e göre çözülmeli. Bugünkü davranış:
`docker build -f <checkoutRoot>/<build>/<dockerfile> <checkoutRoot>`.

### Neden çoğu projede görünmüyor

Compose dosyası repo kökündeyse ve servisler `build: .` kullanıyorsa, context ile checkout
kökü zaten çakışır; kusur görünmez kalır. Yalnızca compose dosyası alt dizindeyse veya
servisler alt dizinlerden build ediliyorsa ortaya çıkar — ikisi de compose'un normal
kullanımı.

### Senden istediklerim

1. `apps/api/src/modules/deployments/compose/build.service.ts` ve build context'i hazırlayan
   yardımcıları oku. Docker build çağrısında **context argümanının** nereden geldiğini bul
   ve `dosya:satır` ile göster.
2. `build` alanının context'e neden ulaşmadığını açıkla. Tar/stream context'i üreten kod
   varsa (`docker.ts` benzeri) onu da izle.
3. Düzeltmeyi yaz. Compose semantiğine uy: `context` COPY'lerin tabanıdır, `dockerfile`
   context'e görelidir.
4. Geriye uyumluluğu koru: `build` boş/`.` olan mevcut projelerin davranışı değişmemeli.
5. Regresyon testi ekle — compose dosyası alt dizinde, servis alt dizinden build ediliyor.
6. Commit mesajı ve kısa PR açıklaması yaz.

## İkincil buglar (öncelik 2 — ana bug bittikten sonra)

Her biri için: kök nedeni `dosya:satır` ile doğrula, düzeltmeyi yaz, ayrı commit yap.

1. **`service sync` build context'i yanlış tabana göre relatifleştiriyor.**
   `apps/cli/.../service.ts` içinde `baseDir = path.dirname(abs)` — compose dosyasının
   dizini. `relativizeContext` mutlak context'i repo köküne değil bu tabana göre
   relatifleştiriyor, sonuçta `..` ve `../../frontend` gibi checkout dışına çıkan değerler
   üretiyor. Oysa `service create --build` yardımı "relative to repo root" diyor. Taban repo
   kökü olmalı; `..` içeren sonuç üretiliyorsa sync hata vermeli.

2. **`openship.json`'daki `framework` alanı hiç uygulanmıyor.**
   `prepare.service.ts` → `applyOpenshipOverlay` şu alanları uyguluyor: `packageManager`,
   `rootDirectory`, `buildImage`, `productionPaths`, `port`, `productionMode`, `runtime`,
   `domains`, `env`, `resources`, `services`. `framework` listede yok. Yalnızca metadata
   fold'undan (`metadata/openship.ts` → `applyFrameworkOverride`) akıyor, o da detection'ın
   seçtiği alt dizinin snapshot'ında çalışıyor; kök dışı compose yerleşiminde hiç geçmiyor.
   `framework` overlay'e eklenmeli.

3. **Geçersiz `openship.json` sessizce yutuluyor.**
   Şemanın kökünde `additionalProperties: false` var; tanımsız tek bir alan dosyanın
   tamamını geçersiz kılıyor ve hiçbir uyarı üretmeden auto-detection'a düşülüyor. Build
   log'una görünür bir uyarı eklenmeli: hangi alan, hangi satır.

4. **`/opt/openship/static` hedef sunucuda hiç oluşturulmuyor.**
   `bare.ts` → `promoteBuildArtifact` `${workDir}/releases` için `mkdir` çağırıyor; bu SSH
   kullanıcısı olarak çalışıyor ve stok `/opt` (root:root 0755) altında EACCES alıyor.
   Depoda bu dizini uzak sunucuda oluşturan/chown'layan kod yok — yalnızca docker volume
   mount'u olarak tanımlı. Sunucu bağlama akışına bir provisioning adımı eklenmeli.

5. **`project connect` domain'i exposed servise inmiyor.**
   Projeye custom domain bağlıyken exposed servis `domainType: "free"` kalıyor ve ön deploy
   kontrolü `Free subdomain requires Openship Cloud` diyor. Self-hosted kullanıcı için bu
   mesaj yanlış yönlendirici. Tek exposed servis varsa domain ona da yazılmalı; yazılamıyorsa
   mesaj "servise custom domain bağla" demeli.

6. **`project delete` bayat static imajlarda patlıyor.**
   `image /opt/openship/static/.bu: (HTTP code 301) unexpected` — imaj adı da kesilmiş
   görünüyor (`.bu`), muhtemelen ayrı bir string truncation. Silme, temizlenemeyen bir imaj
   yüzünden tamamen durmamalı.

7. **`openship deploy --watch` deployment ID döndürmüyor.**
   `✔ Deployment queued` / `No deployment id returned; nothing to watch.` — `--watch`
   işlevsiz kalıyor.

8. **`docker-compose` framework'ü UI seçicisinden çıkarılmış.**
   `Frameworks.tsx` → `EXCLUDED_STACKS`. Bu kasıtlıysa dashboard compose projesi
   oluşturamıyor demektir ve bunu kullanıcıya söylemiyor; proje sessizce static pipeline'ına
   düşüyor. Ya seçiciye eklenmeli ya da compose dosyası tespit edildiğinde kullanıcı doğru
   akışa yönlendirilmeli.

## Çıktı formatı

Her bug için:

- **Kök neden** — `dosya:satır` + gerekirse kod alıntısı
- **Düzeltme** — uygulanan diff
- **Test** — eklenen/güncellenen test
- **Risk** — geriye uyumluluk etkisi

Sonda: ana bug için hazır bir PR açıklaması (başlık + gövde), ve düzeltmelerin hangi
sırayla merge edilmesi gerektiği.
