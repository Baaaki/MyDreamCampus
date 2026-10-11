# Faz 9 — Son kontrol

## Başlamadan
- Faz 8 bitmiş ve sistem Cloudflare üzerinden yayında olmalı.
- Bu fazı kullanıcı canlı ortamda yapar. Yürütücü AI maddeleri tek tek okur,
  kullanıcının sonucunu işaretler ve başarısız maddeler için hata kaydı açar
  (ilgili faz dosyasına `> ENGEL` notu veya yeni görev).

## Kontrol listesi

### Giriş
- [ ] Giriş kartında üç rol düğmesi var ("Öğrenci / Öğretmen / Admin Olarak
  Giriş"); düğme e-posta ve şifreyi (şifre = e-posta) doldurur, "Giriş Yap"
  ile girilir. Kartta 1 saatlik geri dönüş uyarısı görünüyor.
- [ ] Süper admin hesabı hiçbir listede görünmüyor.
- [ ] Demo şeridi görünüyor.

### Demo öğrenci
- [ ] Ders kaydı ve önkoşul senaryosu: kayıt sayfasında Bahar dönemine
  geçince "2. Sınıf Dersleri" altında CENG201 listeleniyor; seçilip
  gönderilince program danışman onayına düşüyor.
- [ ] Not sayfası ve transkript dolu.
- [ ] Yoklama sayfası (Yoklama menüsü) aktif dönemi (Güz) gösteriyor:
  CENG301 ve CENG350 için katılım sayıları ve haftalık oturumlar dolu.
- [ ] Öğretmenin açtığı yoklamanın QR'ı ile yoklama alınıyor (mobil
  uygulamadan); aynı QR ikinci kez okutulunca "zaten alındı" deniyor.
- [ ] Yemek rezervasyonu (web, Yemekhane → gelecek haftanın seçimi):
  - `4242 4242 4242 4242` → onaylandı;
  - `4000 0000 0000 0002` → reddedildi;
  - iptal (web: Yemekhane → Geçmiş Randevularım → "İptal Et"; mobil:
    Yemekhane) çalışıyor ve ödeme iade ediliyor. İptal, rezervasyon
    haftasından önceki Cuma 23:59'da kapanır: o saatten sonra satırda "İptal
    süresi doldu" yazar; hafta sonu gelecek haftanın iptali bu yüzden
    yoktur (`CANCEL_CUTOFF_PASSED`), bu bir hata değildir.
- [ ] Yemek QR'ı (mobil): admin "Yemekhane QR Kodları" sayfasındaki öğün
  QR'ı mobil uygulamanın tarayıcısıyla okutulunca o günün onaylı
  rezervasyonu kullanılıyor; ikinci okutma "Bu rezervasyon zaten
  kullanıldı" diye reddediliyor. QR yalnız öğün
  saatinde geçer (öğle 11:00–13:00, akşam 16:00–19:00); dışındaysan zaman
  makinesiyle rezervasyonu olan bir günün öğle saatine al, sonra "Sıfırla".

### Demo öğretmen
- [ ] Yoklama oturumu açılıyor, QR dönüyor, elle işaretleme çalışıyor.
- [ ] Not girişi yapılıyor; 87.5 girilince 87.5 olarak görünüyor.
- [ ] Danışman olarak onay bekleyen programı onaylayıp reddedebiliyor.

### Demo admin
- [ ] Ders, öğrenci ve öğretmen eklenebiliyor. Yeni öğrenci "şifre = e-posta"
  ile giriş yapınca zorunlu şifre değişimi ekranına düşüyor ve API başka
  isteklere izin vermiyor.
- [ ] Dönem ve periyot yönetimi dört türü de gösteriyor.
- [ ] Menü, kafeterya ve kapalı günler yönetilebiliyor.
- [ ] Denetim kaydı gerçek veri gösteriyor.
- [ ] Zaman makinesi:
  - ileri tarihe alınca tüm servislerin durumu aynı saati gösteriyor;
  - "Sıfırla" ile gerçek saate dönüyor. (K1 = Hayır: şerit ve 30 dk
    sonra kendiliğinden kapanma yok.)

### Korumalar
- [ ] Demo hesabında şifre değiştirme ve "tüm cihazlardan çık" kapalı:
  Oturumlar sayfasındaki "Tüm Cihazlardan Çık" düğmesi demo hesabında devre
  dışı ve nedenini yazıyor (API de 403 `DEMO_ACCOUNT_ACTION_FORBIDDEN`
  döner). Süper adminde onaydan sonra bütün oturumlar kapanıyor ve giriş
  sayfasına dönülüyor.
- [ ] Demo admin demo öğrenciyi silemiyor, pasifleştiremiyor ve e-postasını
  değiştiremiyor.
- [ ] Oturumlar sayfasında diğer ziyaretçilerin IP'leri maskeli.

### Süper admin ve kalıcı durum
- [ ] Süper admin ilk girişte şifre değiştirme ekranına düşüyor; değişimden
  sonra yeni şifreyle giriyor.
- [ ] Kalıcı Veri → "Düzenlemeye Başla" → başka bir tarayıcıda demo admin
  kayıt yapamıyor (K3, 503 "Sistem şu an yönetici tarafından
  güncelleniyor") ve şerit görüyor.
- [ ] Süper adminin eklediği kayıt "Kaydet ve Yayına Al" sonrası kalıcı;
  sürüm geçmişine yeni sürüm ekleniyor.
- [ ] "Ziyaretçi Değişikliklerini Şimdi Geri Al" → demo adminin eklediği
  kayıt gidiyor, süper adminin kaydı duruyor.
- [ ] "Bu Sürüme Dön" çalışıyor.
- [ ] Süper admin şifresi geri dönüşte korunuyor: şifreyi değiştirdikten
  sonra alınmış bir sürüme de, öncesine de ("Bu Sürüme Dön") dönülünce süper
  admin güncel şifresiyle giriyor ve şifre değiştirme ekranına düşmüyor.
- [ ] Saatlik geri dönüş: `.env`'de `REVERT_AFTER_MINUTES`'i geçici olarak
  `2` yap ve demo-ops'u yeniden oluştur; demo adminle bir kayıt ekle, ~2
  dakika sonra kaydın gittiğini gözlemle, değeri `60`'a geri al.
- [ ] (K4) `.../api/catalog/admin/ops` Cloudflare Access kodu istiyor.

### Altyapı
- [ ] Sunucunun LAN IP'sinden doğrudan erişim yok (tunnel modunda port
  kapalı).
- [ ] Servis loglarında ziyaretçilerin gerçek IP'leri görünüyor.
- [ ] UptimeRobot yeşil; auto-deploy yalnız CI'ı geçen commit'i deploy
  ediyor.
- [ ] Mobil APK: giriş paneli, rezervasyon ve kartla ödeme çalışıyor.

### Kod
- [x] `grep -rni mock frontend/src --include=*.ts --include=*.tsx | grep -v "\.test\."`
  boş dönüyor. (25.09 doğrulandı)
- [x] README §5 komutlarının hepsi yeşil. (25.09: backend 11 modül vet temiz /
  1160 test; frontend typecheck+lint+Prettier+build temiz / 138 test; mobil
  tsc temiz / 103 test)

## Faz sonu
- Tüm maddeler işaretliyse README §6'da Faz 9'u **Bitti** yap ve Oturum
  günlüğüne yaz.
- Push için kullanıcıdan izin iste (tek dal `main`).
