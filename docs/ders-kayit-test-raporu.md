# Ders Kayıt Senaryosu — Test Raporu

**Tarih:** 29 Ekim 2026 · **Dal:** `feat/mobile-backend-attendance`

## Nasıl test edildi

- Kullanıcının yığınına dokunulmadı. HEAD'in `git archive` kopyasından ayrı proje
  adıyla izole bir yığın kuruldu (`mdc-e2e`, Caddy `127.0.0.1:18080`), CI'daki
  `.env` ile, `SEED_DEMO=true`, `DEMO_MODE=true`.
- Seed senaryosu: yürüyen dönem **2026-2027 Güz** (kayıt penceresi 36 gün önce
  kapandı, dersler 5. haftada), açık pencere **2026-2027 Bahar** (CENG201/302/304/402/452
  ve Bilgisayar Bilimleri 1. sınıf açılışları).
- Öğrenci, danışman ve yönetici istekleri API üzerinden (Bearer +
  `X-Client-Type: mobile`; servisler header ile gelen istekte CSRF'yi atlıyor)
  gönderildi; olayların attendance ve grades'e yayılımı servis veritabanlarından
  okundu. Web sayfası (`frontend/src/pages/student/enrollment`) kod incelemesiyle
  tarandı.
- Kontenjan, çakışma ve danışmansız öğrenci durumları için **yalnız izole
  yığında** test amaçlı veri değişikliği yapıldı (kontenjanı 1'e çekmek, fazladan
  ders saati eklemek, danışmanı `NULL` yapmak); her birinin yanında belirtildi.

## Durum (29 Ekim 2026, ikinci tur)

1–10 düzeltildi ve izole yığında uçtan uca doğrulandı. 11'de kalan tek madde
(`Idempotency-Key` olmadan da kabul) bilinçli bir not. Bulgu 5'in çözümü:
pencere açıkken öğrenci onaylı programını yeniden gönderir, program danışman
onayına döner; o dönemde yoklama ya da not kaydı varsa `409
APPROVED_PROGRAM_IN_USE`.

## Özet

| # | Önem | Katman | Bulgu |
|---|---|---|---|
| 1 | Yüksek | enrollment + attendance | Kapanmış pencereden kalan program dönem ortasında onaylanınca öğrenci, onaydan önceki tüm oturumlardan devamsız sayılıyor |
| 2 | Yüksek | enrollment | Danışmanı olmayan / danışmanı pasifleştirilen öğrencinin bekleyen programını kimse göremiyor ve onaylayamıyor; program kontenjan tutmaya devam ediyor |
| 3 | Orta | web | "Yeniden Ders Kaydı Yap" bekleyen programı önce siliyor; backend'in atomik değiştirme yolu kullanılmıyor, koltuklar boşa çıkıyor |
| 4 | Orta | web | Programı geri çekme (DELETE) başarısız olsa da arayüz programı silinmiş gösteriyor |
| 5 | Orta | enrollment | Onaylanmış programda ekle-bırak yolu yok: öğrenci de, danışman da, yönetici de açık pencerede değişiklik yapamıyor |
| 6 | Orta | enrollment | Ret kaydında danışman adı hep boş; web "Reddeden: •" gösteriyor |
| 7 | Düşük | enrollment | Reddedilen derslerin `instructor` alanı hep boş |
| 8 | Düşük | web | Ret sonrası reddedilen dersleri seçili getiren kod yolu hiç çalışmıyor |
| 9 | Düşük | web | İkinci bir ret, sayfa yeniden yüklenmeden görünmeyebilir |
| 10 | Düşük | web | Onay/ret, sayfa yenilenmeden en fazla 5 dakika görünmüyor |
| 11 | Düşük | enrollment | Hata kodu ve yanıt tutarsızlıkları |

---

## 1. Geç onaylanan program, öğrenciyi otomatik devamsız bırakıyor

**Ne oluyor:** Onayın bir son tarihi yok ("approval is not bound to the window").
Seed'de Güz penceresinin son günlerinde gönderilip hâlâ onay bekleyen üç program var
(Ali Çelik, Oğuz Yıldırım, Zeynep Arslan). Program beklerken öğrenci attendance'ta
derse kayıtlı değil: QR okuttuğunda `NOT_ENROLLED` alıyor. Danışman 5. haftada
onaylayınca öğrenci, hiç okutamadığı 1–4. haftalar dahil dersin yapılmış tüm
oturumlarından devamsız sayılıyor.

**Tekrar üretme:**
1. `ali.celik` için `GET /api/attendance/my` → `courses: []` (kayıtlı değil).
2. `ahmet.yilmaz` olarak `GET /api/enrollment/advisor/pending-programs`, Ali'nin
   Güz programını `POST .../programs/{id}/approve`.
3. Birkaç saniye sonra Ali için `GET /api/attendance/my`.

**Kanıt:**
```
CENG301: theory={present_count: 0, absent_count: 7, total_sessions: 7, min_required: 5, passed: false}
CENG350: theory={present_count: 0, absent_count: 4, total_sessions: 4, min_required: 3, passed: false}
```

**Kod:** onayda dönem/tarih kontrolü yok —
[enrollment_service_advisor.go:19](../new-backend/services/enrollment-service/internal/service/enrollment_service_advisor.go#L19);
devam oranı dersin tüm oturumlarından hesaplanıyor —
[attendance_service.go:652](../new-backend/services/attendance-service/internal/service/attendance_service.go#L652).

**Öneri:** Danışman onayı için bir son tarih (ör. kayıt penceresi + N gün) ve
süresi geçen bekleyen programlar için bir son durum; ya da attendance'ın
öğrenciyi yalnız kayıt tarihinden sonraki oturumlarla değerlendirmesi
(`enrollment.program.approved` olayının zamanını saklayarak).

## 2. Danışmansız öğrencinin programı kimsenin listesinde değil

**Ne oluyor:** Onay ve ret, öğrencinin *şu anki* danışmanı çağıranla eşleşmezse 403
dönüyor. Öğretmen pasifleştirildiğinde (`staff.deactivated`) student-service
danışmanlarının `advisor_id`'sini `NULL` yapıyor. Bu öğrencilerin bekleyen
programları:
- eski danışmanın listesinden düşüyor ve onu onaylayamıyor (403),
- yöneticinin listesi hep boş (yönetici kimsenin danışmanı değil),
- yönetici de onaylayamıyor/reddedemiyor (403) — rota yöneticiye izin verdiği
  hâlde,
- program `pending` kaldıkça koltuk tutuyor (`CountActiveCourseEnrollments`
  `pending` + `approved` sayıyor).

Seed'de hiç danışmanı olmayan iki öğrenci de var (`kaan.aydin`, `defne.aslan`);
Bahar'da 1. sınıf BM açılışı olmadığı için bugün program gönderemiyorlar, ama
açılış eklendiği an aynı duruma düşerler.

**Tekrar üretme (test amaçlı veri değişikliğiyle):**
1. `ali.celik` Bahar için CENG302 gönderir → `201 pending`.
2. (test) `UPDATE student.students SET advisor_id = NULL WHERE id = '<ali>'` —
   pasifleştirme olayının yaptığının aynısı.
3. Ahmet'in ve yöneticinin bekleyen listeleri, ardından onay/ret denemeleri.

**Kanıt:**
```
in ahmet's list: ['Emir Çelik', 'Baran Koç']     ← Ali yok
in admin's list: 200 []
[403] admin approves ali: {"error": "Bu işlem için yetkiniz yok", "code": "FORBIDDEN"}
[403] admin rejects ali:  {"error": "Bu işlem için yetkiniz yok", "code": "FORBIDDEN"}
[403] former advisor approves ali: FORBIDDEN
[200] ali still pending
```
Danışmanı olan bir öğrencide de yönetici onayı 403: `[403] admin approves zeynep`.

**Kod:** rota yöneticiye açık —
[module.go:117](../new-backend/services/enrollment-service/internal/module.go#L117);
servis danışman eşleşmesi istiyor —
[enrollment_service_advisor.go:50](../new-backend/services/enrollment-service/internal/service/enrollment_service_advisor.go#L50),
[:144](../new-backend/services/enrollment-service/internal/service/enrollment_service_advisor.go#L144);
pasifleştirmede danışman siliniyor —
[students.sql:125](../new-backend/services/student-service/internal/sql/queries/students.sql#L125).

**Öneri:** Yöneticinin danışman kontrolünü atlaması (rota zaten izin veriyor) ve
yöneticiye danışmansız öğrencilerin bekleyen programlarını listelemek; ya da
pasifleştirmede bekleyen programları yeni danışmana devretmek.
`/api/students/orphaned` zaten var, bekleyen program sayısıyla birlikte
gösterilebilir.

## 3. "Yeniden Ders Kaydı Yap" programı önce siliyor

**Ne oluyor:** Bekleyen program afişindeki düğme önce `DELETE /programs` ile
programı geri çekiyor, öğrenci sonra dersleri sıfırdan seçip gönderiyor. Geri
çekme anında koltuklar serbest kalıyor; arada ders dolarsa ya da yeni seçim bir
kontrole takılırsa öğrencinin elinde hiçbir şey kalmıyor. Backend tam bunu önlemek
için bekleyen programı yeni gönderimle **aynı transaction'da** değiştiriyor, ama
web bu yolu kullanmıyor. Testte doğrulandı: bekleyen programı CENG304 olan Emir,
dolu CENG302 ile yeniden gönderince `409 COURSE_FULL` aldı ve programı
`pending:CENG304` olarak kaldı (kontenjan test için 1'e çekildi).

**Kod:** [index.tsx:313](../frontend/src/pages/student/enrollment/index.tsx#L313)
(`handleCancelEnrollment`) → [index.tsx:299](../frontend/src/pages/student/enrollment/index.tsx#L299).

**Öneri:** Bekleyen programda düğme yalnız seçimi düzenlenebilir yapsın
(`isReadOnly`'yi kaldırsın, mevcut dersleri seçili getirsin) ve gönderim doğrudan
`POST /programs` olsun; geri çekme ayrı ve açıkça "programı sil" diye kalsın.

## 4. Geri çekme hatası yutuluyor

**Ne oluyor:** `withdraw` mutasyonunun `onError`'ı da "Yeni ders kaydı
yapabilirsiniz" gösteriyor ve `onSettled` hata ne olursa olsun önbellekteki
programı `null` yapıyor. Silinecek program yokken (404) bu kasıtlı, ama
`403 ENROLLMENT_PERIOD_ENDED`, 5xx veya ağ hatasında da arayüz programı silinmiş
gösteriyor; sunucuda program `pending` duruyor ve danışman onu sonra onaylayabilir.

**Kod:** [index.tsx:299-310](../frontend/src/pages/student/enrollment/index.tsx#L299).

**Öneri:** Yalnız 404'ü "zaten yok" say; diğer hatalarda programı önbellekte
bırak ve hata mesajını göster (`apiErrorMessage`).

## 5. Onaylanmış programda değişiklik yapılamıyor

**Ne oluyor:** Pencere açıkken bile onaylanmış program kilitli:
```
[409] zeynep resubmit after approval: ALREADY_SUBMITTED
[403] zeynep cancel after approval:   "Onaylanmış ders kaydı iptal edilemez"
[403] ahmet approves zeynep again:    CANNOT_MODIFY_APPROVED
```
Danışmanın onayı geri alma, yöneticinin programı düzenleme yolu da yok. Danışman
pencerenin ilk gününde onaylarsa öğrenci pencere boyunca (seed'de 180 gün) ders
ekleyip bırakamıyor.

**Kod:** [enrollment_service.go:217-221](../new-backend/services/enrollment-service/internal/service/enrollment_service.go#L217),
[:431](../new-backend/services/enrollment-service/internal/service/enrollment_service.go#L431).

**Öneri:** Bu bilinçli bir kural olabilir; değilse pencere açıkken onaylanmış
programın değişikliğinin yeniden onaya düşmesi (ekle-bırak) ya da danışmana
"onayı geri al" eylemi.

## 6. Ret kaydında danışman adı boş

**Ne oluyor:** Handler danışman adını Gin bağlamındaki `fullname` anahtarından
okuyor; JWT ara katmanı bu anahtarı hiç koymuyor. Her ret kaydı
`advisor_fullname: ""` ile yazılıyor ve web "Reddeden: • 29.10.2026" gösteriyor.

**Kanıt:** `GET /api/enrollment/latest-rejection?semester=2026-2027-Spring`
(`emir.celik`) → `"advisor_fullname": ""`.

**Kod:** [enrollment_handler.go:450](../new-backend/services/enrollment-service/internal/handler/enrollment_handler.go#L450);
bağlama konanlar [auth.go:206-214](../new-backend/shared/platform/middleware/auth.go#L206);
gösterim [index.tsx:411](../frontend/src/pages/student/enrollment/index.tsx#L411).

**Öneri:** Adı staff-service'ten (`GET /internal/staff/:id`) ya da öğrencinin
kaydındaki `advisor_name`'den almak.

## 7. Reddedilen derslerde hoca adı boş

Ret anlık görüntüsü (`rejected_courses.courses[].instructor`) hiç doldurulmuyor;
kanıt: aynı yanıtta iki dersin `"instructor": ""`.
[enrollment_service_advisor.go:164](../new-backend/services/enrollment-service/internal/service/enrollment_service_advisor.go#L164).
Program dersleri zaten katalogdan okunabiliyor (`GetCoursesByIDs`).

## 8. Ret sonrası dersler seçili gelmiyor

Sayfa, "reddedilen programın dersleri seçili kalsın ki düzeltilip tekrar
gönderilsin" diye `status === "rejected"` olan programı bekliyor
([index.tsx:170-172](../frontend/src/pages/student/enrollment/index.tsx#L170),
[:201](../frontend/src/pages/student/enrollment/index.tsx#L201)). Backend ise
reddedilen programı siliyor (`my-enrollments` ret sonrası `programs: []`), bu
durum hiç oluşmuyor; öğrenci boş seçimle başlıyor. Ret kaydındaki
`rejected_courses` kimlikleri bunun için kullanılabilir.

## 9. İkinci ret görünmeyebilir (kod incelemesi)

Ret sorgusu `staleTime: Infinity`
([index.tsx:165](../frontend/src/pages/student/enrollment/index.tsx#L165)) ve
gönderimde önbelleğe `null` yazılıyor ([:282](../frontend/src/pages/student/enrollment/index.tsx#L282)).
Öğrenci yeniden gönderir, danışman tekrar reddeder ve öğrenci uygulama içinde
(tam sayfa yenilemeden, program sorgusu bayatladıktan sonra — global
`staleTime` 5 dk, `gcTime` 10 dk) sayfaya geri dönerse, program sorgusu boş
döner ama ret sorgusu önbellekteki `null`'u yeniden çekmez: ret afişi ve sebebi
görünmez, sayfa boş bir kayıt ekranı gösterir. Tam sayfa yenileme düzeltir.
Tarayıcıda doğrulanmadı.

## 10. Onay/ret sayfada kendiliğinden görünmüyor

Global `staleTime` 5 dk ve `refetchOnWindowFocus: false`
([query-provider.tsx:12](../frontend/src/components/providers/query-provider.tsx#L12));
onay bekleyen öğrenci sayfayı yenilemeden onayı ya da reddi 5 dakikaya kadar
görmüyor. Kayıt sayfası için kısa bir `refetchInterval` ya da odaklanınca
yenileme yeterli.

## 11. Küçük tutarsızlıklar

- Onaylanmış programı geri çekme `code: "FORBIDDEN"` dönüyor; aynı durumda
  onay `CANNOT_MODIFY_APPROVED` dönüyor.
- Boş ders listesi `{"error": "Gönderilen bilgiler geçersiz"}` — `code` yok.
- Onay yanıtındaki derslerde `schedule_sessions: []` ve hoca yok (liste ve
  gönderim yanıtları ikisini de taşıyor).
- `POST /programs` `Idempotency-Key` olmadan da kabul ediliyor; web ve mobil
  gönderdiği için sorun değil, yalnız not.

---

## Doğru çalıştığı doğrulananlar

- `open-semesters`: açık Bahar + programı olan kapalı Güz.
- `available-courses`: bölüm ve ≤ sınıf seviyesi; boş/yanlış dönem; askıdaki
  ve mezun öğrenci `403 STUDENT_DEACTIVATED`.
- Gönderim ve bekleyen programın atomik değiştirilmesi (DB'de tek program kalıyor;
  hatalı gönderim eskisini bozmuyor).
- Ön koşul (`PREREQUISITES_NOT_MET`), sınıf seviyesi (`INVALID_CLASS_LEVEL`),
  bölüm (`INVALID_DEPARTMENT`), tekrar eden ders, bilinmeyen ders, başka dönemin
  dersi (`COURSE_NOT_FOUND`), kapalı pencere (`ENROLLMENT_PERIOD_ENDED`).
- Idempotency: aynı anahtar + aynı gövde aynı yanıtı döndürüyor, farklı gövde
  `422 IDEMPOTENCY_KEY_REUSED`.
- Program çakışması `409 SCHEDULE_CONFLICT` (test için fazladan saat eklendi).
- Kontenjan `409 COURSE_FULL`; son koltuğa aynı anda iki gönderimde yalnız biri
  kazanıyor (kontenjan test için 1'e çekildi). Koltuk sayıları `pending` +
  `approved` ile doğru.
- Danışman onayı/reddi, yanlış danışman 403, ret kayıtları
  (`latest-rejection`, `my-rejections`), ret sonrası yeniden gönderim, bekleyen
  programı geri çekme.
- Onaylanan Bahar programının olaylarla yayılımı: attendance
  `enrollments_view` ve grades `student_course_registrations` Bahar satırları.

## Bu çalışmada düzeltilenler (kayıtla ilgili)

- `my-enrollments` derslerde hoca adını göndermiyordu (katalog yanıtında vardı,
  atılıyordu) — `e7b8b3e`.
- Mobil, yeni onaylanan **sonraki dönem** programını yürüyen dönem gibi
  gösteriyordu (`programs[0]` gönderim sırasına göre); artık yoklamanın döndüğü
  dönemin programı seçiliyor — `4559488`. Bahar programı onaylandıktan sonra
  emülatörde doğrulandı: profil ve ders programı "2026-2027 Güz".
- Mobil ders programı, `day_of_week`'i sayı sandığı için boş çıkıyordu —
  `397702a`.
