# Auto-Finalize Bug — Devam Notu

**Tarih:** 2026-04-19
**Durum:** Sebep tespit edildi, fix uygulanmadı
**Etkilenen servis:** `backend/services/grades-service`

---

## Problem Özeti

Öğrenci #123456 **Halis Muhsin**'in **BİL 1013** dersine ait tüm notları (midterm, final, homework) girilip `is_locked=true` olarak kaydedilmesine rağmen ders **finalize olmadı** — `student_completed_courses` tablosunda hiçbir kayıt yok, frontend'de "Aktif Dersler" listesinde "henüz not girilmedi" görünüyor.

Frontend'de `/student/grades` sayfasında ders aktif tarafta kalıyor; completed sekmesine geçmiyor.

---

## Kök Sebep (Smoking Gun)

`backend/services/grades-service/sql/queries/assessment_scores.sql` içindeki `UpsertAssessmentScore`:

```sql
INSERT INTO student_assessment_scores (
    registration_id, slug, score, is_absent, graded_by, graded_at, is_locked
) VALUES (
    $1, $2, $3, $4, $5, NOW(), TRUE  -- INSERT'te is_locked=TRUE
)
ON CONFLICT (registration_id, slug) DO UPDATE SET
    score = EXCLUDED.score,
    is_absent = EXCLUDED.is_absent,
    graded_by = EXCLUDED.graded_by,
    graded_at = NOW()
    -- is_locked GÜNCELLENMİYOR — UPDATE case'inde olduğu gibi kalıyor
WHERE student_assessment_scores.is_locked = FALSE
```

### İki Bug Var

**Bug 1 — `UpsertAssessmentScore` UPDATE case'i `is_locked` setlenmiyor:**
- INSERT'te `is_locked=TRUE` ✓
- UPDATE'te `is_locked` değişmiyor → eğer skor önceden `unlock`'lanmışsa, `SubmitScore` sonrası hâlâ `unlocked` kalır
- Sonuç: `checkAllScoresLocked` `false` döner → `AutoFinalize` tetiklenmez

