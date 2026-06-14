# Wizard Ders Görünmeme Sorunu - ÇÖZÜLDÜ

## Sorun
Wizard'ın 3. adımı (Ders Ekleme) ve 4. adımı (Önizleme) açılan dersleri göstermiyordu.

## Gerçek Kök Neden
```
GET /api/semesters/2026-2027-Fall/courses?limit=200 → 400 Bad Request
```

Backend'deki `PaginationRequest` struct'ında Gin validation tag'i:
```go
// backend/services/course-catalog-service/internal/dto/common_dto.go
type PaginationRequest struct {
    Page  int `form:"page" binding:"omitempty,min=1"`
    Limit int `form:"limit" binding:"omitempty,min=1,max=100"`  // <-- max=100
}
```

Wizard `limit=200` gönderiyordu → Gin binding-level validation fail → 400 Bad Request → catch bloğu boş array bırakıyor → dersler görünmüyor.

İronik olan: handler'da binding'den sonra `if req.Limit > 100 { req.Limit = 100 }` güvenlik kontrolü var ama oraya hiç ulaşılamıyordu çünkü struct-level validation daha önce devreye girdi.

## Gerçek Fix (tek satır)
```diff
- .get(`${fetchSemesterName}/courses`, { searchParams: { limit: 200 } })
+ .get(`${fetchSemesterName}/courses`, { searchParams: { limit: 100 } })
```
Dosya: `frontend/src/pages/admin/system/semesters/new/index.tsx`

## Yanlış Tanı (Önceki Oturumlarda)
Sorun "React state yönetimi" olarak teşhis edildi. Aşağıdaki değişiklikler yapıldı ama hiçbiri sorunu çözmedi çünkü sorun state'te değil, API validation'daydı:

### Review Sonucu — Tüm değişiklikler kalıyor
Hiçbiri zararlı değil. Sorunun kök nedeni olmasa da hepsi ya ayrı bug fix ya da UX iyileştirmesi:

1. **`fetchSemesterName` fallback** — Savunmacı ama zararsız, `undefined` URL edge case'ini önlüyor. KALIYOR.
2. **`window focus` event listener** — Başka sekmede ders ekleyip dönünce auto-refresh. KALIYOR (UX).
3. **409 Conflict handling** — Dönem zaten varsa wizard devam edebiliyor. KALIYOR (ayrı bug fix).
4. **Step >= 2 useEffect auto-fetch** — Step değiştiğinde otomatik yükleme. KALIYOR.
5. **semester-courses hardcoded `2024-2025-Fall` fix** — Ayrı bir gerçek bug fix. KALIYOR.
6. **Wizard "Ders Ekle" → `?semester=...&from=wizard`** — Doğru dönemi aktarıyor. KALIYOR.
7. **"Wizard'a Dön" butonu + dönem dropdown** — UX iyileştirmesi. KALIYOR.
