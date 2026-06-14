# MyDreamCampus

**MyDreamCampus**, öğrencilerin ders kayıtlarından yoklamalara, not girişlerinden kafeterya işlemlerine kadar tüm üniversite süreçlerini yöneten tam kapsamlı bir platformdur. Hem **Web** hem de **Mobil** uygulama olarak hizmet verir.

Bu proje, hem hızlı geliştirme yapılabilmesi hem de ileride kolayca ölçeklenebilmesi için **Modüler Monolit (Modular Monolith)** mimarisiyle sıfırdan, modern teknolojilerle geliştirilmiştir.

## Ekran Görüntüleri

> _Yer tutucular — proje içi ekran görüntülerini `docs/screenshots/` altına ekleyebilirsiniz._

| Web Arayüzü | Mobil Uygulama |
|-----|--------|
| ![Web dashboard](docs/screenshots/web-dashboard.png) | ![Mobile attendance](docs/screenshots/mobile-attendance.png) |

## Mimari ve Vizyon (Neden Bu Altyapı Seçildi?)

Proje, yönetimi ve dağıtımı zor olan parçalı mikroservis mimarisinden, daha sağlam ve yönetilebilir olan **Modüler Monolit** mimariye geçirilmiştir.

**1. İnsan Kaynakları ve Proje Yönetimi İçin Avantajları:**
- **Hızlı Geliştirme:** Tek bir kod tabanı sayesinde yeni özellikler çok daha hızlı eklenir, ürün pazara daha çabuk çıkar.
- **Düşük Maliyet:** Sunucu maliyetleri ve bakım eforu minimuma indirilmiştir. Sistem az kaynakla çok iş yapar.
- **Mobil ve Web Uyumu:** Tüm platformlar aynı güçlü arka ucu (backend) kullanır, böylece veri tutarsızlığı yaşanmaz.

**2. Yazılım Uzmanları İçin Teknik Detaylar (Geleceğe Hazır Yapı):**
- **Mantıksal İzolasyon:** Her modül (Auth, Öğrenci, Notlar) kendi paketi içinde tamamen izoledir (`internal/modules/`). "Spagetti kod" oluşumu engellenmiştir.
- **Veritabanı İzolasyonu:** Tek bir PostgreSQL veritabanı çalışsa da, her modülün kendi şeması (Schema) vardır. Modüller arası sıkı bağ (Foreign Key) kurulmamıştır.
- **Mikroservise Geçiş (Future-Proof):** Eğer ileride sistem çok büyürse (örn: Ders Kayıt dönemi yoğunluğu), bu mimari sayesinde istenilen modül birkaç saat içinde koparılıp ayrı bir **Mikroservis** olarak dışarı çıkartılabilir. Modüller arası iletişim halihazırda asenkron olarak **RabbitMQ** (Event-Driven) ile sağlanmaktadır.

## Kullanılan Modern Teknolojiler (Tech Stack)

Sistem tamamen sektör standartlarında, güncel ve yüksek performanslı araçlarla inşa edilmiştir:

*   **Arka Uç (Backend):** Go 1.26, Gin, PostgreSQL 18, RabbitMQ 4.0, Redis 7.2
*   **Ön Yüz (Web):** React 19, Vite, Tailwind CSS v4, shadcn/ui
*   **Mobil Uygulama:** React Native 0.81, Expo 54
*   **Bildirim Sistemi:** E-posta (MailHog ile test) ve Mobil Anlık Bildirim (Push Notification) altyapısı ayrı bir servis olarak asenkron çalışır.

## Yerel Ortamda Çalıştırma (Geliştiriciler İçin)

Projeyi kendi bilgisayarınızda test etmek oldukça basittir. 

**Gereksinimler:** Docker, Go 1.26+ ve Node 20+

```bash
# 1. Altyapıyı ayağa kaldırın (Veritabanı, Redis, RabbitMQ vb.)
cd new-backend/infrastructure
docker compose up -d

# 2. Ana Uygulamayı (Backend) başlatın
cd ../monolith
make run

# 3. Bildirim Servisini (E-posta ve Push) başlatın (Yeni bir terminalde)
cd ../services/notification
go run cmd/main.go

# 4. Web Arayüzünü başlatın (Yeni bir terminalde)
cd ../../../frontend
npm install
npm run dev
```

**Erişim Noktaları:**
- Web Arayüzü: `http://localhost:3000`
- Giden E-postaları Görme (MailHog): `http://localhost:8025`
- Backend API: `http://localhost:8080`
- RabbitMQ Yönetim Paneli: `http://localhost:15672`
