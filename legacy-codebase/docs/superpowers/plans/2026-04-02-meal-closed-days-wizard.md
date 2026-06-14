# Meal Closed Days Wizard Integration - Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add cafeteria closed days to the semester creation wizard (step 2) so admin can define holiday dates during semester setup, with catalog-service forwarding them to meal-service.

**Architecture:** Frontend wizard step 2 gets a new "Yemekhane Kapalı Günler" card where admin adds date+reason pairs. On semester creation, catalog-service receives closed_days in the POST body and forwards them via internal HTTP to meal-service's new batch endpoint. Meal-service stores them in its existing closed_days table.

**Tech Stack:** Go (Gin), React (TypeScript), date-fns, shadcn/ui components

---

## File Map

### Backend — Meal Service
| Action | File | Purpose |
|--------|------|---------|
| Modify | `backend/services/meal-service/internal/handler/closed_days_handler.go` | Add `BatchCreateClosedDays` handler + `RegisterInternalRoutes` |
| Modify | `backend/services/meal-service/cmd/main.go` | Mount internal route group for closed-days batch |

### Backend — Catalog Service
| Action | File | Purpose |
|--------|------|---------|
| Modify | `backend/services/course-catalog-service/config/config.go` | Add `MealService` config |
| Modify | `backend/services/course-catalog-service/internal/handler/semester_status_handler.go` | Add `closedDayEntry` type, `ClosedDays` field in request, `distributeClosedDays` method, add `Meal` to `ServiceURLs` |
| Modify | `backend/services/course-catalog-service/cmd/main.go` | Pass `Meal` URL to `ServiceURLs` |

### Frontend
| Action | File | Purpose |
|--------|------|---------|
| Modify | `frontend/src/lib/types.ts` | Add `closed_days` to `CreateSemesterRequest` |
| Modify | `frontend/src/pages/admin/system/semesters/new/index.tsx` | Add closed days state, UI card in step 2, preview in step 4, include in API payload |

---

## Task 1: Meal Service — Batch Closed Days Endpoint

**Files:**
- Modify: `backend/services/meal-service/internal/handler/closed_days_handler.go`

- [ ] **Step 1: Add batch request/response types and handler**

Add these types and the `BatchCreateClosedDays` handler after the existing `DeleteClosedDay` function (after line 195, before `toClosedDayResponse`):

```go
type batchCreateClosedDaysRequest struct {
	ClosedDays []createClosedDayRequest `json:"closed_days" binding:"required,min=1"`
}

type batchCreateClosedDaysResponse struct {
	Created []closedDayResponse `json:"created"`
	Skipped []string            `json:"skipped"`
}

// BatchCreateClosedDays adds multiple closed days at once, skipping duplicates.
// POST /internal/closed-days/batch
func (h *ClosedDaysHandler) BatchCreateClosedDays(c *gin.Context) {
	var req batchCreateClosedDaysRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
			"code":  "VALIDATION_ERROR",
		})
		return
	}

	var created []closedDayResponse
	var skipped []string

	for _, entry := range req.ClosedDays {
		date, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			skipped = append(skipped, entry.Date+" (invalid format)")
			continue
		}

		closedDay, err := h.repo.CreateClosedDay(c.Request.Context(), db.CreateClosedDayParams{
			Date:   pgtype.Date{Time: date, Valid: true},
			Reason: entry.Reason,
		})
		if err != nil {
			// Duplicate date — skip silently
			skipped = append(skipped, entry.Date+" (already exists)")
			continue
		}

		created = append(created, toClosedDayResponse(closedDay))
	}

	h.logger.Info("batch closed days processed",
		zap.Int("created", len(created)),
		zap.Int("skipped", len(skipped)),
	)

	c.JSON(http.StatusCreated, batchCreateClosedDaysResponse{
		Created: created,
		Skipped: skipped,
	})
}
```

- [ ] **Step 2: Add `RegisterInternalRoutes` method**

Add this method after the existing `RegisterRoutes` method (after line 33):

```go
// RegisterInternalRoutes mounts internal closed days endpoints for service-to-service calls.
func (h *ClosedDaysHandler) RegisterInternalRoutes(rg *gin.RouterGroup) {
	closedDays := rg.Group("/closed-days")
	{
		closedDays.POST("/batch", h.BatchCreateClosedDays)
	}
}
```

- [ ] **Step 3: Verify build**

Run: `cd /home/nautilus/Desktop/Playground/mydreamcampus/backend && go build ./services/meal-service/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add backend/services/meal-service/internal/handler/closed_days_handler.go
git commit -m "feat(meal): add batch closed days endpoint for internal service calls"
```