**Bug 2 — `LockScore` admin handler'ı `checkAllScoresLocked` çağırmıyor:**
- [grade_service.go:972](../backend/services/grades-service/internal/service/grade_service.go#L972) `LockScore` fonksiyonu sadece DB'de lock set ediyor
- `SubmitScore` (line 171) ve `BulkSubmitScores` (line 217) `checkAllScoresLocked` → `AutoFinalize` çağırıyor; `LockScore` ÇAĞIRMIYOR
- Sonuç: Admin manuel olarak skoru kilitlediğinde finalize tetiklenmez

Halis'in vakasında muhtemelen şu olmuş: skor önce girilip kilitlendi (INSERT, locked=true), sonra düzeltme için yeniden girildi (UPDATE, lock değişmedi), ama daha önce bir unlock olduysa hâlâ unlocked kaldı. Veya admin sonradan LockScore ile kilitledi ama finalize hiç tetiklenmedi.

### Kanıt Logu

Test sırasında midterm'i unlock + resubmit ettik:

```
06:27:16.395  not all scores locked   {"slug": "midterm", "locked_count": 0, "total_students": 1}
```

`SubmitScore` çağrıldı, UPDATE case'i çalıştı, `is_locked` `false` kaldı → `checkAllScoresLocked` `false` döndü → finalize atlandı.

---

## Daha Önce Çalıştırılan Komutlar (Tekrarlama!)

### 1. Halis'in registration ve course ID'leri

```bash
sudo docker exec -it mydreamcampus-postgres-grades psql -U postgres -d mydreamcampus_grades -c "
SELECT r.id AS reg_id, r.course_id
FROM student_course_registrations r
JOIN courses_cache c ON r.course_id = c.id
JOIN students_cache s ON r.student_id = s.id
WHERE s.student_number = '123456' AND c.course_code = 'BİL 1013';
"
```

**Çıktı:**
- `reg_id` = `94843aab-d577-42df-956e-26f21c315e76`
- `course_id` = `019d9a3d-78c8-7b11-977d-fad06d25e83a`

### 2. Course'un instructor bilgisi

```bash
sudo docker exec -it mydreamcampus-postgres-grades psql -U postgres -d mydreamcampus_grades -c "
SELECT id, course_code, instructor_id, instructor_fullname
FROM courses_cache WHERE id = '019d9a3d-78c8-7b11-977d-fad06d25e83a';
"
```

**Çıktı:**
- `instructor_id` = `019b4a11-713c-76d0-839f-30275a3b387b`
- `instructor_fullname` = `Dr. Ahmet Yılmaz`

### 3. Schema ve skorlar

```bash
sudo docker exec -it mydreamcampus-postgres-grades psql -U postgres -d mydreamcampus_grades -c "
SELECT
  c.course_code,
  jsonb_pretty(c.assessment_schema) AS schema,
  (SELECT json_agg(json_build_object(
      'slug', sa.slug, 'score', sa.score,
      'is_absent', sa.is_absent, 'is_locked', sa.is_locked,
      'graded_at', sa.graded_at))
   FROM student_assessment_scores sa
   JOIN student_course_registrations r ON sa.registration_id = r.id
   WHERE r.course_id = c.id) AS scores
FROM courses_cache c
WHERE c.course_code = 'BİL 1013';
"
```

**Önemli sonuç:** `courses_cache`'te BİL 1013 için 4 row var (farklı dönem/şube), Halis'in kaydı **3 assessment'lı şemaya sahip olan** (midterm 40%, final 50%, homework 10%) row'a ait. 3 skoru da `is_locked=true` (test öncesi durumda).

### 4. Halis için completed_course (sanity check)

```bash
sudo docker exec -it mydreamcampus-postgres-grades psql -U postgres -d mydreamcampus_grades -c "
SELECT student_number, course_code, grade_point, finalized_at
FROM student_completed_courses
WHERE student_number = '123456';"
```

**Çıktı:** `(0 rows)` — onaylandı, hiç finalize edilmemiş.

### 5. Test: midterm unlock (admin)

```bash
curl -s -X POST http://localhost:8007/api/grades/admin/scores/unlock \
  -H "Authorization: Bearer dummy" \
  -H "X-Internal-Secret: changeme_internal_secret" \
  -H "X-User-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-User-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"registration_id":"94843aab-d577-42df-956e-26f21c315e76","slug":"midterm"}'
```

**Çıktı:** `{"message":"score unlocked successfully"}` — başarılı.

### 6. Test: midterm resubmit (instructor)

```bash
curl -s -X POST http://localhost:8007/api/grades/course/019d9a3d-78c8-7b11-977d-fad06d25e83a/scores \
  -H "Authorization: Bearer dummy" \
  -H "X-Internal-Secret: changeme_internal_secret" \
  -H "X-User-ID: 019b4a11-713c-76d0-839f-30275a3b387b" \
  -H "X-User-Role: teacher" \
  -H "Content-Type: application/json" \
  -d '{"registration_id":"94843aab-d577-42df-956e-26f21c315e76","slug":"midterm","score":59,"is_absent":false}'
```

**Çıktı:** `{"id":"...","slug":"midterm","score":59,...}` — skor güncellendi ama log: `not all scores locked, locked_count=0, total_students=1` → finalize tetiklenmedi (Bug 1 kanıtlandı).

> ⚠️ Not: Admin olarak `SubmitScore` çağrılamıyor (`NOT_COURSE_INSTRUCTOR` hatası). Sadece dersin gerçek instructor'ı (Dr. Ahmet Yılmaz) skor girebiliyor.

> ⚠️ **Yan etki — DİKKAT:** Test sonunda Halis'in midterm'i hâlâ `unlock` durumda. Frontend'de sayfa açılırsa "midterm girilmemiş" gibi görünebilir. **İlk fix'ten önce ya da sonra mutlaka tekrar lock'lanmalı.**

---

## Kontekst — İlgili Dosyalar

| Dosya | Satır | İlgi |
|---|---|---|
| [backend/services/grades-service/sql/queries/assessment_scores.sql](../backend/services/grades-service/sql/queries/assessment_scores.sql) | 1-13 | `UpsertAssessmentScore` — Bug 1 buradan kaynaklanıyor |
| [backend/services/grades-service/internal/service/grade_service.go](../backend/services/grades-service/internal/service/grade_service.go) | 705 | `checkAllScoresLocked` — kilit sayımını yapıyor |
| [backend/services/grades-service/internal/service/grade_service.go](../backend/services/grades-service/internal/service/grade_service.go) | 171, 217 | `SubmitScore` ve `BulkSubmitScores` finalize check yapıyor |
| [backend/services/grades-service/internal/service/grade_service.go](../backend/services/grades-service/internal/service/grade_service.go) | 972 | `LockScore` — Bug 2: finalize check yok |
| [backend/services/grades-service/internal/service/grade_service.go](../backend/services/grades-service/internal/service/grade_service.go) | 242 | `AutoFinalize` |
| [backend/tmp/logs/grades-service.log](../backend/tmp/logs/grades-service.log) | — | Live log (air her restart'ta sıfırlıyor) |

---

## Yapılması Gerekenler — Adım Adım

### Adım 1: State'i düzelt (Halis'in midterm'i hâlâ unlock!)

Önce midterm'i tekrar lock'la:

```bash
curl -s -X POST http://localhost:8007/api/grades/admin/scores/lock \
  -H "Authorization: Bearer dummy" \
  -H "X-Internal-Secret: changeme_internal_secret" \
  -H "X-User-ID: 00000000-0000-0000-0000-000000000001" \
  -H "X-User-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"registration_id":"94843aab-d577-42df-956e-26f21c315e76","slug":"midterm"}'
```

Doğrulama:
```bash
sudo docker exec -it mydreamcampus-postgres-grades psql -U postgres -d mydreamcampus_grades -c "
SELECT slug, is_locked FROM student_assessment_scores
WHERE registration_id = '94843aab-d577-42df-956e-26f21c315e76';"
```
Üç slug için de `is_locked = t` görmeliyiz.

### Adım 2: Bug 1 Fix — `UpsertAssessmentScore` UPDATE case'inde `is_locked = TRUE`

`backend/services/grades-service/sql/queries/assessment_scores.sql`:

```diff
 ON CONFLICT (registration_id, slug) DO UPDATE SET
     score = EXCLUDED.score,
     is_absent = EXCLUDED.is_absent,
     graded_by = EXCLUDED.graded_by,
-    graded_at = NOW()
+    graded_at = NOW(),
+    is_locked = TRUE
 WHERE student_assessment_scores.is_locked = FALSE
 RETURNING *;
```

Sonra:
```bash
cd backend/services/grades-service && make sqlc
```

**Mantık:** Submit edilen skor zaten kilitli olmalı (semantik tutarlılık). Re-submit edilen skor da kilitli kalmalı.

### Adım 3: Bug 2 Fix — `LockScore` handler finalize check çağırsın

`backend/services/grades-service/internal/service/grade_service.go` içinde `LockScore` fonksiyonu (line 972):

```go
func (s *GradeService) LockScore(ctx context.Context, registrationID uuid.UUID, slug string) error {
    // ... mevcut kod ...

    if err := s.scoreRepo.LockScore(ctx, registrationID, slug); err != nil {
        logger.Error("failed to lock score", zap.Error(err))
        return err
    }

    logger.Info("score locked by admin", ...)

    // YENİ EKLENECEK: finalize check
    reg, err := s.registrationRepo.GetRegistrationByID(ctx, registrationID)
    if err != nil {
        return nil // lock zaten başarılı, finalize check best-effort
    }
    allLocked, err := s.checkAllScoresLocked(ctx, reg.CourseID)
    if err != nil {
        logger.Error("failed to check locked scores after lock", zap.Error(err))
        return nil
    }
    if allLocked {
        logger.Info("all scores locked after admin lock, triggering auto-finalize",
            zap.String("course_id", reg.CourseID.String()))
        if _, err := s.AutoFinalize(ctx, reg.CourseID, reg.InstructorID); err != nil {
            logger.Error("auto-finalize after admin lock failed", zap.Error(err))
        }
    }
    return nil
}
```

> Not: `GetRegistrationByID` row tipinin `CourseID` ve `InstructorID` alanlarını döndürdüğünü doğrula — döndürmüyorsa repo metodu uyarla.

### Adım 4: Halis'i finalize et (Fix 2 sonrası)

Bug 2 fix'lendikten ve `air` rebuild ettikten sonra, Halis'in herhangi bir skorunu unlock + lock yap:

```bash
# 1) midterm'i unlock
curl -s -X POST http://localhost:8007/api/grades/admin/scores/unlock \
  -H "Authorization: Bearer dummy" -H "X-Internal-Secret: changeme_internal_secret" \
  -H "X-User-ID: 00000000-0000-0000-0000-000000000001" -H "X-User-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"registration_id":"94843aab-d577-42df-956e-26f21c315e76","slug":"midterm"}'
echo

# 2) midterm'i tekrar lock — bu sefer finalize tetiklenecek
curl -s -X POST http://localhost:8007/api/grades/admin/scores/lock \
  -H "Authorization: Bearer dummy" -H "X-Internal-Secret: changeme_internal_secret" \
  -H "X-User-ID: 00000000-0000-0000-0000-000000000001" -H "X-User-Role: admin" \
  -H "Content-Type: application/json" \
  -d '{"registration_id":"94843aab-d577-42df-956e-26f21c315e76","slug":"midterm"}'
echo
```

Logda şunu görmeliyiz:
```
all scores locked after admin lock, triggering auto-finalize
finalize grading {"course_code": "BİL 1013", ...}
```

### Adım 5: Doğrulama

```bash
sudo docker exec -it mydreamcampus-postgres-grades psql -U postgres -d mydreamcampus_grades -c "
SELECT student_number, course_code, weighted_average, grade_point, finalized_at
FROM student_completed_courses
WHERE student_number = '123456';"
```

Bir row görmeliyiz: BİL 1013, hesaplanmış weighted_average, grade_point (örn. FF veya CC), finalized_at dolu.

Sonra frontend'de `/student/grades` aç → BİL 1013 "Tamamlanan Dersler" tablosunda görünmeli.

---

## Tartışma Noktaları (Fix Öncesi Karara Bağlanacak)

1. **Adım 2'deki Bug 1 fix'i doğru semantik mi?** Alternatif: `UpsertAssessmentScore` UPDATE case'inde `is_locked` değiştirmesin (mevcut davranış), bunun yerine bir kullanıcı ayrıca `LockScore` çağırsın. Bu durumda Bug 2 fix'i tek başına yeterli olur. Tercih edilen yaklaşıma karar ver.
2. **Race condition:** `LockScore` + `AutoFinalize` tek transaction'da değil — paralel çalışan iki istek aynı dersi finalize etmeye kalkabilir. `AutoFinalize` idempotent mi (DeleteCompletedCourse + Create var, görünüşe göre evet)? `CreateCompletedCourse`'ta upsert var mı kontrol et.
3. **Test:** Fix sonrası unit/integration test eklenmeli — şu an grades-service'te bu akış için test yok (kontrol edilmesi gereken).

---

## Hızlı Komut Referansı (kopyala-yapıştır için)

```bash
# Postgres'e psql aç
sudo docker exec -it mydreamcampus-postgres-grades psql -U postgres -d mydreamcampus_grades

# Live logları izle (yeni terminal)
tail -f /home/nautilus/Desktop/Playground/mydreamcampus/backend/tmp/logs/grades-service.log

# air'i yeniden başlat (rebuild için)
pkill -f "grades-service/tmp/main"
# air watcher otomatik restart eder
```

**Sabit ID'ler:**
- Halis registration_id: `94843aab-d577-42df-956e-26f21c315e76`
- BİL 1013 course_id: `019d9a3d-78c8-7b11-977d-fad06d25e83a`
- Dr. Ahmet Yılmaz instructor_id: `019b4a11-713c-76d0-839f-30275a3b387b`
- Admin user_id: `00000000-0000-0000-0000-000000000001`
- Internal secret (default): `changeme_internal_secret`
