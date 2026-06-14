package repository

import (
	"context"
	"errors"
	"fmt"

	sharedErrors "github.com/baaaki/mydreamcampus/monolith/internal/platform/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"
	serviceErrors "github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/errors"
	"github.com/baaaki/mydreamcampus/monolith/internal/modules/staff/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TeacherProfileRepository struct {
	queries *db.Queries
	pool    *pgxpool.Pool
}

func NewTeacherProfileRepository(pool *pgxpool.Pool) *TeacherProfileRepository {
	return &TeacherProfileRepository{
		queries: db.New(pool),
		pool:    pool,
	}
}

// GetTeacherProfileByStaffID retrieves teacher profile by staff ID
func (r *TeacherProfileRepository) GetTeacherProfileByStaffID(ctx context.Context, staffID uuid.UUID) (db.GetTeacherProfileByStaffIDRow, error) {
	profile, err := r.queries.GetTeacherProfileByStaffID(ctx, utils.UUIDToPgtype(staffID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetTeacherProfileByStaffIDRow{}, fmt.Errorf("%w: teacher profile for staff %s not found", serviceErrors.ErrTeacherProfileNotFoundRepo, staffID)
		}
		return db.GetTeacherProfileByStaffIDRow{}, fmt.Errorf("%w: failed to get teacher profile: %v", sharedErrors.ErrQueryFailed, err)
	}
	return profile, nil
}

// CreateTeacherProfile creates an empty teacher profile for a staff member
func (r *TeacherProfileRepository) CreateTeacherProfile(ctx context.Context, staffID uuid.UUID) (db.TeacherProfile, error) {
	params := db.CreateTeacherProfileParams{
		StaffID: utils.UUIDToPgtype(staffID),
		// All other fields will be NULL/empty by default
	}

	profile, err := r.queries.CreateTeacherProfile(ctx, params)
	if err != nil {
		return db.TeacherProfile{}, fmt.Errorf("%w: failed to create teacher profile: %v", sharedErrors.ErrQueryFailed, err)
	}
	return profile, nil
}

// UpdateTeacherProfile updates teacher profile
func (r *TeacherProfileRepository) UpdateTeacherProfile(ctx context.Context, params db.UpdateTeacherProfileParams) (db.TeacherProfile, error) {
	profile, err := r.queries.UpdateTeacherProfile(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.TeacherProfile{}, fmt.Errorf("%w: teacher profile not found for update", serviceErrors.ErrTeacherProfileNotFoundRepo)
		}
		return db.TeacherProfile{}, fmt.Errorf("%w: failed to update teacher profile: %v", sharedErrors.ErrQueryFailed, err)
	}
	return profile, nil
}

// DeleteTeacherProfileByStaffID deletes teacher profile by staff ID
func (r *TeacherProfileRepository) DeleteTeacherProfileByStaffID(ctx context.Context, staffID uuid.UUID) error {
	err := r.queries.DeleteTeacherProfileByStaffID(ctx, utils.UUIDToPgtype(staffID))
	if err != nil {
		return fmt.Errorf("%w: failed to delete teacher profile: %v", sharedErrors.ErrQueryFailed, err)
	}
	return nil
}

// ListTeacherProfiles lists teacher profiles with pagination
func (r *TeacherProfileRepository) ListTeacherProfiles(ctx context.Context, limit, offset int32) ([]db.ListTeacherProfilesRow, int64, error) {
	// Get total count
	total, err := r.queries.CountTeacherProfiles(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to count teacher profiles: %v", sharedErrors.ErrQueryFailed, err)
	}

	// Get profiles list
	profiles, err := r.queries.ListTeacherProfiles(ctx, db.ListTeacherProfilesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, 0, fmt.Errorf("%w: failed to list teacher profiles: %v", sharedErrors.ErrQueryFailed, err)
	}

	return profiles, total, nil
}
