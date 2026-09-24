# MyDreamCampus — Herkese Açık Demo Yayın Planı

> Projeyi herkese açık canlı demo olarak yayına almak için yapılacak işlerin
> tek kaynağı bu dizindir. Her oturum buradan okur, kaldığı yerden devam eder
> ve yaptığı işi buraya işaretler.

**Plan durumu:** ONAYLANDI (23.09.2026)

---

## 1. Yürütücü AI için çalışma protokolü

1. Bu README'yi baştan sona oku. `CLAUDE.md` oturumla birlikte zaten yüklü;
   onun kuralları burada da geçerli.
2. Plan durumu "ONAYLANDI" değilse kullanıcıdan onay iste, başka iş yapma.
3. §6 İlerleme tablosunda durumu **Bekliyor** veya **Devam ediyor** olan ilk
   fazı bul. **Yalnızca o fazın dosyasını aç**; diğer faz dosyalarını okuma
   (context tasarrufu).
4. Faz dosyasındaki "Başlamadan" bölümünü uygula (zorunlu okumalar, açık
   kararların cevabı, bağımlı fazların bitmiş olması).
5. İşaretlenmemiş ilk görevden başla, sırayla ilerle. Her görevde:
   1. Uygula.
   2. §5 doğrulama komutlarından ilgili olanları çalıştır.
   3. Görevin kutusunu `- [x]` yap ve bu doküman değişikliğini görevin kod
      commit'ine dahil et.
   4. Atomic commit at (format: `CLAUDE.md` §7).
6. Faz bitince:
   - §6'da fazın durumunu **Bitti** yap.
   - Fazdaki görevlerin yanına commit hash'lerini ekle
     (`git log --oneline` ile bul), örnek: `- [x] 1.2 ... — \`abc1234\``.
   - §7 Oturum günlüğüne bir satır ekle.
   - Bunları tek bir `docs(infra): update demo launch progress` commit'i
     olarak at.
   - Kullanıcıya kısa Türkçe rapor ver ve **dur**. Kullanıcı "devam"
     demeden sonraki faza geçme.
7. Context büyüdüyse fazın ortasında da durabilirsin: mevcut görevi bitir,
   işaretle, günlüğe "kaldığı yer" yaz, kullanıcıya yeni pencere açmasını
   söyle.

### İşaretleme kuralları
- `- [ ]` bekliyor, `- [x]` bitti.
- Kısmen yapıldıysa işaretleme; görevin altına `> Not (GG.AA): ...` yaz.
- Engel çıkarsa görevin altına `> ENGEL (GG.AA): ...` yaz ve kullanıcıya
  sor. Sonraki görev bu göreve bağlı değilse ona geçebilirsin; bunu da not et.
- Plandan sapman gerekirse (görev yanlış, dosya taşınmış, daha iyi yol var)
  önce kullanıcıya sor, sonra görev metnini güncelle.

---

## 2. Genel kurallar

