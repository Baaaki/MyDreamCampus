# Faz 1 — Güvenlik ve doğruluk düzeltmeleri

## Başlamadan
- Oku: `new-backend/skills.md`. Frontend ve mobil görevlerine (1.1, 1.4,
  1.10) gelince `frontend/skills.md` ve `mobile/skills.md`.
- Bağımlılık yok.

## Bağlam
23.09.2026 tarihli inceleme bulguları. Her görev bağımsız bir commit'tir.

## Görevler

- [ ] **1.1 İlk giriş şifre değişimi sunucuda zorunlu**
  - **Sorun:** `force_password_change` yalnız context'e yazılıyor
    (`new-backend/shared/platform/middleware/auth.go` ≈178) ama hiçbir yerde
    kontrol edilmiyor. `ErrForcePasswordChange`
    (`services/auth-service/internal/errors/errors.go:38`) hiç kullanılmıyor.
    Yeni kullanıcının ilk şifresi e-postası
    (`services/auth-service/internal/service/event_service.go` ≈92, ≈201;
    kullanıcı kararı gereği bu kalıyor). Değişim zorunlu olmazsa e-postayı
    bilen herkes tam yetkiyle girer.
  - **Yapılacak:**
    - `JWTAuth` içinde `claims.ForcePasswordChange == true` ise yalnız
      `POST /api/auth/change-password` ve `POST /api/auth/logout` geçsin.
      Diğer isteklere 403 dönsün:
      `{"error":"Devam etmek için şifrenizi değiştirmeniz gerekiyor","code":"FORCE_PASSWORD_CHANGE"}`.
    - Web `frontend/src/lib/api-client.ts` (`afterResponse`): 403 +
      `FORCE_PASSWORD_CHANGE` gelirse `/auth/change-password`'e yönlendir.
    - Mobil `mobile/services/api.ts`: aynı durumda şifre değiştirme ekranına
      yönlendir (`setOnUnauthorized` benzeri bir callback).
  - **Kabul:** Middleware birim testi: bayraklı token izinli yolda geçer,
    diğer yolda 403 alır; bayraksız token etkilenmez.
  - **Commit:** `fix(shared): enforce forced password change on the server`
    (+ `fix(frontend): ...`, `fix(mobile): ...`)

- [ ] **1.2 Yemekhane: çift kullanım ve çift iade**
  - **Sorun:** Kullanım ve iptal UPDATE'lerinde durum koşulu yok; kontrol Go
    tarafında UPDATE'ten önce yapılıyor. Aynı QR ile eşzamanlı iki tarama ikisi
    de başarılı oluyor; kullanım ve iptal aynı anda gelirse hem yemek hem iade
    alınıyor. Idempotency middleware DELETE isteklerini atlıyor.
  - **Yapılacak:**
    - `services/meal-service/internal/sql/queries/reservations.sql`:
      - `MarkReservationUsed` (≈71): `AND status = 'confirmed' AND is_used = false`.
      - `CancelReservation` (≈111): aynı koşul.
      - `make sqlc`.
    - Satır dönmezse (`pgx.ErrNoRows`):
      - kullanım (`reservation_service.go` `UseReservation` ≈728) →
        `ErrReservationAlreadyUsed`;
      - iptal (`repository/reservation_repository.go`
        `CancelReservationWithRefund` ≈203) → `ErrInvalidStatusForCancel`;
        transaction geri alınır, outbox'a yazılmaz, iade istenmez.
    - `new-backend/shared/platform/middleware/idempotency.go:90`:
      `http.MethodDelete` ekle. İptal route'u zaten middleware'i takıyor
      (`meal-service/internal/module.go:142`).
  - **Kabul:** Servis testleri: ikinci kullanım ve ikinci iptal hata verir.
    Middleware testi: aynı anahtarla gelen DELETE ilk yanıtı tekrar döner.
  - **Commit:** `fix(meal): make reservation use and cancel atomic`,
    `fix(shared): apply idempotency to DELETE requests`

