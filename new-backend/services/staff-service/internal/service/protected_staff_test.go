package service

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/staff/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/staff/internal/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStaffStore struct {
	StaffStore
	staff db.Staff
	err   error
}

func (m *mockStaffStore) GetStaffByID(ctx context.Context, id uuid.UUID) (db.Staff, error) {
	if m.err != nil {
		return db.Staff{}, m.err
	}
	return m.staff, nil
}

func (m *mockStaffStore) SoftDeleteStaffWithEvent(ctx context.Context, id uuid.UUID, eventPayload map[string]any) error {
	return nil
}

func TestDeleteStaff_ProtectedAccount_ReturnsForbidden(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{
		Admin: config.AdminConfig{
			Email: "admin@university.edu.tr",
		},
		Demo: config.DemoConfig{
			TeacherEmail: "ahmet.yilmaz@uni.edu.tr",
		},
		ProtectedAccountEmails: []string{
			"admin@university.edu.tr",
			"ahmet.yilmaz@uni.edu.tr",
		},
	}

	staffID := uuid.New()
	svc := NewStaffService(&mockStaffStore{
		staff: db.Staff{
			Email: "ahmet.yilmaz@uni.edu.tr",
		},
	}, cfg)

	err := svc.DeleteStaff(context.Background(), staffID.String())
	assert.ErrorIs(t, err, serviceErrors.ErrProtectedAccountDeletionForbidden)
}

func TestAdminStaffService_Update_ProtectedEmail_ReturnsForbidden(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{
		Admin: config.AdminConfig{
			Email: "admin@university.edu.tr",
		},
		ProtectedAccountEmails: []string{
			"admin@university.edu.tr",
		},
	}

	store := newFakeAdminStaffStore()
	id := uuid.New()
	store.rows[id] = db.AdminStaff{
		Email: "admin@university.edu.tr",
		Title: "Yönetici",
	}

	svc := NewAdminStaffService(store, cfg)

	// Attempt to change email of a protected account
	req := validAdminStaffRequest()
	req.Email = "new.email@university.edu.tr"

	_, err := svc.Update(context.Background(), id.String(), req)
	assert.ErrorIs(t, err, serviceErrors.ErrProtectedAccountEmailChangeForbidden)

	// Attempt to change a non-protected account to a protected email
	otherID := uuid.New()
	store.rows[otherID] = db.AdminStaff{
		Email: "user@university.edu.tr",
		Title: "Personel",
	}

	req2 := validAdminStaffRequest()
	req2.Email = "admin@university.edu.tr"

	_, err = svc.Update(context.Background(), otherID.String(), req2)
	assert.ErrorIs(t, err, serviceErrors.ErrProtectedAccountEmailChangeForbidden)
}
