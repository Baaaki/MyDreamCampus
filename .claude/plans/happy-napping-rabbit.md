# Frontend: Sistem Yönetimi (Time Machine & Academic Periods)

## Context
Backend'de 4 servise Time Machine ve Academic Periods sistemi eklendi. Admin'in sidebar'ında "Zamanı Değiştir" nav item'ı olacak ve `/system` sayfasına yönlendirecek. Sadece admin layout'unda görünecek.

---

## Traefik Config Fix (Zorunlu)

**Problem:** `catalog-service-public` Traefik router'ı `PathPrefix(/api/catalog) && Method(GET)` ile priority 100'de, forward-auth bypass ediyor. Bu yüzden `GET /api/catalog/admin/time/status` ve `GET /api/catalog/admin/periods` istekleri auth header'sız gelir ve backend `RequireAdmin()` reddeder.

**Dosya:** `backend/infrastructure/traefik/dynamic.yml`
- Yeni router eklenir: `catalog-admin` — `PathPrefix(/api/catalog/admin)` → priority 120, forward-auth ile

---

## Frontend Dosya Planı

### 1. `frontend/lib/types.ts` — Yeni tipler (dosya sonuna)
- `TimeStatus`, `AcademicPeriod`, `CreatePeriodRequest`, `UpdatePeriodRequest`

### 2. `frontend/lib/services/system-service.ts` — API wrapper'ları
- Servis path mapping sabitleri (4 servis x time/period path'leri)
- Time Machine: `getAllTimeStatuses()`, `simulateTimeAll(time)`, `resetTimeAll()` — hepsi `Promise.allSettled` ile 4 servise paralel
- Periods: `listPeriods(service)`, `createPeriod(service, data)`, `updatePeriod(service, id, data)`, `deletePeriod(service, id)`
- Mevcut `gradesApi`, `enrollmentApi`, `mealApi`, `catalogApi` kullanılır

### 3. `frontend/app/(admin)/system/page.tsx` — Ana sayfa

**Bölüm 1 — Zaman Makinesi (Card):**
- 4 servisten paralel status çekme
- Her servis satırda: isim + Badge (yeşil=real, turuncu=simulated) + zaman
- `<input type="datetime-local">` + "Simüle Et" + "Sıfırla" butonları
- Paralel istekler, sonuçlar Toast ile

**Bölüm 2 — Akademik Dönemler (Card):**
- Tabs: `Notlar | Kayıt | Yemekhane | Ders Kataloğu`
- Table: dönemleri listeler
- "Yeni Dönem Ekle" → Dialog formu
- Satır aksiyonları: Düzenle (Dialog), Sil (AlertDialog onay)

### 4. `frontend/components/layout/sidebar.tsx` — Nav item
- `Timer` ikonu ile "Zamanı Değiştir", `href: '/system'`
- "Ayarlar" öncesine eklenir

---

## Kullanılacak mevcut dosyalar
- `components/enrollment/Toast.tsx`, `components/ui/*.tsx` (Card, Button, Badge, Table, Dialog, Tabs, Input, Label, Select)
- `lib/api-client.ts` (gradesApi, enrollmentApi, mealApi, catalogApi)
- `date-fns` + `date-fns/locale/tr`

## Verification
1. `bun run build` başarılı
2. Admin sidebar'da "Zamanı Değiştir" görünür, tıklayınca `/system` açılır
3. Simüle Et/Sıfırla tüm servislerde çalışır
4. Akademik dönem CRUD her servis tab'ında çalışır
