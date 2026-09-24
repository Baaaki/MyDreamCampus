# Faz 5 — Frontend'den mock temizliği

## Başlamadan
- Oku: `frontend/skills.md`.
- **Faz 4 bitmiş olmalı:** `GET /api/catalog/faculties` ucu çalışıyor ve
  mock içeriği seed'e taşındı.

## Hedef
`frontend/src` içinde (test dosyaları hariç) şunların hiçbiri kalmaz:
`mock` kelimesi, mock veri, "Test Modu" anahtarı, sahte fallback. Test
altyapısındaki `vi.mock` ve `__mocks__` uygulama kodu değildir; kalır.

## Görevler

- [x] **5.1 Fakülte listesi API'den** (gemini ile yapıldı)
  - `lib/services/catalog-service.ts`: `getFaculties()`. Hook
    `useFaculties()` (TanStack Query, `staleTime: Infinity`).
  - `mockFaculties` kullanan 12 dosyayı hook'a geçir; yüklenme ve hata
    durumlarını göster:
    - `components/course-hierarchy-view.tsx`
    - `pages/admin/catalog/index.tsx`, `add/index.tsx`, `edit/index.tsx`
    - `pages/admin/grades/index.tsx`
    - `pages/admin/attendance/index.tsx`
    - `pages/admin/semester-courses/index.tsx`, `review/index.tsx`
    - `pages/admin/staff/index.tsx`, `personel-details/index.tsx`
    - `pages/admin/students/index.tsx`, `advisors/index.tsx`,
      `student/index.tsx`
  - **Commit:** `refactor(frontend): load faculties from the catalog service`

- [x] **5.2 Admin sayfalarında "Test Modu"** (gemini ile yapıldı)
  - `pages/admin/grades/index.tsx`: `useMockData` (≈89-104, 121, 187, 212),
    anahtar (≈579-590).
  - `pages/admin/attendance/index.tsx`: ≈71-201.
  - `pages/admin/attendance/sessionId/index.tsx`: ≈39-90;
    `generateMockSessionRecords` ve `markMockStudentPresent`.
  - **Commit:** `refactor(frontend): drop the mock data toggles from admin pages`


- [x] **5.3 Diğer sahte dallar** (gemini ile yapıldı)
  - `pages/student/grades/index.tsx` (≈20, 41, 163): `VITE_USE_MOCK_API`
    dalı. `frontend/.env.example`'daki `VITE_USE_MOCK_API=true` satırını sil
    (örnek dosyayı kopyalayan sahte not görüyor).
  - `pages/admin/meal/cafeterias/index.tsx` (≈80-163, 268): gömülü sahte
    kafeterya listesi yerine boş durum mesajı.
  - `pages/student/cafeteria/menu/index.tsx` (≈219, 394): `isUsingMockData`
    → `hasNoMenu`; metin "Bu ay için menü yayınlanmadı".
  - **Commit:** `refactor(frontend): remove inline mock fallbacks`

- [x] **5.4 Dönem yönetimi** (gemini ile yapıldı)
  - **Backend:** catalog'un `/api/catalog/admin/periods` uçları yalnız
    `catalog` türünü kapsıyor
    (`new-backend/shared/platform/repository/simple_period_repository.go`,
    `scope`). `period_type` parametresi veya alanıyla dört türü (`catalog`,
    `enrollment`, `grading`, `attendance`) yönetebilir hale getir. Tür
    değişince enrollment, grades ve attendance projeksiyonlarını besleyen
    event'ler akmaya devam etmeli; catalog'un dağıtım mantığını incele.
  - **Frontend:**
    - `lib/services/system-service.ts` (≈117-190): `listGradesPeriods` ve
      `listSimplePeriods` şu an grades, enrollment ve attendance servislerine
      gidip 404 alıyor; bunun yerine catalog'a `type` parametresiyle gitsin.
    - `pages/admin/system/semesters/index.tsx`: `buildMockData` (≈80-180),
      önizleme modu (≈187-421) ve ≈630'daki kullanım silinsin.
    - `semesters/new/index.tsx` (≈179-180) güncellensin.
  - **Commit:** `fix(catalog): manage every period type from the catalog`,
    `refactor(frontend): read periods from the catalog`

- [x] **5.5 Denetim kaydı** (gemini ile yapıldı)
  - `system-service.ts` `listAuditLog` (≈260) sahte veri dönüyor. Bunun
    yerine `GET /api/catalog/admin/audit-log` çağrılsın. Filtre parametrelerini
    ve yanıt biçimini
    `new-backend/services/catalog-service/internal/handler/audit_handler.go:30`'dan
    al.
  - `pages/admin/system/audit` sayfasını gerçek veriye uyarla.
  - **Commit:** `feat(frontend): show the real audit log`

- [x] **5.6 mock_data'yı sil** (gemini ile yapıldı)
  - `frontend/src/mock_data/` dizini kullanıcı talimatı doğrultusunda seed/referans
    amacıyla `docs/mock_data/` altına taşındı; `frontend/src` altından tamamen çıkarıldı.
  - Kontroller:
    - `grep -rni mock frontend/src --include=*.ts --include=*.tsx | grep -v "\.test\."`
      boş döndü (test yardımcıları hariç hiçbir mock kalmadı).
    - `bun run build` sonrası `frontend/dist` içinde `mockFaculties` ve
      mock'a özgü bir metin (örn. `fevzi.cakmak@mydreamcampus.edu.tr`)
      bulunmadığı doğrulandı.
    - Backend'de "MOCK" log ve yorumları kalmadı:
      `grep -rn "MOCK" new-backend --include=*.go` temizlendi (`fcm.go`).
  - **Commit:** `chore(frontend): delete the mock data module`

## Faz sonu
- README §5 yeşil olmalı.
- README §1 madde 6.
