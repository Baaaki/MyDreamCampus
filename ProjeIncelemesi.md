# MyDreamCampus — Gerçekçi Proje Değerlendirmesi

## Projenin Rakamsal Özeti

| Metrik | Değer |
|---|---|
| Elle yazılmış Go kodu (generated hariç) | ~45.600 satır |
| Frontend TS/TSX kodu (api-types hariç) | ~35.100 satır |
| SQL (migration + query) | ~3.500 satır |
| Toplam kaynak dosya | ~508 |
| Backend test dosyası | 78 |
| Frontend test dosyası | 8 |
| Backend modül sayısı | 9 (auth, staff, student, catalog, enrollment, attendance, grades, meal, payment) |
| Frontend sayfa/rol sayısı | 3 rol (admin/teacher/student), ~40+ sayfa |
| Altyapı bileşeni | PostgreSQL 18, Redis, RabbitMQ, MailHog, Grafana/Loki/Promtail |

---

## 1. Basit mi? — Hayır, Basit Değil

Bu projeyi "basit" diye nitelendirmek yanlış olur. Şu kapsamda bir sistem var:

- **9 ayrı domain modülü**, her biri kendi schema'sı, migration'ı, repository'si, service'i, handler'ı, DTO'su ve error tanımıyla
- **Modular monolith** mimarisi — modüller arası sınır korunmuş, in-process client ve RabbitMQ event'leri ile iletişim
- **Outbox pattern** ile event delivery garantisi
- **3 farklı rol** için ayrı frontend layout ve route yapısı
- **Event-driven** bildirim servisi (ayrı Go servisi, ayrı Postgres)
- **Ders kayıt** modülünde prerequisite check, schedule conflict detection, capacity check, advisory lock'lu transaction
- **JWT + Redis blacklist** ile session yönetimi, token versioning, fail-closed/fail-open stratejisi
- **Rate limiting** — IP bazlı, user bazlı, endpoint bazlı (login/password gibi hassas endpoint'lerde fail-closed)
- **CSRF koruması**, security headers, internal header stripping
- Observability stack: Grafana + Loki + Promtail

Bu kapsamdaki bir projeyi tek kişi geliştirmek — AI yardımıyla da olsa — ciddi bir iştir.

---

## 2. İyi Yapılmış Şeyler (Gerçekten İyi)

### Mimari Farkındalık
- Modular monolith seçimi doğru bir karar. Türkiye'de birçok şirkette "mikroservis yazıyoruz" diye başlayıp sonra yönetemez hale gelen projeler var. Sen bunu baştan doğru konumlandırmışsın.
- Her modülün `module.go` dosyasında tüm dependency wiring'in yapılması, `RegisterRoutes` ile route'ların dışarıya açılması — bu **clean, idiomatik Go** yaklaşımı.
- `internal/platform` altında paylaşılan altyapı (middleware, database, logger, errors) — modüllerin birbirine doğrudan bağımlı olmasını engellemiş.

### Güvenlik
- **Fail-closed** vs **fail-open** ayrımının bilinçli yapılması çok iyi. Login ve password change'de Redis erişilemezse 503 döndürmek, genel endpoint'lerde ise availability'ye öncelik vermek — bu tür detaylar 5+ yıl tecrübeli geliştiricilerin bile atladığı şeyler.
- Token versioning ile logout-all mekanizması, JTI bazlı blacklisting
- CSRF token'ın cookie'den okunup header'a eklenmesi
- `StripInternalHeaders` ile header spoofing koruması
- Argon2id hash seçimi (bcrypt değil)

### Event-Driven Tasarım
- Outbox pattern'in her modülde tekil bir `OutboxWorker` ile merkezi çalıştırılması
- Failed event retry mekanizması, max retry limiti
- Downstream queue binding'lerin uygulama başlangıcında declare edilmesi (mesaj kaybını önlemek için)

### Frontend
- `ky` ile yazılmış API client'ta single-flight refresh, CSRF token attachment, 401 auto-retry — bunlar production-grade detaylar
- Lazy loading ile code splitting
- Role-based routing ve AuthGuard

### Geliştirici Deneyimi
- `CLAUDE.md` dosyasındaki detaylı AI talimatları — hangi komutun nerede kullanılacağı, hangi dosyalara dokunulmayacağı, commit formatı, failure mode'lar
- `Makefile` ile tek komutla tüm stack'in ayağa kaldırılması
- `skills.md` dosyaları (frontend ve backend için ayrı)
- sqlc ile type-safe SQL

---

## 3. Zayıf Yönler ve Eksikler (Dürüstçe)

### AI İzi Belirgin
- Git history'de sadece **6 commit** var. Tüm proje neredeyse tek seferde atılmış. Bu, projenin iteratif geliştirilmediğini, büyük bloklar halinde AI'a yazdırıldığını gösteriyor. Bir mülakata girdiğinde bu, ilk sorulan sorulardan biri olacak.
- Bazı yerlerde AI'ın bıraktığı "plan section 5.5.2", "plan section 8" gibi referanslar var. Bunlar projenin AI ile birlikte oluşturulduğunun açık kanıtı.
- `checkPrerequisites` fonksiyonunda `passed := true` ile bypass — bu, AI'ın bitmemiş feature'ları stub'lamasının tipik bir örneği.

### Test Derinliği Yetersiz
- 78 backend test dosyası var ama çoğu DTO validation ve error code testi. **Service-level integration test** eksik.
- Frontend'de sadece 8 test dosyası — 35.000 satırlık bir frontend için bu çok az.
- E2E test dizini (`test/e2e`) var ama boş veya minimal görünüyor.
- Testlerin çoğu "yapılmış olsun" seviyesinde, edge case coverage zayıf.

### Pratikte Çalıştığına Dair Kanıt Yok
- README'deki ekran görüntüleri placeholder. Canlı demo, video kaydı, deployment süreci yok.
- CI/CD pipeline `.github` altında ama incelemem gereken içeriği göremedim — muhtemelen minimal veya yok.
- Projenin gerçekten uçtan uca çalışıp çalışmadığı belli değil.

### Bazı Teknik Sorunlar
- `main.go`'da `os.Setenv("JWT_SECRET", cfg.JWT.Secret)` — env variable'ı runtime'da set etmek anti-pattern. Config struct zaten var, neden env'e geri yazıyorsun?
- `json.Marshal` → `json.Unmarshal` ile DTO dönüşümü (enrollment service, line 83-85) — bu bir code smell. Aynı yapıyı paylaşan iki DTO varsa, ortak bir interface veya mapper kullanılmalı.
- Docker compose'da `postgres:postgres` credentials — development için sorun yok ama `.env.example` ile secrets management'ın gösterilmesi beklenirdi.
- Compiled binary (`main`, `monolith_bin`) git'e push edilmiş — 47MB'lık bir binary. Bu ciddi bir hata. `.gitignore`'a eklenmeli.

### Mimari Tutarsızlıklar
- `CLAUDE.md` mikroservis port tablosu veriyor (8001-8008) ama proje artık monolith. Eski dökümantasyon temizlenmemiş.
- Bazı modüller RabbitMQ consumer'ı `Bootstrap`'ta kuruyor (meal), bazıları `New`'da (auth). Tutarlılık yok.
- `frontend/skills.md` 19.000 satır — bu, AI'a verilen talimat dosyasının aşırı büyümesi anlamına geliyor. Normal bir projede bu kadar büyük bir instruction dosyası olmaz.

---

## 4. Türkiye Yazılım Sektörü Gerçeklerinde Bu Proje Nerede?

### Olumlu Taraf
Bu projeyi bir **portföy projesi** olarak değerlendirdiğimde, Türkiye'deki ortalama 1 yıllık junior developer'ın çok üstünde. Sebepleri:

- Türkiye'de junior'ların büyük çoğunluğu hala React + Express/Node.js ile basit CRUD yapıyor. Go seçmiş olman bile seni farklı bir kategoriye koyuyor.
- Modular monolith, outbox pattern, event-driven architecture — bu kavramları bilen ve uygulayan junior sayısı Türkiye'de çok az.
- Güvenlik detayları (rate limiting, CSRF, fail-closed) — bu seviyede güvenlik farkındalığı olan junior'ı sektörde çok az gördüm.

### Sert Gerçekler

**1. AI ile yazmak artık bir avantaj değil, bir norm.**
2026'da AI-assisted coding herkesin yaptığı bir şey. "AI ile yaptım" demek ne artı ne eksi. Önemli olan, **kodun her satırını anlayıp anlayamadığın**. Bir mülakatta sana outbox pattern'i neden seçtiğini, advisory lock'un ne yaptığını, fail-closed ile fail-open arasındaki trade-off'u sorduklarında ezberden değil deneyimden cevap verebiliyor musun?

**2. Bu proje henüz "production" değil.**
Hiç gerçek kullanıcısı olmayan bir proje, ne kadar iyi yazılırsa yazılsın, bir **öğrenme projesi** olarak değerlendirilir. Türkiye'deki şirketler (özellikle iyi olanlar) "ne yazdın" değil "ne çalıştırdın, ne deploy ettin, hangi sorunu çözdün" sorar.

**3. Kapsamın genişliği, derinliğin yetersizliğini gizliyor.**
9 modül var ama hiçbiri tam bitmiş değil:
- Enrollment'ta prerequisite check bypass edilmiş
- Payment modülü neredeyse boş
- Attendance'ın finalize helper'ları var ama uçtan uca akış belli değil
- Frontend'de test coverage çok düşük

Türkiye'de iyi bir technical interview'da bu hemen fark edilir. **3 modülü mükemmel yapıp teslim etmek, 9 modülü yarım yapmaktan çok daha değerli olurdu.**

**4. Sektördeki konumun: "Umut vadeden junior, ama henüz kanıtlamamış."**
- Bu proje bir **Getir, Trendyol, Hepsiburada** gibi şirketlere başvururken portföyde "Go bilen, mimari düşünebilen aday" izlenimi yaratır.
- Ama tek başına yetmez. Senden beklenen: bu projedeki herhangi bir modülü beyaz tahtada sıfırdan tasarlayabilmen, trade-off'ları açıklayabilmen.
- Ortalama bir yazılım evine (outsourcing/danışmanlık) bu projeyle başvursan muhtemelen "overqualified" görünürsün ama teknik derinliği sorgulandığında cevap veremezsen "AI yazdırdı" damgası yersin.

---

## 5. Net Tavsiyeler

1. **Git history'ni düzelt.** Projeyi sıfırdan, atomic commit'lerle yeniden oluştur. Her feature kendi branch'inde, PR açılmış gibi düşün. Bu tek başına, projenin algısını tamamen değiştirir.

2. **3 modülü derinleştir.** Auth + Enrollment + Grades'i uçtan uca, test coverage ile birlikte bitmiş hale getir. Geri kalanları "planned" olarak bırak.

3. **Canlı demo koy.** Railway, Fly.io veya bir VPS'e deploy et. README'ye canlı link ve ekran görüntüleri ekle. Türkiye'deki hiring manager'lar için "çalışan bir şey görmek" her şeyden değerli.

4. **Binary'leri git'ten kaldır.** `git filter-branch` veya `git-filter-repo` ile geçmişten temizle.

5. **AI referanslarını temizle.** "plan section X" gibi internal referansları koddan çıkar. CLAUDE.md ve skills.md'yi public repo'da bırakma — bunlar senin "kopya kağıdın" gibi duruyor.

6. **Bir blog yazısı yaz.** "Mikroservisten modular monolith'e neden geçtim" veya "Go'da outbox pattern implementasyonu" gibi bir yazı, portföy projesinden çok daha etkili olur.

---

## Sonuç

Bu proje, mimari farkındalık ve kapsam açısından 1 yıllık bir mezun için **ortalamanın üstünde**. Ama AI ile yapılmış olması, eksik test coverage, çalışan demo'nun yokluğu ve git history'nin düzlüğü — bunlar projenin değerini ciddi oranda düşürüyor.

Şu haliyle bu proje "Bu çocuk potansiyeli olan biri" dedirtir ama "Bu çocuk işi biliyor" dedirtmez. İkisi arasındaki fark, detaylarda ve derinlikte.
