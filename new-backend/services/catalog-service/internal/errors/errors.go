package errors

import (
	"fmt"
	"net/http"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
)

// ScheduleConflictError wraps the sentinel error and carries conflict details
type ScheduleConflictError struct {
	AppErr     *sharedErrors.AppError
	CourseCode string
	Department string
	DayOfWeek  string
	SlotNumber int16
}

func (e *ScheduleConflictError) Error() string {
	return fmt.Sprintf("[%s] %s", e.AppErr.Code, e.AppErr.Message)
}

func (e *ScheduleConflictError) Unwrap() error {
	return e.AppErr
}

func NewScheduleConflictError(courseCode, department, dayOfWeek string, slotNumber int16) *ScheduleConflictError {
	return &ScheduleConflictError{
		AppErr:     ErrInstructorScheduleConflict,
		CourseCode: courseCode,
		Department: department,
		DayOfWeek:  dayOfWeek,
		SlotNumber: slotNumber,
	}
}

// ============================================================================
// COURSE CATALOG SERVICE SPECIFIC ERRORS
// These errors are specific to course catalog management domain
// ============================================================================

var (
	// Catalog errors (AppError for HTTP responses)
	ErrCourseNotFound           = sharedErrors.New("COURSE_NOT_FOUND", "Ders bulunamadı", http.StatusNotFound)
	ErrCourseCodeExists         = sharedErrors.New("COURSE_CODE_EXISTS", "Bu ders kodu zaten mevcut", http.StatusConflict)
	ErrInvalidPrerequisite      = sharedErrors.New("INVALID_PREREQUISITE", "Geçersiz ön koşul dersi", http.StatusBadRequest)
	ErrInvalidPrerequisiteLevel = sharedErrors.New("INVALID_PREREQUISITE_LEVEL", "Ön koşul dersin sınıf seviyesi, dersin sınıf seviyesinden düşük olmalıdır", http.StatusBadRequest)
	ErrInvalidStatus            = sharedErrors.New("INVALID_STATUS", "Geçersiz ders durumu", http.StatusBadRequest)
	ErrCourseNotActive          = sharedErrors.New("COURSE_NOT_ACTIVE", "Ders aktif değil", http.StatusBadRequest)

	// Semester course errors (AppError for HTTP responses)
	ErrInvalidSemesterFormat  = sharedErrors.New("INVALID_SEMESTER_FORMAT", "Geçersiz dönem formatı. Beklenen: YYYY-YYYY-Fall veya YYYY-YYYY-Spring", http.StatusBadRequest)
	ErrSemesterCourseNotFound = sharedErrors.New("SEMESTER_COURSE_NOT_FOUND", "Dönem dersi bulunamadı", http.StatusNotFound)
	ErrCourseAlreadyOpened    = sharedErrors.New("COURSE_ALREADY_OPENED", "Ders bu dönem için zaten açılmış", http.StatusConflict)
	ErrClassLevelMismatch     = sharedErrors.New("CLASS_LEVEL_MISMATCH", "Sınıf seviyesi katalogdakiyle uyuşmuyor", http.StatusBadRequest)
	ErrSemesterAlreadyExists  = sharedErrors.New("SEMESTER_ALREADY_EXISTS", "Hedef dönemde zaten açılmış dersler var", http.StatusConflict)
	ErrSourceSemesterNotFound = sharedErrors.New("SOURCE_SEMESTER_NOT_FOUND", "Kaynak dönem bulunamadı", http.StatusNotFound)
	ErrCourseHasEnrollments   = sharedErrors.New("COURSE_HAS_ENROLLMENTS", "Derse kayıtlı öğrenciler var", http.StatusConflict)

	// Instructor errors (AppError for HTTP responses)
	ErrInstructorNotFound         = sharedErrors.New("INSTRUCTOR_NOT_FOUND", "Öğretim üyesi bulunamadı", http.StatusNotFound)
	ErrInstructorNotActive        = sharedErrors.New("INSTRUCTOR_NOT_ACTIVE", "Öğretim üyesi aktif değil", http.StatusBadRequest)
	ErrInstructorNotInDepartment  = sharedErrors.New("INSTRUCTOR_NOT_IN_DEPARTMENT", "Öğretim üyesi bu bölüme ait değil", http.StatusBadRequest)
	ErrInstructorScheduleConflict = sharedErrors.New("INSTRUCTOR_SCHEDULE_CONFLICT", "Öğretim üyesinin ders programında çakışma var", http.StatusConflict)

	// Schedule errors (AppError for HTTP responses)
	ErrInvalidSlotNumber       = sharedErrors.New("INVALID_SLOT_NUMBER", "Geçersiz ders saati (1-9 arasında olmalıdır)", http.StatusBadRequest)
	ErrInvalidDayOfWeek        = sharedErrors.New("INVALID_DAY_OF_WEEK", "Geçersiz gün", http.StatusBadRequest)
	ErrInvalidSessionType      = sharedErrors.New("INVALID_SESSION_TYPE", "Geçersiz ders türü (teorik veya uygulama olmalıdır)", http.StatusBadRequest)
	ErrTheoryHoursZero         = sharedErrors.New("THEORY_HOURS_ZERO", "Teorik ders programı oluşturulamaz: dersin katalogda teorik saati yok", http.StatusBadRequest)
	ErrLabHoursZero            = sharedErrors.New("LAB_HOURS_ZERO", "Uygulama programı oluşturulamaz: dersin katalogda uygulama saati yok", http.StatusBadRequest)
	ErrTheorySlotCountMismatch = sharedErrors.New("THEORY_SLOT_COUNT_MISMATCH", "Teorik ders saati sayısı katalogdaki teorik saatle aynı olmalıdır", http.StatusBadRequest)
	ErrLabSlotCountMismatch    = sharedErrors.New("LAB_SLOT_COUNT_MISMATCH", "Uygulama saati sayısı katalogdaki uygulama saatiyle aynı olmalıdır", http.StatusBadRequest)
	ErrCourseCreditsZero       = sharedErrors.New("COURSE_CREDITS_ZERO", "Dersin katalogda kredisi yok, dönem dersi açılamaz", http.StatusBadRequest)

	// Assessment errors (AppError for HTTP responses)
	ErrInvalidAssessmentSchema    = sharedErrors.New("INVALID_ASSESSMENT_SCHEMA", "Geçersiz değerlendirme şeması", http.StatusBadRequest)
	ErrAssessmentWeightNotHundred = sharedErrors.New("ASSESSMENT_WEIGHT_NOT_HUNDRED", "Değerlendirme ağırlıklarının toplamı 100 olmalıdır", http.StatusBadRequest)
	ErrDuplicateAssessmentSlug    = sharedErrors.New("DUPLICATE_ASSESSMENT_SLUG", "Değerlendirme kısa adı tekrar ediyor", http.StatusBadRequest)

	// Semester status errors (AppError for HTTP responses)
	ErrSemesterNotActive = sharedErrors.New("SEMESTER_NOT_ACTIVE", "Dönem aktif değil, değişiklik yapılamaz", http.StatusForbidden)
	// IMPORTANT: "semester_courses" (courses offered this semester) vs "course_catalog" (all courses ever defined).
	// semester_courses: FROZEN once semester is activated. No add/remove/modify — not even admin.
	// course_catalog: can be modified anytime, independent of semesters.
	ErrSemesterCourseFrozen = sharedErrors.New("SEMESTER_COURSE_FROZEN", "Dönem aktifleştirildikten sonra açılan dersler değiştirilemez", http.StatusForbidden)
	ErrSemesterNotPlanned   = sharedErrors.New("SEMESTER_NOT_PLANNED", "Dönem dersleri yalnızca dönem planlama aşamasındayken değiştirilebilir", http.StatusForbidden)

	// Deadline errors (AppError for HTTP responses)
	ErrCourseCreationPeriodEnded   = sharedErrors.New("COURSE_CREATION_PERIOD_ENDED", "Bu dönem için ders açma süresi sona erdi", http.StatusForbidden)
	ErrCourseCreationPeriodNotOpen = sharedErrors.New("COURSE_CREATION_PERIOD_NOT_OPEN", "Ders açma dönemi henüz başlamadı", http.StatusForbidden)

	// Transaction errors (AppError for HTTP responses)
	ErrTransactionFailed = sharedErrors.New("TRANSACTION_FAILED", "Beklenmeyen bir hata oluştu, lütfen tekrar deneyin", http.StatusInternalServerError)

	// Repository-specific sentinel errors (for internal use)
	ErrCourseNotFoundRepo         = sharedErrors.ErrNotFoundRepo
	ErrCourseExistsRepo           = sharedErrors.ErrAlreadyExistsRepo
	ErrSemesterCourseNotFoundRepo = sharedErrors.ErrNotFoundRepo
	ErrCourseAlreadyOpenedRepo    = sharedErrors.ErrAlreadyExistsRepo
)
