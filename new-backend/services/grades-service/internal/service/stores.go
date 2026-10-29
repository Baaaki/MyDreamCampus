package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/grades/internal/db"
	"github.com/baaaki/mydreamcampus/grades/internal/repository"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The services depend on these narrow views of the repositories rather than
// on the repositories themselves, so the grading workflows can be tested
// without a database. WithTx returns the view again, which the repositories'
// own WithTx cannot do; the adapters below close that gap.

type txBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type cacheStore interface {
	GetCourseCacheByID(ctx context.Context, id uuid.UUID) (db.GradesCoursesView, error)
	GetStudentCacheByID(ctx context.Context, id uuid.UUID) (db.GradesStudentsView, error)
	IsPrerequisiteCourse(ctx context.Context, courseCode string) (bool, error)
}

type registrationStore interface {
	GetRegistrationByID(ctx context.Context, id uuid.UUID) (db.GetRegistrationByIDRow, error)
	GetRegistrationsByIDs(ctx context.Context, ids []uuid.UUID) ([]db.GetRegistrationsByIDsRow, error)
	GetRegistrationsByCourse(ctx context.Context, courseID uuid.UUID) ([]db.GetRegistrationsByCourseRow, error)
	GetActiveRegistrationsByStudent(ctx context.Context, studentID uuid.UUID) ([]db.GetActiveRegistrationsByStudentRow, error)
	CountRegistrationsByCourse(ctx context.Context, courseID uuid.UUID) (int64, error)
	CountEligibleRegistrationsByCourse(ctx context.Context, courseID uuid.UUID) (int64, error)
	DeleteRegistrationsByCourse(ctx context.Context, courseID uuid.UUID) error
	StudentHasGradesInSemester(ctx context.Context, studentID uuid.UUID, semester string) (bool, error)
	WithTx(tx pgx.Tx) registrationStore
}

type scoreStore interface {
	GetScoreByRegistrationAndSlug(ctx context.Context, registrationID uuid.UUID, slug string) (db.GradesStudentAssessmentScore, error)
	GetScoresByRegistration(ctx context.Context, registrationID uuid.UUID) ([]db.GradesStudentAssessmentScore, error)
	GetLockedRegistrationsBySlug(ctx context.Context, slug string, ids []uuid.UUID) ([]uuid.UUID, error)
	CountScoresBySlugAndCourse(ctx context.Context, arg db.CountScoresBySlugAndCourseParams) (int64, error)
	CountLockedScoresBySlugAndCourse(ctx context.Context, courseID uuid.UUID, slug string) (int64, error)
	UpsertAssessmentScoreWithEvent(ctx context.Context, scoreParams db.UpsertAssessmentScoreParams, eventParams *db.CreateOutboxEventParams) (db.GradesStudentAssessmentScore, error)
	BulkUpsertAssessmentScoresWithEvents(ctx context.Context, entries []repository.BulkUpsertEntry) (int, error)
	LockScore(ctx context.Context, registrationID uuid.UUID, slug string) error
	UnlockScore(ctx context.Context, registrationID uuid.UUID, slug string) error
	LockScoresByCourseAndSlug(ctx context.Context, courseID uuid.UUID, slug string) error
	DeleteScoresByCourse(ctx context.Context, courseID uuid.UUID) error
	LockCourseGrading(ctx context.Context, courseID uuid.UUID) error
	WithTx(tx pgx.Tx) scoreStore
}

type completedStore interface {
	GetCompletedCourseByStudentAndCourse(ctx context.Context, studentID uuid.UUID, courseID uuid.UUID) (db.GradesStudentCompletedCourse, error)
	GetCompletedCoursesByCourse(ctx context.Context, courseID uuid.UUID) ([]db.GradesStudentCompletedCourse, error)
	GetCompletedCoursesByStudent(ctx context.Context, studentID uuid.UUID) ([]db.GradesStudentCompletedCourse, error)
	GetTranscriptData(ctx context.Context, studentID uuid.UUID) ([]db.GetTranscriptDataRow, error)
	CalculateStudentGPA(ctx context.Context, studentID uuid.UUID) (db.CalculateStudentGPARow, error)
	CreateCompletedCourse(ctx context.Context, arg db.CreateCompletedCourseParams) (db.GradesStudentCompletedCourse, error)
	DeleteCompletedCourse(ctx context.Context, arg db.DeleteCompletedCourseParams) error
	UpdateCompletedCourseAfterAppeal(ctx context.Context, arg db.UpdateCompletedCourseAfterAppealParams) error
	WithTx(tx pgx.Tx) completedStore
}

type outboxStore interface {
	CreateOutboxEvent(ctx context.Context, arg db.CreateOutboxEventParams) (db.GradesOutboxEvent, error)
	WithTx(tx pgx.Tx) outboxStore
}

type periodStore interface {
	GetActivePeriodBySemester(ctx context.Context, semester string) (*sharedRepo.SimplePeriod, error)
}

type registrationRepoAdapter struct {
	*repository.RegistrationRepository
}

func (r registrationRepoAdapter) WithTx(tx pgx.Tx) registrationStore {
	return registrationRepoAdapter{r.RegistrationRepository.WithTx(tx)}
}

type scoreRepoAdapter struct{ *repository.ScoreRepository }

func (r scoreRepoAdapter) WithTx(tx pgx.Tx) scoreStore {
	return scoreRepoAdapter{r.ScoreRepository.WithTx(tx)}
}

type completedRepoAdapter struct {
	*repository.CompletedRepository
}

func (r completedRepoAdapter) WithTx(tx pgx.Tx) completedStore {
	return completedRepoAdapter{r.CompletedRepository.WithTx(tx)}
}

type outboxRepoAdapter struct{ *repository.OutboxRepository }

func (r outboxRepoAdapter) WithTx(tx pgx.Tx) outboxStore {
	return outboxRepoAdapter{r.OutboxRepository.WithTx(tx)}
}
