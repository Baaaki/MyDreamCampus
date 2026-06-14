package repository

import (
	"context"
	"time"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/db"
	"github.com/baaaki/mydreamcampus/shared/audit"
	"github.com/baaaki/mydreamcampus/shared/logger"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type SemesterStatusRepository struct {
	queries     *db.Queries
	pool        *pgxpool.Pool
	auditLogger audit.Logger
}

func NewSemesterStatusRepository(pool *pgxpool.Pool, auditLogger audit.Logger) *SemesterStatusRepository {
	return &SemesterStatusRepository{
		queries:     db.New(pool),
		pool:        pool,
		auditLogger: auditLogger,
	}
}

func (r *SemesterStatusRepository) CreateSemester(ctx context.Context, name string, hardDeadline time.Time) (db.Semester, error) {
	return r.queries.CreateSemester(ctx, db.CreateSemesterParams{
		Name:         name,
		HardDeadline: utils.TimeToPgTimestamptz(hardDeadline),
	})
}

func (r *SemesterStatusRepository) GetSemesterByName(ctx context.Context, name string) (db.Semester, error) {
	return r.queries.GetSemesterByName(ctx, name)
}

func (r *SemesterStatusRepository) GetActiveSemester(ctx context.Context) (db.Semester, error) {
	return r.queries.GetActiveSemester(ctx)
}

func (r *SemesterStatusRepository) ListSemesters(ctx context.Context) ([]db.Semester, error) {
	return r.queries.ListSemesters(ctx)
}

// HasActiveSemester checks if any semester is currently active.
// INVARIANT: Only one semester can be active at any given time.
// A new semester cannot be activated until the current active one is completed.
func (r *SemesterStatusRepository) HasActiveSemester(ctx context.Context) (bool, error) {
	return r.queries.HasActiveSemester(ctx)
}

func (r *SemesterStatusRepository) ActivateSemester(ctx context.Context, id uuid.UUID) (db.Semester, error) {
	return r.queries.ActivateSemester(ctx, utils.UUIDToPgtype(id))
}

func (r *SemesterStatusRepository) CompleteSemester(ctx context.Context, id uuid.UUID) (db.Semester, error) {
	return r.queries.CompleteSemester(ctx, utils.UUIDToPgtype(id))
}

// IsSemesterActive checks if a semester is active and its hard_deadline has not passed.
// If hard_deadline has passed, it auto-completes the semester and writes an audit log.
// This method satisfies the semester.Checker interface.
func (r *SemesterStatusRepository) IsSemesterActive(ctx context.Context, semesterName string) (bool, error) {
	log := logger.WithContextAndFields(ctx,
		zap.String("repository", "SemesterStatusRepository"),
		zap.String("method", "IsSemesterActive"),
	)

	semester, err := r.queries.GetSemesterByName(ctx, semesterName)
	if err != nil {
		return false, err
	}

	if semester.Status == db.SemesterStatusCompleted {
		return false, nil
	}
	if semester.Status == db.SemesterStatusPlanned {
		return false, nil
	}

	// status == active — check hard_deadline
	if semester.HardDeadline.Valid && time.Now().After(semester.HardDeadline.Time) {
		// Auto-complete: hard deadline has passed
		if err := r.queries.AutoCompleteSemester(ctx, semesterName); err != nil {
			log.Warn("failed to auto-complete semester",
				zap.String("semester", semesterName),
				zap.Error(err),
			)
		} else {
			// Audit log: semester.auto_completed
			if r.auditLogger != nil {
				r.auditLogger.Log(ctx, audit.AuditEvent{
					ActorID:      "system",
					ActorRole:    "system",
					Action:       "semester.auto_completed",
					ResourceType: "semester",
					ResourceID:   utils.PgtypeToUUIDString(semester.ID),
					Details: map[string]any{
						"semester_name": semesterName,
						"hard_deadline": semester.HardDeadline.Time.Format(time.RFC3339),
						"reason":        "hard_deadline has passed",
					},
				})
			}
			log.Info("semester auto-completed due to hard deadline",
				zap.String("semester", semesterName),
			)
		}
		return false, nil
	}

	return true, nil
}

// GetSemesterStatus returns the status of a semester by name.
func (r *SemesterStatusRepository) GetSemesterStatus(ctx context.Context, semesterName string) (db.SemesterStatus, error) {
	semester, err := r.queries.GetSemesterByName(ctx, semesterName)
	if err != nil {
		return "", err
	}
	return semester.Status, nil
}

// SemesterInfo contains the essential semester data needed by other services.
type SemesterInfo struct {
	Name           string
	Status         string
	HardDeadline   time.Time
	IsPastDeadline bool
}

// GetSemesterInfo returns semester info including hard_deadline for enforcement by other services.
func (r *SemesterStatusRepository) GetSemesterInfo(ctx context.Context, semesterName string) (*SemesterInfo, error) {
	semester, err := r.queries.GetSemesterByName(ctx, semesterName)
	if err != nil {
		return nil, err
	}

	hardDeadline := utils.PgTimestamptzToTime(semester.HardDeadline)
	isPast := time.Now().After(hardDeadline)

	return &SemesterInfo{
		Name:           semester.Name,
		Status:         string(semester.Status),
		HardDeadline:   hardDeadline,
		IsPastDeadline: isPast,
	}, nil
}