---

## Task 2: Meal Service — Mount Internal Routes

**Files:**
- Modify: `backend/services/meal-service/cmd/main.go`

- [ ] **Step 1: Add `sharedMiddleware` import if not present**

Check if `sharedMiddleware` is already imported. If not, add to imports:

```go
sharedMiddleware "github.com/baaaki/mydreamcampus/shared/middleware"
```

- [ ] **Step 2: Add internal route group in `setupRouter`**

After the closing `}` of the `api` group (after line 247, before `return router`), add the internal route group following the same pattern used by grades-service, attendance-service, and enrollment-service:

```go
	// Internal routes (service-to-service, no auth)
	internal := router.Group("/api/meals/internal")
	internal.Use(sharedMiddleware.StripInternalHeaders())
	{
		closedDaysHandler.RegisterInternalRoutes(internal)
	}
```

- [ ] **Step 3: Verify build**

Run: `cd /home/nautilus/Desktop/Playground/mydreamcampus/backend && go build ./services/meal-service/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add backend/services/meal-service/cmd/main.go
git commit -m "feat(meal): mount internal routes for closed-days batch endpoint"
```

---

## Task 3: Catalog Service — Add Meal Service Config

**Files:**
- Modify: `backend/services/course-catalog-service/config/config.go`

- [ ] **Step 1: Add `MealService` field to Config struct**

In the `Config` struct (line 11), add `MealService` after `AttendanceService`:

```go
type Config struct {
	Server            config.ServerConfig
	Database          config.DatabaseConfig
	RabbitMQ          config.RabbitMQConfig
	Redis             config.RedisConfig
	JWT               config.JWTConfig
	StaffService      StaffServiceConfig
	RateLimit         config.RateLimitConfig
	EnrollmentService ServiceURLConfig
	GradesService     ServiceURLConfig
	AttendanceService ServiceURLConfig
	MealService       ServiceURLConfig
}
```

- [ ] **Step 2: Add default and load for `MEAL_SERVICE_URL`**

In the `Load()` function, add the default after the `ATTENDANCE_SERVICE_URL` default (after line 43):

```go
viper.SetDefault("MEAL_SERVICE_URL", "http://localhost:"+config.MealServicePort)
```

And in the config struct construction (after line 69, the `AttendanceService` line):

```go
MealService:       ServiceURLConfig{BaseURL: viper.GetString("MEAL_SERVICE_URL")},
```

- [ ] **Step 3: Verify build**

Run: `cd /home/nautilus/Desktop/Playground/mydreamcampus/backend && go build ./services/course-catalog-service/...`
Expected: No errors

- [ ] **Step 4: Commit**

```bash
git add backend/services/course-catalog-service/config/config.go
git commit -m "feat(catalog): add MEAL_SERVICE_URL config"
```

---

## Task 4: Catalog Service — Forward Closed Days to Meal Service

**Files:**
- Modify: `backend/services/course-catalog-service/internal/handler/semester_status_handler.go`
- Modify: `backend/services/course-catalog-service/cmd/main.go`

- [ ] **Step 1: Add `Meal` to `ServiceURLs` struct**

In `semester_status_handler.go`, modify the `ServiceURLs` struct (line 26-30):

```go
type ServiceURLs struct {
	Enrollment string
	Grades     string
	Attendance string
	Meal       string
}
```

- [ ] **Step 2: Add `closedDayEntry` type and update `createSemesterRequest`**

After the `semesterPeriods` struct (after line 71), add:

```go
type closedDayEntry struct {
	Date   string `json:"date" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}