- [ ] **1.3 Kapatılan veya e-postası değişen kullanıcının token'ları**
  - **Sorun:** `HandleUserDeactivated` (`auth-service/internal/service/event_service.go`
    ≈402) ve `HandleUserUpdated`'in e-posta dalı (≈336) DB'de `token_version`'ı
    artırıyor ama Redis'teki minimum versiyonu yazmıyor. `JWTAuth` yalnız
    Redis'e baktığı için kapatılan kullanıcı 15 dakika daha tam yetkili kalıyor.
  - **Yapılacak:**
    - EventService'e dar bir interface ile Redis enjekte et
      (`BlacklistAllUserTokens`). Örnek: `auth_service.go`'daki `authCache`;
      bağlantıyı `internal/module.go`'da kur.
    - `DeactivateUser` sorgusu `:one ... RETURNING token_version` olsun.
      `CheckEmailVersionSync`'in yeni versiyonu döndürüp döndürmediğini
      kontrol et. `make sqlc`.
    - Transaction commit'inden **sonra**
      `BlacklistAllUserTokens(userID, newVersion)` çağır. Redis hatası event'i
      başarısız yapmasın, sadece loglansın (DB tarafı zaten commit edildi).
  - **Kabul:** Sahte cache ile birim testi: çağrı doğru versiyonla yapılıyor.
  - **Commit:** `fix(auth): revoke access tokens on deactivation and email change`

- [ ] **1.4 Refresh token rotasyonu atomik**
  - **Sorun:** `RefreshAccessToken` (`auth_service.go` ≈466)
    `_ = s.sessionRepo.DeleteSession(ctx, jti)` ile hatayı yutuyor; oturum
    kontrolü ile silme arasında kilit yok. Aynı refresh token eşzamanlı iki
    kez kullanılabiliyor ve iki ayrı oturum doğuyor.
  - **Yapılacak:**
    - Yeni sorgu `DeleteSessionByJTI :execrows`. Silme ve yeni oturumu
      oluşturma aynı transaction'da olsun; 0 satır silinirse `ErrSessionNotFound`
      (401) dön.
    - Token ailesini iptal etme (tüm oturumları kapatma) **YAPMA**: iki sekme
      aynı anda refresh ederse kullanıcı her yerden atılır.
    - Web `frontend/src/lib/api-client.ts`: refresh başarısız olursa login'e
      atmadan önce orijinal isteği bir kez daha dene (başka bir sekme cookie'yi
      yenilemiş olabilir). O da 401 dönerse login'e yönlendir.
  - **Kabul:** Aynı refresh token'la ikinci çağrı `ErrSessionNotFound` alır.
  - **Commit:** `fix(auth): rotate refresh tokens atomically`,
    `fix(frontend): retry once before redirecting after a failed refresh`

- [ ] **1.5 Ondalıklı not ve itirazda 0 puan**
  - **Sorun:**
    - `services/grades-service/internal/service/grade_service.go:129` ve
      `:270` `fmt.Sprintf("%d", int(*score))` kullanıyor; kolon `DECIMAL(5,2)`.
      87.5 veritabanına 87 olarak yazılıyor, event'e ise 87.5 gidiyor.
    - `internal/dto/grade_dto.go:217`: float üzerinde `required` olduğu için 0
      boş sayılıyor; itirazda not 0'a çekilemiyor.
  - **Yapılacak:**
    - Kırpma yerine `strconv.FormatFloat(*score, 'f', 2, 64)`.
    - `NewScore` → `*float64` + `binding:"required,min=0,max=100"`; servisteki
      kullanımları güncelle.
  - **Kabul:** Test: 87.5 kaydedilip 87.50 okunuyor; itirazda 0 kabul ediliyor.
  - **Commit:** `fix(grades): keep decimal scores and allow zero on appeal`

- [ ] **1.6 Danışman onayında 403 ve kilit**
  - **Sorun:**
    - `services/enrollment-service/internal/service/enrollment_service_advisor.go:53`
      ve `:140` yetki hatasında `ErrUnauthorized` (401) dönüyor. Frontend
      401'de önce refresh deneyip sonra login'e atıyor; admin route'a izinli
      olduğu için onaylamaya çalışan admin oturumdan atılıyor.
    - Onay sorgusunda (`internal/sql/queries/enrollment_programs.sql:35`
      `UpdateProgramStatus`) `status='pending'` koşulu yok. Eşzamanlı iki onay,
      iki ayrı `enrollment.program.approved` event'i üretiyor.
  - **Yapılacak:**
    - Yetki hatasında `ErrForbidden` (403) dön.
    - `ApproveProgramWithEvent`
      (`internal/repository/enrollment_repository.go` ≈199): transaction içinde
      önce `LockPendingProgram` çağır; reject yolu bunu zaten yapıyor (≈260).
      Satır yoksa 409 dönen bir hata kullan (`ErrProgramNotPending` yoksa
      oluştur, mesajı Türkçe).
  - **Kabul:** Testler: yanlış danışman 403 alır; pending olmayan program 409
    alır.
  - **Commit:** `fix(enrollment): return 403 for advisor mismatch and lock program on approve`

