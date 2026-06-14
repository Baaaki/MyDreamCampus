# MyDreamCampus Frontend

Next.js 16 + React 19 ile geliştirilmiş kampüs yönetim sistemi frontend uygulaması.

## 🚀 Teknoloji Stack

- **Framework**: Next.js 16.1.1
- **React**: 19.2.3
- **TypeScript**: 5.x
- **Styling**: Tailwind CSS 4.x
- **HTTP Client**: ky 1.14.2
- **QR Code**: qrcode.react 4.2.0
- **Date Utils**: date-fns 4.1.0
- **Package Manager**: Bun

## 📦 Kurulum

```bash
# Bağımlılıkları yükle
bun install

# Development server'ı başlat
bun run dev

# Production build
bun run build

# Production server'ı başlat
bun run start
```

## 🗂️ Proje Yapısı

```
frontend/
├── app/                          # Next.js App Router
│   ├── auth/                     # Auth Service sayfaları
│   │   ├── login/               # Login sayfası
│   │   ├── change-password/     # Şifre değiştirme
│   │   └── sessions/            # Oturum yönetimi
│   ├── staff/                    # Staff Service sayfaları
│   │   ├── page.tsx             # Personel listesi
│   │   └── profile/             # Personel profili
│   ├── student/                  # Student Service sayfaları
│   │   ├── page.tsx             # Öğrenci listesi
│   │   ├── profile/             # Öğrenci profili
│   │   └── advisor/             # Danışman atama
│   ├── catalog/                  # Course Catalog sayfaları
│   │   ├── page.tsx             # Ders kataloğu
│   │   └── schedule/            # Ders programı
│   ├── enrollment/               # Enrollment Service
│   │   └── page.tsx             # Ders kayıt
│   ├── attendance/               # Attendance Service
│   │   ├── teacher/             # Öğretmen yoklama (QR)
│   │   └── student/             # Öğrenci yoklama
│   ├── grades/                   # Grades Service
│   │   ├── student/             # Öğrenci transcript
│   │   └── teacher/             # Not girişi
│   ├── meal/                     # Meal Service
│   │   ├── student/             # Yemek rezervasyonu
│   │   └── admin/               # Admin QR kodları
│   ├── layout.tsx               # Root layout
│   └── page.tsx                 # Ana sayfa
├── lib/                          # Utility ve helper'lar
│   ├── api-client.ts            # ky HTTP client (her servis için)
│   ├── types.ts                 # TypeScript tipleri
│   └── constants.ts             # Sabitler (TIME_SLOTS, vb.)
└── .env.local                    # Environment variables
```

## 🌐 Servisler ve Endpoint'ler

### Auth Service (`/api/v1/auth`)
- `POST /login` - Giriş yap
- `POST /change-password` - Şifre değiştir
- `GET /sessions` - Aktif oturumları listele
- `DELETE /sessions/:id` - Oturum sonlandır

### Staff Service (`/api/v1/staff`)
- `GET /` - Personel listesi (pagination + filtering)
- `GET /me` - Kendi profil bilgim
- `GET /instructors` - Öğretim görevlileri listesi

### Student Service (`/api/v1/students`)
- `GET /` - Öğrenci listesi (pagination + filtering)
- `GET /me` - Kendi profil bilgim
- `PUT /:id/advisor` - Danışman atama

### Course Catalog Service (`/api/v1/catalog`)
- `GET /offerings` - Dönemlik ders listesi
- `GET /schedule/my` - Kendi ders programım
- `GET /teacher/my-courses` - Öğretmenin dersleri

### Enrollment Service (`/api/v1/enrollment`)
- `GET /my` - Kayıtlı derslerim
- `POST /enroll` - Derslere kayıt ol
- `DELETE /:id` - Dersi bırak

### Attendance Service (`/api/v1/attendance`)
- `POST /sessions` - Yoklama oturumu başlat (öğretmen)
- `GET /sessions/:id/refresh` - QR kod yenile
- `POST /mark` - Yoklama işaretle (öğrenci)
- `GET /student/summary` - Yoklama özeti