- **Dal:** `main` — projede tek dal var (`demo-readiness` 24.09.2026'da
  `main`'e alındı ve silindi). Push etmeden önce kullanıcıya sor.
- **landingpage/** asla stage edilmez. `git add -A` / `git add .` yok;
  dosyaları tek tek ekle (`CLAUDE.md` §14).
- **Dil:** Konuşma ve bu dokümanlar Türkçe; kod, dosya adı, commit İngilizce;
  UI metni Türkçe, log İngilizce.
- **Satır numaraları** 23.09.2026 itibarıyladır, kaymış olabilir. Dosyada
  sembol adıyla ara.
- **Docker** bu makinede `sudo` ister ve sandbox çalıştıramaz. Tam yığın
  doğrulaması için ya kullanıcıya kopyala-yapıştır komut ver ya da CI'daki
  `backend-e2e` job'una güven (dalı push etmek için izin iste).
- **Migration:** yazmak serbest, ÇALIŞTIRMAK kullanıcıya ait.
- `CLAUDE.md` §6 "SOR" listesi geçerli. §4'teki onaylı kapsam dışındaki her
  şey için (yeni kütüphane, planda olmayan şema/event değişikliği, route
  silme) kullanıcıya sor.

---

## 3. Hedef

Proje, CV için **herkese açık canlı demo** olarak kullanıcının ev sunucusunda
çalışacak ve **Cloudflare Tunnel** ile internete açılacak.

- Giriş ekranının yanında üç **public demo hesabı** (admin, öğretmen,
  öğrenci) e-posta ve şifresiyle yazar. Ziyaretçiler hesap açmadan bunlarla
  girer.
- Kullanıcının kendi **süper admin** hesabı var. Ona kimse erişemez; kalıcı
  veriyi yalnız o girer.
- Süper admin dışındaki herkes her şeyi değiştirebilir (admin paneli dahil,
  tamamen açık) ama bu değişiklikler **kalıcı değildir**: her gece 04:00'te
  sistem süper adminin en son kaydettiği duruma ("kalıcı durum") döner.
- **Zaman makinesi:** kapalıyken dünyadaki gerçek saat; açılınca tüm servisler
  aynı simüle saatle çalışır. Amaç, zamana bağlı davranışları ziyaretçilerin
  inceleyebilmesi.
- **Yemekhane ödemesi** sahte kartla (test kartları). Gerçek para yok,
  cüzdan/bakiye yok.
- Koddaki **tüm mock veri ve mock dallar kaldırılır**; içerik kaybolmaz,
  seed ile veritabanına girer.
- **Ertelendi:** kullanıcı kaydı (self-signup), gerçek e-posta (SMTP).

---

## 4. Kararlar

### Verilmiş kararlar
| Konu | Karar | Faz |
|---|---|---|
| Admin paneli | Ziyaretçilere tamamen açık; yalnız sistem hesapları korunur | 6 |
| Kalıcılık | Süper admin "düzenleme modu" → "kaydet" ile kalıcı durum alır; her gece 04:00'te ona dönülür | 7 |
| Ödeme | payment servisine kendi DB'si + outbox; sahte kartla onay | 3 |
| Yeni kullanıcının ilk şifresi | E-posta adresi kalır; ilk girişte değişim **sunucuda** zorunlu | 1 |
| Zaman makinesi | Tüm servislerde çalışır; ofset modeli, saat akar | 2 |
| Demo öğretmen / öğrenci | Mevcut seed hesapları: `ahmet.yilmaz@uni.edu.tr`, `zeynep.sahin@uni.edu.tr` (şifre = e-posta) | 4, 6 |
| Mock veriler | Koddan silinir, içerik seed'e taşınır | 4, 5 |
| Yayın | Ev sunucusu + Cloudflare Tunnel | 8 |

### Açık kararlar — ilgili faza başlamadan kullanıcıya sor, cevabı yaz
| # | Soru | Önerilen | Faz | Cevap |
|---|---|---|---|---|
| K1 | Zaman makinesi 30 dk sonra kendiliğinden kapansın ve açıkken herkese şerit gösterilsin mi? | Evet | 2 | Hayır — otomatik kapanma ve şerit yok (23.09) |
| K2 | Simüle saat en fazla ±2 yıl ofsetle sınırlansın mı? | Evet | 2 | Evet (23.09) |
| K3 | Süper admin düzenleme yaparken diğer herkese yazma kapansın mı? (Kapanmazsa o sıradaki ziyaretçi değişiklikleri de kalıcı olur.) | Evet | 7 | Evet (23.09) |
| K4 | "Kalıcı Veri" uçları Cloudflare Access (e-postaya tek kullanımlık kod) arkasına alınsın mı? | Evet | 7, 8 | Evet (23.09) |
| K5 | Demo admin e-postası | `demo.admin@uni.edu.tr` | 6 | `demo.admin@mydreamcampus.com` — alan adı `@mydreamcampus.com` (23.09) |

### Onaylı kapsam (`CLAUDE.md` §6 açısından)
Plan onaylandığında aşağıdakiler için ayrıca sorma:
- payment servisine yeni DB/rol (`payment` / `payment_svc`) ve tabloları.
- catalog'a `faculties` ve `departments` tabloları.
- `auth.users`'a `is_superadmin` ve `is_demo` kolonları.
- Yeni `demo-ops` altyapı konteyneri ve `docker-compose.tunnel.yml`.
- Faz dosyalarında adı geçen yeni endpoint'ler ve "Kalıcı Veri" sayfası.
- meal rezervasyon yanıtında `payment_url` → `payment_id` değişikliği (web ve
  mobil aynı fazda güncellenir).
- Zaman makinesi ve yazma kilidi için Redis anahtarları/kanalı (RabbitMQ event'i
  değil).

Yine de sorulacaklar: yeni kütüphane, migration çalıştırma, planda olmayan her
şema veya event payload değişikliği.

---

## 5. Doğrulama komutları

```bash
# Backend (her modül; tek servis için ilgili dizinde çalıştır)
cd new-backend && for d in shared services/*-service; do (cd "$d" && go vet ./... && go test ./...) || echo "FAIL: $d"; done

# sqlc (sorgu değiştiyse, ilgili servis kökünde)
make sqlc

# Frontend
cd frontend && bun run typecheck && bun run lint && bun run test && bun run build

# Mobil
cd mobile && npx tsc --noEmit && npx jest --ci
```

Başlangıç durumu (24.09.2026, Faz 0 sonu): 11 Go modülünde `go vet` temiz,
972 Go testi geçiyor (notification ve payment'ta test yok); frontend
`typecheck`, `lint` ve `build` temiz, 68 test geçiyor; mobil `tsc` temiz,
67 test geçiyor.

---

## 6. İlerleme

| Faz | Dosya | Konu | Durum |
|---|---|---|---|
| 0 | [phase-0-setup.md](phase-0-setup.md) | Hazırlık: onay, kararlar, dal | Bitti |
| 1 | [phase-1-security.md](phase-1-security.md) | Güvenlik ve doğruluk düzeltmeleri | Bitti |
| 2 | [phase-2-time-machine.md](phase-2-time-machine.md) | Tüm servislerde zaman makinesi | Bitti |
| 3 | [phase-3-payment.md](phase-3-payment.md) | Sahte kartla ödeme | Bekliyor |
| 4 | [phase-4-seed.md](phase-4-seed.md) | Referans veri ve seed'in yeniden yazımı | Bekliyor |
| 5 | [phase-5-mock-cleanup.md](phase-5-mock-cleanup.md) | Frontend'den mock temizliği | Bekliyor |
| 6 | [phase-6-accounts.md](phase-6-accounts.md) | Süper admin, demo hesapları, giriş paneli | Bekliyor |
| 7 | [phase-7-baseline.md](phase-7-baseline.md) | Kalıcı durum ve gece 04:00 geri dönüşü | Bekliyor |
| 8 | [phase-8-deploy.md](phase-8-deploy.md) | Ev sunucusu + Cloudflare Tunnel | Bekliyor |
| 9 | [phase-9-final-check.md](phase-9-final-check.md) | Son kontrol (tarayıcı + mobil) | Bekliyor |

Durum değerleri: **Bekliyor**, **Devam ediyor**, **Bitti**, **Engelli**.

**Bağımlılıklar**
- Faz 0 her şeyden önce.
- Faz 1, 2 ve 3 birbirinden bağımsız; sıra önerilendir.
- Faz 4, Faz 5'ten önce bitmeli: mock içerik silinmeden seed'e aktarılmalı.
- Faz 6, Faz 4'e bağlı (demo hesapları seed'le ilişkili).
- Faz 7, Faz 4 ve 6'ya bağlı (ilk kalıcı durum seed'den alınır; süper admin
  işareti gerekir).
- Faz 8, Faz 7'ye; Faz 9 en sona.

---

## 7. Oturum günlüğü

| Tarih | Faz / görev | Ne yapıldı | Kaldığı yer |
|---|---|---|---|
| 23.09.2026 | — | Plan yazıldı (inceleme oturumu) | Faz 0 bekliyor |
| 23–24.09.2026 | Faz 0 (0.1–0.5) | Plan onayı, K1–K5, dal; Go 1.27, Docker/CI, frontend (vite 8, react-router 8, lint 92→0) ve mobil (Expo SDK 57) güncellemeleri. 0.5: backend 11 modül vet temiz / 972 test; frontend typecheck+lint+build temiz / 68 test; mobil tsc temiz / 67 test — hepsi yeşil | Faz 1 bekliyor |
| 24.09.2026 | Faz 1 (1.1–1.10) | Güvenlik ve doğruluk düzeltmeleri, 16 commit. Planda olmayan ek düzeltmeler: e-posta değişim event'i hiç işlenemiyordu (1.3), iki sekme refresh yarışında cookie'ler siliniyordu (1.4), mobil yoklama ekranı hata metnini göstermiyordu (1.8). Backend 11 modül vet temiz / 997 test; frontend typecheck+lint+build temiz / 73 test; mobil tsc temiz / 76 test | Faz 2 bekliyor |
| 24.09.2026 | Faz 2 (2.1–2.8) | Zaman makinesi ofset modeline geçti ve Redis (`clock:state` + `clock:changed`, 10 sn yeniden okuma) ile 9 servise yayıldı; token/oturum/envelope gerçek saatte; SQL `NOW()` karşılaştırmaları servis saatine bağlandı; her serviste `admin/time/status`, catalog'da ±2 yıl sınırlı simulate/reset + audit; frontend sayfası backend'e bağlandı; e2e'ye eklendi. 2.6 K1 = Hayır nedeniyle yapılmadı. Planda olmayan ekler: catalog son tarih kontrolleri ve attendance/meal TTL-zamanlayıcı süreleri servis saatine alındı; CI'ı kıran iki eski sorun düzeltildi (auth TZ testi, Prettier). 16 commit (bu ilerleme commit'i dahil). Backend 11 modül vet temiz / 1033 test; frontend typecheck+lint+Prettier+build temiz / 81 test; mobil 76 test (tsc notu faz dosyasında). e2e push sonrası CI'da doğrulanacak | Faz 3 bekliyor |
