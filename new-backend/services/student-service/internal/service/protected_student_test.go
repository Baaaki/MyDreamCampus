package service

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/student/internal/db"
	"github.com/baaaki/mydreamcampus/student/internal/dto"
	serviceErrors "github.com/baaaki/mydreamcampus/student/internal/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockStudentRepo struct {
	StudentRepositoryInterface
	student db.Student
	err     error
}

func (m *mockStudentRepo) GetStudentByID(ctx context.Context, id uuid.UUID) (db.Student, error) {
	if m.err != nil {
		return db.Student{}, m.err
	}
	return m.student, nil
}

func (m *mockStudentRepo) SoftDeleteStudentWithEvent(ctx context.Context, id uuid.UUID, eventPayload map[string]any) error {
	return nil
}

func (m *mockStudentRepo) UpdateStudentWithEvent(ctx context.Context, id uuid.UUID, params db.UpdateStudentParams, eventPayload map[string]any) (db.Student, error) {
	return m.student, nil
}

func TestDeleteStudent_ProtectedAccount_ReturnsForbidden(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{
		Admin: config.AdminConfig{
			Email: "admin@university.edu.tr",
		},
		Demo: config.DemoConfig{
			StudentEmail: "zeynep.sahin@uni.edu.tr",
		},
		ProtectedAccountEmails: []string{
			"admin@university.edu.tr",
			"zeynep.sahin@uni.edu.tr",
		},
	}

	studentID := uuid.New()
	svc := NewStudentService(&mockStudentRepo{
		student: db.Student{
			ID:    utils.UUIDToPgtype(studentID),
			Email: "zeynep.sahin@uni.edu.tr",
		},
	}, nil, cfg)

	err := svc.DeleteStudent(context.Background(), studentID.String())
	assert.ErrorIs(t, err, serviceErrors.ErrProtectedAccountDeletionForbidden)
}

func TestUpdateStudent_ProtectedAccount_DeactivationForbidden(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{
		Admin: config.AdminConfig{
			Email: "admin@university.edu.tr",
		},
		Demo: config.DemoConfig{
			StudentEmail: "zeynep.sahin@uni.edu.tr",
		},
		ProtectedAccountEmails: []string{
			"admin@university.edu.tr",
			"zeynep.sahin@uni.edu.tr",
		},
	}

	studentID := uuid.New()
	svc := NewStudentService(&mockStudentRepo{
		student: db.Student{
			ID:     utils.UUIDToPgtype(studentID),
			Email:  "zeynep.sahin@uni.edu.tr",
			Status: utils.StringToPgText("active"),
		},
	}, nil, cfg)

	// Attempt to deactivate (e.g. suspended, withdrawn, graduated)
	withdrawnStatus := "withdrawn"
	_, err := svc.UpdateStudent(context.Background(), studentID.String(), dto.UpdateStudentRequest{
		Status: &withdrawnStatus,
	})
	assert.ErrorIs(t, err, serviceErrors.ErrProtectedAccountDeactivationForbidden)
}

func TestUpdateStudent_ProtectedAccount_EmailChangeForbidden(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{
		Admin: config.AdminConfig{
			Email: "admin@university.edu.tr",
		},
		Demo: config.DemoConfig{
			StudentEmail: "zeynep.sahin@uni.edu.tr",
		},
		ProtectedAccountEmails: []string{
			"admin@university.edu.tr",
			"zeynep.sahin@uni.edu.tr",
		},
	}

	studentID := uuid.New()
	svc := NewStudentService(&mockStudentRepo{
		student: db.Student{
			ID:    utils.UUIDToPgtype(studentID),
			Email: "zeynep.sahin@uni.edu.tr",
		},
	}, nil, cfg)

	// Attempt to change email of protected student
	newEmail := "hacked@uni.edu.tr"
	_, err := svc.UpdateStudent(context.Background(), studentID.String(), dto.UpdateStudentRequest{
		Email: &newEmail,
	})
	assert.ErrorIs(t, err, serviceErrors.ErrProtectedAccountEmailChangeForbidden)

	// Attempt to claim protected email for a regular student
	otherStudentID := uuid.New()
	svc2 := NewStudentService(&mockStudentRepo{
		student: db.Student{
			ID:    utils.UUIDToPgtype(otherStudentID),
			Email: "normal.student@uni.edu.tr",
		},
	}, nil, cfg)

	protectedEmail := "zeynep.sahin@uni.edu.tr"
	_, err = svc2.UpdateStudent(context.Background(), otherStudentID.String(), dto.UpdateStudentRequest{
		Email: &protectedEmail,
	})
	assert.ErrorIs(t, err, serviceErrors.ErrProtectedAccountEmailChangeForbidden)
}
