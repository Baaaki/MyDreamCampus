# Faz 6 — Süper admin, demo hesapları ve giriş paneli

## Başlamadan
- Oku: `new-backend/skills.md`, `frontend/skills.md`, `mobile/skills.md`.
- README §4'teki **K5** (demo admin e-postası) cevaplanmış olmalı.
- Faz 4 bitmiş olmalı (demo öğretmen ve öğrenci seed'den geliyor).

## Bağlam
- Şu an tek bir admin var. `SeedAdmin`
  (`services/auth-service/internal/service/auth_service.go` ≈750-790, sabit id
  `00000000-0000-0000-0000-000000000001`) onu `.env`'deki `ADMIN_EMAIL` /
  `ADMIN_INITIAL_PASSWORD` ile açıyor. Hesabın yalnız auth'ta kaydı var,
  staff'ta yok.
- Kodda "admin" rolü 21 yerde metin olarak, `RequireAdmin` / `RequireRole` ise
  32 yerde kontrol ediliyor. Bu yüzden **yeni bir rol eklenmeyecek**, bir
  bayrak eklenecek.
- Kullanıcının istediği model:
  - Üç public hesap (admin, öğretmen, öğrenci) giriş ekranının yanında
    e-posta ve şifresiyle yazar.
  - Süper admin görünmez ve yalnız kullanıcıya aittir.

## Hesap modeli
| Hesap | Nasıl oluşur | Özellik |
|---|---|---|
| Süper admin | `.env` `ADMIN_EMAIL` / `ADMIN_INITIAL_PASSWORD` (mevcut `SeedAdmin`) | `is_superadmin=true`; giriş ekranında görünmez |
| Demo admin | auth açılışta `DEMO_ADMIN_EMAIL` ile (şifre = e-posta) | `role=admin`, `is_demo=true`, zorunlu şifre değişimi kapalı |
| Demo öğretmen | seed: `ahmet.yilmaz@uni.edu.tr` | `is_demo=true`, zorunlu şifre değişimi kapalı |
| Demo öğrenci | seed: `zeynep.sahin@uni.edu.tr` | `is_demo=true`, zorunlu şifre değişimi kapalı |

## Görevler

- [x] **6.1 auth şeması ve token** (gemini ile yapıldı) — `d340727`
  - Migration: `auth.users`'a
    - `is_superadmin boolean not null default false`
    - `is_demo boolean not null default false`
  - `SeedAdmin`: `ADMIN_EMAIL` kullanıcısında `is_superadmin=true`. Mevcut
    kurulumlarda da çalışsın (idempotent UPDATE).
  - Token claim'i `super_admin` (bool, omitempty):
    - `new-backend/shared/platform/utils/jwt.go` `Claims`
    - auth `generateAccessToken` (≈817)
    - `JWTAuth` context'e `is_superadmin` yazsın.
  - `shared/platform/middleware/rbac.go`'ya `RequireSuperAdmin()` ekle.
  - Login yanıtında `user.is_superadmin` dönsün (frontend menü görünürlüğü
    için).
  - Testler.
  - **Commit:** `feat(auth): mark the super admin account`

- [x] **6.2 Demo hesapları** (gemini ile yapıldı) — `22f55f8`
  - Config (`shared/config/config.go`): `DEMO_MODE` (bool),
    `DEMO_ADMIN_EMAIL`, `DEMO_TEACHER_EMAIL`, `DEMO_STUDENT_EMAIL`.
    Compose'da `x-service-env`'e ve `.env.example`'a ekle.
  - auth açılışında `DEMO_MODE` açıksa demo admini oluştur (yoksa):
    `role=admin`, şifre = e-posta, `force_password_change=false`,
    `is_demo=true`.
  - `seed.sh`: öğretmen ve öğrenci demo hesaplarını `is_demo=true` ve
    `force_password_change=false` yap. Mevcut UPDATE'i genişlet (`seed.sh`
    ≈130).
  - `GET /api/auth/demo-accounts`:
    - Herkese açık; `DEMO_MODE` kapalıysa 404.
    - Yanıt: `[{role, label, email, password}]`. Yalnız var olan ve aktif demo
      hesapları döner.
    - Şifre "şifre = e-posta" kuralından hesaplanır, DB'den okunmaz.
  - **Commit:** `feat(auth): provision public demo accounts`

- [x] **6.3 Sistem hesaplarını koruma** (gemini ile yapıldı) — `a9764de`, `f4220be`, `16b0170`
  Admin paneli ziyaretçilere tamamen açık olduğu için bu korumalar şart. Aksi
  halde bir ziyaretçi:
  - demo öğrenciyi silip herkesi sabaha kadar dışarıda bırakabilir,
  - demo hesabın şifresini değiştirebilir,
  - süper admini pasifleştirebilir.

  **Korunan hesaplar:** süper admin + 3 demo hesabı. Config
  `PROTECTED_ACCOUNT_EMAILS`; compose'da `ADMIN_EMAIL` ve `DEMO_*_EMAIL`
  değerlerinden oluşsun.

  **Yapılacak:**
  - **auth:**
    - Demo hesaplarında `change-password`, `logout-all` ve kendi oturumu
      dışında bir oturumu silmek (`DELETE /sessions/:id`) → 403 "Demo
      hesabında bu işlem kapalı".
    - Korunan e-postalar için `request-password-reset` sessizce hiçbir şey
      yapmasın; yanıt her zamanki gibi aynı dönsün.
  - **staff / student:** korunan e-postaya sahip kaydı silme, pasifleştirme
    veya e-postasını değiştirme → 403. İlgili yerler: `DeleteStaff`,
    `UpdateStaff`, `DeleteStudent`, `UpdateStudent`; admin-staff'ta da varsa
    orası.
  - **auth consumer:** korunan bir hesap için deaktivasyon event'i gelirse
    uygulama, uyarı logla (savunma derinliği).
  - **Oturum listesi** (`GetUserSessions`, auth_service.go): demo
    hesaplarında, çağıranın kendi oturumu dışındakilerin IP'si maskelensin
    (örn. `85.105.x.x`) ve cihaz bilgisi kısaltılsın. Paylaşılan hesapta
    ziyaretçiler birbirinin IP'sini görmesin.
  - Testler.
  - **Commit:** `feat(auth): protect the super admin and demo accounts`,
    `feat(staff): ...`, `feat(student): ...`

- [x] **6.4 Giriş ekranı paneli ve demo şeridi** (gemini ile yapıldı) — `86c564a`, `5791997`
  - **Web** `frontend/src/pages/auth/login/index.tsx`:
    - Sağda (dar ekranda altta) bir "Demo hesapları" kartı; veriyi
      `GET /api/auth/demo-accounts`'tan alır.
    - Her hesap için rol, e-posta, şifre, "Kopyala" ve "Bu hesapla gir"
      (formu doldurup gönderir).
    - Uç 404 dönerse panel gizlenir.
  - **Uygulama geneli şerit** (demo modunda): "Bu bir demo. Yaptığınız
    değişiklikler her gece 04:00'te geri alınır. Gerçek kişisel bilgi
    girmeyin." Demo modunun açık olduğu bilgisi `demo-accounts` yanıtından
    alınır.
  - `pages/auth/forgot-password`: demo modunda "Demo ortamında e-posta
    gönderilmez" notu.
  - **Mobil** `mobile/app/(auth)/login.tsx`: aynı liste (dokununca form
    dolar).
  - **Commit:** `feat(frontend): list demo accounts on the login page`,
    `feat(mobile): list demo accounts on the login screen`

## Faz sonu
- README §5 yeşil olmalı.
- README §1 madde 6.