### Grades Service (`/api/v1/grades`)
- `GET /transcript/my` - Transkript
- `GET /course/:id/grades` - Dersin notları (öğretmen)
- `POST /course/:id/grades/bulk` - Toplu not girişi

### Meal Service (`/api/v1/meal`)
- `GET /cafeterias` - Yemekhane listesi
- `GET /reservations/my` - Rezervasyonlarım
- `POST /reservations` - Yeni rezervasyon
- `DELETE /reservations/:id` - Rezervasyon iptal
- `GET /qr/:cafeteria_id/:date/:meal_time` - QR kod (admin)

## 🔑 Önemli Sabitler

### Time Slots (Ders Saatleri)
```typescript
TIME_SLOTS = {
  1: "08:30-09:15",
  2: "09:25-10:10",
  3: "10:20-11:05",
  4: "11:15-12:00",
  5: "12:10-12:55",
  6: "13:00-13:45",
  7: "13:55-14:40",
  8: "14:50-15:35",
  9: "15:45-16:30"
}
```

### Yoklama Devamsızlık Limiti
```typescript
ATTENDANCE_ABSENCE_LIMIT = 3
```

### Yemek Rezervasyon Penceresi
```typescript
MEAL_RESERVATION_WINDOW = {
  START_DAY: 1,      // Pazartesi
  START_HOUR: 8,
  END_DAY: 5,        // Cuma
  END_HOUR: 13
}
```

## 🎨 UI/UX Özellikleri

### Tüm Componentler Client-Side
- Her component `"use client"` directive ile başlar
- Server Components kullanılmamıştır

### QR Kod Özellikleri
- **Attendance**: Öğretmen QR kodu her 15 saniyede yenilenir
- **Meal**: Günlük QR kodlar (öğle/akşam yemeği için ayrı)
- Kütüphane: `qrcode.react`

### HTTP Client (ky)
- Otomatik JWT token ekleme (localStorage'dan)
- 401 Unauthorized'da otomatik logout
- Retry logic (2 kez, belirli status code'lar için)
- Her servis için ayrı API client

### Responsive Design
- Tailwind CSS ile responsive grid'ler
- Mobile-first yaklaşım
- Tablo görünümlerinde horizontal scroll

## 🔐 Authentication Flow

1. Login (`/auth/login`)
2. Token localStorage'a kaydedilir
3. Her istekte `Authorization: Bearer <token>` header'ı eklenir
4. 401 hatası alınırsa otomatik logout
5. İlk girişte `force_password_change` kontrolü

## 📊 Veri Akışı

```
User Action
    ↓
Client Component (use client)
    ↓
ky HTTP Client (lib/api-client.ts)
    ↓
API Gateway (Traefik - http://localhost/api/...)
    ↓
Backend Microservices
    ↓
Response
    ↓
Component State Update
    ↓
UI Re-render
```

## 🛠️ Development

### Environment Variables
```bash
# .env.local
NEXT_PUBLIC_API_BASE_URL=http://localhost
```

### Port
- Development: `http://localhost:3000`
- Production: Traefik üzerinden yönlendirme

## 📝 Notlar

- Tüm sayfalar **client component** olarak geliştirilmiştir
- shadcn-ui componentleri kullanıldığında ekleyin: `bunx shadcn@latest add <component-name>`
- QR kod tarama için gerçek kamera yerine manuel input kullanılmıştır (demo amaçlı)
- Pagination her sayfada ayrı ayrı uygulanmıştır
- Error handling her component'te try-catch ile yapılmıştır

## 🚀 Production Deployment

```bash
# Build
bun run build

# Start
bun run start
```

## 📚 Kaynaklar

- [Next.js Documentation](https://nextjs.org/docs)
- [ky HTTP Client](https://github.com/sindresorhus/ky)
- [qrcode.react](https://github.com/zpao/qrcode.react)
- [date-fns](https://date-fns.org/)