- [ ] **1.7 DB bağlantı havuzu**
  - **Sorun:** `new-backend/shared/platform/database/database.go:20` servis
    başına `MaxConns = 25`. 8 DB'li servisle toplam 200 bağlantı ediyor,
    Postgres'in varsayılan `max_connections` değeri ise 100.
  - **Yapılacak:**
    - `shared/config/config.go`'ya `DB_MAX_CONNS` ekle (varsayılan 10) ve
      havuzda kullan. `MinConns` 2 olsun.
    - `.env.example`'a yorumlu satır ekle.
  - **Commit:** `fix(shared): cap database pool size`

- [ ] **1.8 Meal'de ortak hata biçimi**
  - **Sorun:** Hata yanıtları servislere göre farklı:
    - meal `{success:false, error:{code, message}}` sarmalayıcısını kullanıyor
      (`services/meal-service/internal/handler/meal_handler.go` ≈508
      `handleError`, `internal/dto/common_dto.go` `ErrorResponseWrapper`);
    - diğer servisler `{error:"<mesaj>", code:"<KOD>"}` kullanıyor.

    Frontend (`frontend/src/lib/api-error.ts`) meal biçimini tanımadığı için
    yemekhanede kullanıcı hep genel mesajı görüyor.
  - **Yapılacak:**
    - meal'i `{error, code}` biçimine geçir; `ErrorResponseWrapper` kullanan
      her yeri güncelle (`grep -rn ErrorResponseWrapper`).
    - Mobilde hata metnini okuyan yerleri (`mobile/services/*`) kontrol et.
  - **Kabul:** Handler testleri güncel; web'de yemekhane hatası gerçek mesajı
    gösteriyor.
  - **Commit:** `fix(meal): use the common error response shape`

- [ ] **1.9 JWT ve CSRF sertleştirme**
  - **Yapılacak:**
    - `jwt.WithValidMethods([]string{"HS256"})` ekle:
      - `shared/platform/utils/jwt.go` (≈144 ve ≈196'daki parse çağrıları);
      - `auth-service/internal/service/auth_service.go`: `parseRefreshToken`,
        `parseRefreshTokenWithoutValidation`, `blacklistAccessToken`
        (`blacklistAccessToken`'ın keyfunc'ında algoritma kontrolü hiç yok).
    - `shared/platform/middleware/csrf.go:23`: CSRF muafiyeti yalnız
      `Authorization` başlığı `"Bearer "` ile başlıyorsa uygulansın.
    - `shared/platform/middleware/auth.go` `OptionalJWTAuth` (≈190): hiçbir
      yerde kullanılmıyor ve iptal edilmiş token'lara bakmıyor; sil.
  - **Kabul:** Testler: HS512 imzalı token reddediliyor; `Basic` başlığı ve
    cookie'yle gelen POST, CSRF token'ı olmadan 403 alıyor.
  - **Commit:** `refactor(shared): harden JWT and CSRF checks`

- [ ] **1.10 Refresh geçici hata alınca oturum silinmesin**
  - **Sorun:** Web ve mobil, refresh isteği ağ hatası veya 5xx alınca da
    oturumu siliyor. Backend ise bu durumları bilerek "tekrar dene" diye
    ayırıyor.
  - **Yapılacak:**
    - Web `frontend/src/lib/api-client.ts` `refreshAccessToken`: yalnız
      401/403'te oturumu bitmiş say. Ağ hatası veya 5xx'te login'e yönlendirme,
      orijinal hatayı döndür.
    - Mobil `mobile/services/api.ts` `refreshOnce`:
      `error.response?.status === 401` değilse token'ları silme.
  - **Kabul:** `frontend/src/lib/api-client.test.ts` ve
    `mobile/services/api.test.ts` bu durumları kapsıyor.
  - **Commit:** `fix(frontend): keep the session on transient refresh failures`,
    `fix(mobile): keep the session on transient refresh failures`

## Faz sonu
- README §5'teki komutların hepsi yeşil olmalı.
- README §1 madde 6'yı uygula.
