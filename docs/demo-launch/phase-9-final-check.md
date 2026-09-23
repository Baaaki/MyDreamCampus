# Faz 9 — Son kontrol

## Başlamadan
- Faz 8 bitmiş ve sistem Cloudflare üzerinden yayında olmalı.
- Bu fazı kullanıcı canlı ortamda yapar. Yürütücü AI maddeleri tek tek okur,
  kullanıcının sonucunu işaretler ve başarısız maddeler için hata kaydı açar
  (ilgili faz dosyasına `> ENGEL` notu veya yeni görev).

## Kontrol listesi

### Giriş
- [ ] Giriş ekranında üç demo hesabı (admin, öğretmen, öğrenci) e-posta ve
  şifresiyle görünüyor; "Bu hesapla gir" çalışıyor.
- [ ] Süper admin hesabı hiçbir listede görünmüyor.
- [ ] Demo şeridi görünüyor.

### Demo öğrenci
- [ ] Ders kaydı ve önkoşul senaryosu çalışıyor.
- [ ] Not sayfası ve transkript dolu.
- [ ] Öğretmenin açtığı yoklamanın QR'ı ile yoklama alınıyor (mobil
  uygulamadan).
- [ ] Yemek rezervasyonu:
  - `4242 4242 4242 4242` → onaylandı;
  - `4000 0000 0000 0002` → reddedildi;
  - iptal çalışıyor.
- [ ] Onaylı rezervasyonun QR ile kullanımı çalışıyor; ikinci kullanım
  reddediliyor.

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
  - şerit görünüyor;
  - 30 dk sonra (K1) ya da "Sıfırla" ile gerçek saate dönüyor.

### Korumalar
- [ ] Demo hesabında şifre değiştirme ve "tüm cihazlardan çık" kapalı.
- [ ] Demo admin demo öğrenciyi silemiyor, pasifleştiremiyor ve e-postasını
  değiştiremiyor.
- [ ] Oturumlar sayfasında diğer ziyaretçilerin IP'leri maskeli.

### Süper admin ve kalıcı durum
- [ ] "Düzenlemeye başla" → başka bir tarayıcıda demo admin kayıt yapamıyor
  (K3) ve şerit görüyor.
- [ ] Süper adminin eklediği kayıt "Kaydet ve yayına al" sonrası kalıcı.
- [ ] "Bugünkü değişiklikleri şimdi geri al" → demo adminin eklediği kayıt
  gidiyor, süper adminin kaydı duruyor.
- [ ] "Bu sürüme dön" çalışıyor.
- [ ] Gece geri dönüşü: `NIGHTLY_RESET_AT`'i geçici olarak 5 dakika sonraya
  ayarla, gözlemle, eski değerine geri al.
- [ ] (K4) `.../api/catalog/admin/ops` Cloudflare Access kodu istiyor.

### Altyapı
- [ ] Sunucunun LAN IP'sinden doğrudan erişim yok (tunnel modunda port
  kapalı).
- [ ] Servis loglarında ziyaretçilerin gerçek IP'leri görünüyor.
- [ ] UptimeRobot yeşil; auto-deploy yalnız CI'ı geçen commit'i deploy
  ediyor.
- [ ] Mobil APK: giriş paneli, rezervasyon ve kartla ödeme çalışıyor.

### Kod
- [ ] `grep -rni mock frontend/src --include=*.ts --include=*.tsx | grep -v "\.test\."`
  boş dönüyor.
- [ ] README §5 komutlarının hepsi yeşil.

## Faz sonu
- Tüm maddeler işaretliyse README §6'da Faz 9'u **Bitti** yap ve Oturum
  günlüğüne yaz.
- Kullanıcıya dalı `main`'e birleştirmek için PR açmayı öner (izin iste).