```

Update `createSemesterRequest` (line 73-77):

```go
type createSemesterRequest struct {
	Name         string           `json:"name" binding:"required"`
	HardDeadline time.Time        `json:"hard_deadline" binding:"required"`
	Periods      *semesterPeriods `json:"periods,omitempty"`
	ClosedDays   []closedDayEntry `json:"closed_days,omitempty"`
}
```

- [ ] **Step 3: Add `distributeClosedDays` method**

Add this method after `createRemotePeriod` (after line 260):

```go
// distributeClosedDays sends closed days to meal-service via internal HTTP.
func (h *SemesterStatusHandler) distributeClosedDays(ctx context.Context, closedDays []closedDayEntry) error {
	if h.serviceURLs.Meal == "" {
		return fmt.Errorf("meal service URL not configured")
	}

	payload := map[string]any{
		"closed_days": closedDays,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal closed days payload: %w", err)
	}

	url := h.serviceURLs.Meal + "/api/meals/internal/closed-days/batch"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP call failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
```

- [ ] **Step 4: Call `distributeClosedDays` in `CreateSemester` handler**

In the `CreateSemester` function, after the period distribution block (after line 166, before the final `c.JSON(http.StatusCreated, resp)`), add:

```go
	// Distribute closed days to meal service
	if len(req.ClosedDays) > 0 {
		if err := h.distributeClosedDays(c.Request.Context(), req.ClosedDays); err != nil {
			handlerLogger.Warn("failed to distribute closed days to meal service", zap.Error(err))
			if len(periodErrors) == 0 {
				periodErrors = []string{}
			}
			periodErrors = append(periodErrors, fmt.Sprintf("meal (closed_days): %v", err))
		}
	}
```

Note: The `periodErrors` variable needs to be available in this scope. Currently the period distribution block returns errors and handles them inline. We need to refactor slightly — extract `periodErrors` before the period block so both blocks can append to it.

Replace the existing period distribution block (lines 156-166) and add closed days handling:

```go
	var periodErrors []string

	if req.Periods != nil {
		periodErrors = h.distributePeriods(c.Request.Context(), req.Name, req.HardDeadline, req.Periods)
	}

	// Distribute closed days to meal service
	if len(req.ClosedDays) > 0 {
		if err := h.distributeClosedDays(c.Request.Context(), req.ClosedDays); err != nil {
			handlerLogger.Warn("failed to distribute closed days to meal service", zap.Error(err))
			periodErrors = append(periodErrors, fmt.Sprintf("meal (closed_days): %v", err))
		}
	}

	if len(periodErrors) > 0 {
		handlerLogger.Warn("some distributions failed", zap.Any("errors", periodErrors))
		c.JSON(http.StatusCreated, gin.H{
			"semester":      resp,
			"period_errors": periodErrors,
		})
		return
	}
```

- [ ] **Step 5: Update `main.go` to pass Meal URL**

In `backend/services/course-catalog-service/cmd/main.go`, modify the `ServiceURLs` initialization (line 119-123):

```go
	semesterStatusHandler := handler.NewSemesterStatusHandler(semesterStatusRepo, periodRepo, catalogAuditLogger, handler.ServiceURLs{
		Enrollment: cfg.EnrollmentService.BaseURL,
		Grades:     cfg.GradesService.BaseURL,
		Attendance: cfg.AttendanceService.BaseURL,
		Meal:       cfg.MealService.BaseURL,
	})
```

- [ ] **Step 6: Verify build**

Run: `cd /home/nautilus/Desktop/Playground/mydreamcampus/backend && go build ./services/course-catalog-service/...`
Expected: No errors

- [ ] **Step 7: Commit**

```bash
git add backend/services/course-catalog-service/internal/handler/semester_status_handler.go backend/services/course-catalog-service/cmd/main.go backend/services/course-catalog-service/config/config.go
git commit -m "feat(catalog): forward closed days to meal service during semester creation"
```

---

## Task 5: Frontend — Update Types

**Files:**
- Modify: `frontend/src/lib/types.ts`

- [ ] **Step 1: Add `closed_days` to `CreateSemesterRequest`**

Update the `CreateSemesterRequest` interface (line 812-816):

```typescript
export interface CreateSemesterRequest {
  name: string;
  hard_deadline: string;
  periods?: SemesterPeriods;
  closed_days?: Array<{ date: string; reason: string }>;
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/lib/types.ts
git commit -m "feat(frontend): add closed_days to CreateSemesterRequest type"
```

---

## Task 6: Frontend — Wizard Step 2 Closed Days Card

**Files:**
- Modify: `frontend/src/pages/admin/system/semesters/new/index.tsx`

- [ ] **Step 1: Add `UtensilsCrossed` and `Trash2` to lucide imports**

Update the lucide-react import (line 5-16) to include `UtensilsCrossed`, `Trash2`, and `Plus`:

```typescript
import {
  ArrowLeft,
  ArrowRight,
  CalendarRange,
  CheckCircle2,
  Clock,
  GraduationCap,
  Loader2,
  Play,
  AlertCircle,
  BookOpen,
  UtensilsCrossed,
  Trash2,
  Plus,
} from 'lucide-react';
```

- [ ] **Step 2: Add `closedDays` state**

In the `SemesterWizardPage` component, after the `periods` state (after line 90), add:

```typescript
  // Step 2: Closed days (meal service)
  const [closedDays, setClosedDays] = useState<Array<{ date: string; reason: string }>>([]);
```

- [ ] **Step 3: Update `handleCreateSemester` to include closed days**

In `handleCreateSemester`, update the `createSemester` call (line 152-155) to include `closed_days`:

```typescript
      const semester = await createSemester({
        name: semesterName,
        hard_deadline: new Date(hardDeadline).toISOString(),
        periods: periodsPayload,
        closed_days: closedDays.length > 0 ? closedDays : undefined,
      });
```

- [ ] **Step 4: Pass closedDays props to StepPeriods and StepPreview**

Update the `StepPeriods` render (line 300-305):

```tsx
      {step === 1 && (
        <StepPeriods
          periods={periods}
          setPeriods={setPeriods}
          hardDeadline={hardDeadline}
          closedDays={closedDays}
          setClosedDays={setClosedDays}
        />
      )}
```

Update the `StepPreview` render (line 313-318):

```tsx
      {step === 3 && (
        <StepPreview
          semesterName={semesterName}
          hardDeadline={hardDeadline}
          periods={periods}
          resolvedSemesterName={resolvedSemesterName}
          closedDays={closedDays}
        />
      )}
```

- [ ] **Step 5: Update `StepPeriods` component to accept and render closed days**

Update the `StepPeriods` function signature and props (lines 486-494):

```tsx
function StepPeriods({
  periods,
  setPeriods,
  hardDeadline,
  closedDays,
  setClosedDays,
}: {
  periods: Record<string, { start: string; end: string }>;
  setPeriods: (v: any) => void;
  hardDeadline: string;
  closedDays: Array<{ date: string; reason: string }>;
  setClosedDays: React.Dispatch<React.SetStateAction<Array<{ date: string; reason: string }>>>;
}) {
```

After the existing service period cards loop (after line 568, before the amber info box), add the closed days card:

```tsx
        {/* Meal Service — Closed Days */}
        <ClosedDaysCard
          closedDays={closedDays}
          setClosedDays={setClosedDays}
          hardDeadline={hardDeadline}
        />
```

- [ ] **Step 6: Create `ClosedDaysCard` component**

Add this component after the `StepPeriods` component (before `StepCourses`):

```tsx
// ============================================================================
// Closed Days Card (used in Step 2)
// ============================================================================

function ClosedDaysCard({
  closedDays,
  setClosedDays,
  hardDeadline,
}: {
  closedDays: Array<{ date: string; reason: string }>;
  setClosedDays: React.Dispatch<React.SetStateAction<Array<{ date: string; reason: string }>>>;
  hardDeadline: string;
}) {
  const [newDate, setNewDate] = useState('');
  const [newReason, setNewReason] = useState('');

  const deadline = hardDeadline ? new Date(hardDeadline) : null;

  const addClosedDay = () => {
    if (!newDate || !newReason.trim()) return;
    if (closedDays.some((d) => d.date === newDate)) return;

    setClosedDays((prev) => [...prev, { date: newDate, reason: newReason.trim() }]);
    setNewDate('');
    setNewReason('');
  };

  const removeClosedDay = (date: string) => {
    setClosedDays((prev) => prev.filter((d) => d.date !== date));
  };

  const isDuplicate = closedDays.some((d) => d.date === newDate);
  const exceedsDeadline = deadline && newDate && new Date(newDate + 'T23:59:59') > deadline;

  return (
    <div className="rounded-lg border border-orange-200 dark:border-orange-800 bg-orange-50/30 dark:bg-orange-950/10 p-4 space-y-3">
      <div className="flex items-center justify-between">
        <div>
          <h4 className="font-medium text-sm flex items-center gap-1.5">
            <UtensilsCrossed className="h-4 w-4 text-orange-600" />
            Yemekhane Kapalı Günler
          </h4>
          <p className="text-xs text-gray-500 dark:text-gray-400">
            Yemekhanenin kapalı olacağı özel günler (resmi tatiller vb.)
          </p>
        </div>
        {closedDays.length > 0 && (
          <Badge variant="outline" className="border-orange-500 text-orange-600">
            {closedDays.length} gün
          </Badge>
        )}
      </div>

      {/* Add new closed day */}
      <div className="flex gap-2 items-end">
        <div className="space-y-1 flex-shrink-0">
          <Label className="text-xs">Tarih</Label>
          <Input
            type="date"
            value={newDate}
            onChange={(e) => setNewDate(e.target.value)}
            className="w-[160px]"
          />
        </div>
        <div className="space-y-1 flex-1">
          <Label className="text-xs">Sebep</Label>
          <Input
            type="text"
            placeholder="ör. Cumhuriyet Bayramı"
            value={newReason}
            onChange={(e) => setNewReason(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                e.preventDefault();
                addClosedDay();
              }
            }}
          />
        </div>
        <Button
          type="button"
          size="sm"
          variant="outline"
          onClick={addClosedDay}
          disabled={!newDate || !newReason.trim() || isDuplicate || !!exceedsDeadline}
          className="flex-shrink-0"
        >
          <Plus className="h-4 w-4 mr-1" />
          Ekle
        </Button>
      </div>

      {isDuplicate && (
        <p className="text-xs text-red-500 flex items-center gap-1">
          <AlertCircle className="h-3 w-3" />
          Bu tarih zaten eklenmiş
        </p>
      )}
      {exceedsDeadline && (
        <p className="text-xs text-red-500 flex items-center gap-1">
          <AlertCircle className="h-3 w-3" />
          Tarih hard deadline'ı aşamaz
        </p>
      )}

      {/* List of added closed days */}
      {closedDays.length > 0 && (
        <div className="rounded-md border border-gray-200 dark:border-gray-700 divide-y divide-gray-200 dark:divide-gray-700">
          {closedDays
            .sort((a, b) => a.date.localeCompare(b.date))
            .map((day) => (
              <div key={day.date} className="flex items-center justify-between px-3 py-2 text-sm">
                <div className="flex items-center gap-3">
                  <span className="font-mono text-xs text-gray-500">{day.date}</span>
                  <span className="text-xs text-gray-400">
                    {format(new Date(day.date + 'T00:00:00'), 'EEEE', { locale: tr })}
                  </span>
                  <span>{day.reason}</span>
                </div>
                <button
                  type="button"
                  onClick={() => removeClosedDay(day.date)}
                  className="text-red-400 hover:text-red-600 p-1"
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            ))}
        </div>
      )}
    </div>
  );
}
```

- [ ] **Step 7: Verify frontend compiles**

Run: `cd /home/nautilus/Desktop/Playground/mydreamcampus/frontend && bun tsc --noEmit`
Expected: No type errors

- [ ] **Step 8: Commit**

```bash
git add frontend/src/pages/admin/system/semesters/new/index.tsx
git commit -m "feat(frontend): add closed days card to semester wizard step 2"
```

---

## Task 7: Frontend — Step 4 Preview

**Files:**
- Modify: `frontend/src/pages/admin/system/semesters/new/index.tsx`

- [ ] **Step 1: Update `StepPreview` to accept `closedDays` prop**

Update the `StepPreview` function signature (lines 622-631):

```tsx
function StepPreview({
  semesterName,
  hardDeadline,
  periods,
  resolvedSemesterName,
  closedDays,
}: {
  semesterName: string;
  hardDeadline: string;
  periods: Record<string, { start: string; end: string }>;
  resolvedSemesterName: string;
  closedDays: Array<{ date: string; reason: string }>;
}) {
```

- [ ] **Step 2: Add closed days table in preview**

After the Periods Summary Card (after line 698, before the Courses Summary Card), add:

```tsx
      {/* Closed Days Summary */}
      {closedDays.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <UtensilsCrossed className="h-5 w-5" />
              Yemekhane Kapalı Günler
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="rounded-lg border border-gray-200 dark:border-gray-700">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Tarih</TableHead>
                    <TableHead>Gün</TableHead>
                    <TableHead>Sebep</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {closedDays
                    .sort((a, b) => a.date.localeCompare(b.date))
                    .map((day) => (
                      <TableRow key={day.date}>
                        <TableCell className="font-mono text-sm">{day.date}</TableCell>
                        <TableCell>
                          {format(new Date(day.date + 'T00:00:00'), 'EEEE', { locale: tr })}
                        </TableCell>
                        <TableCell>{day.reason}</TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}
```

- [ ] **Step 3: Verify frontend compiles**

Run: `cd /home/nautilus/Desktop/Playground/mydreamcampus/frontend && bun tsc --noEmit`
Expected: No type errors

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/admin/system/semesters/new/index.tsx
git commit -m "feat(frontend): show closed days in semester wizard preview step"
```

---

## Task 8: Manual Verification

- [ ] **Step 1: Start services and test the flow**

1. Start meal-service and catalog-service (user runs Docker/services manually)
2. Open `http://localhost:3000/system/semesters` and click "Yeni Dönem"
3. Fill Step 1 (semester name + hard deadline)
4. In Step 2, verify the 4 existing service cards appear as before
5. Verify the "Yemekhane Kapalı Günler" card appears at the bottom
6. Add a few dates with reasons, verify day names appear correctly
7. Try adding a duplicate date — verify validation message
8. Try adding a date past hard deadline — verify validation message
9. Proceed to Step 4 (Preview) — verify closed days table appears
10. Submit — verify no errors

- [ ] **Step 2: Final commit with all changes if any fixups needed**

Only if any fixups were required during manual testing.
