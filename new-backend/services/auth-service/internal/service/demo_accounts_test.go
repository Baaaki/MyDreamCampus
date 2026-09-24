package service

import (
	"context"
	"testing"

	"github.com/baaaki/mydreamcampus/auth/internal/db"
	serviceErrors "github.com/baaaki/mydreamcampus/auth/internal/errors"
	"github.com/baaaki/mydreamcampus/shared/config"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type demoUserRepoMock struct {
	userStore
	activeUsers    []db.GetActiveDemoUsersRow
	activeUsersErr error

	existingUser   db.User
	getUserErr     error
	createdParams  db.CreateUserParams
	createUserErr  error
	ensuredEmail   string
	ensureFlagsErr error
}

func (m *demoUserRepoMock) GetActiveDemoUsers(ctx context.Context) ([]db.GetActiveDemoUsersRow, error) {
	if m.activeUsersErr != nil {
		return nil, m.activeUsersErr
	}
	return m.activeUsers, nil
}

func (m *demoUserRepoMock) GetUserByEmail(ctx context.Context, email string) (db.User, error) {
	if m.getUserErr != nil {
		return db.User{}, m.getUserErr
	}
	return m.existingUser, nil
}

func (m *demoUserRepoMock) CreateUser(ctx context.Context, params db.CreateUserParams) (db.CreateUserRow, error) {
	m.createdParams = params
	if m.createUserErr != nil {
		return db.CreateUserRow{}, m.createUserErr
	}
	return db.CreateUserRow{Email: params.Email}, nil
}

func (m *demoUserRepoMock) EnsureDemoUserFlags(ctx context.Context, email string) error {
	m.ensuredEmail = email
	return m.ensureFlagsErr
}

func TestGetDemoAccounts(t *testing.T) {
	s := &AuthService{
		authRepo: &demoUserRepoMock{
			activeUsers: []db.GetActiveDemoUsersRow{
				{Role: "admin", Email: "demo.admin@mydreamcampus.com"},
				{Role: "teacher", Email: "ahmet.yilmaz@uni.edu.tr"},
				{Role: "student", Email: "zeynep.sahin@uni.edu.tr"},
				{Role: "unknown", Email: "other@uni.edu.tr"},
			},
		},
	}

	accounts, err := s.GetDemoAccounts(context.Background())
	require.NoError(t, err)
	require.Len(t, accounts, 4)

	assert.Equal(t, "admin", accounts[0].Role)
	assert.Equal(t, "Demo Yönetici", accounts[0].Label)
	assert.Equal(t, "demo.admin@mydreamcampus.com", accounts[0].Email)
	assert.Equal(t, "demo.admin@mydreamcampus.com", accounts[0].Password)

	assert.Equal(t, "teacher", accounts[1].Role)
	assert.Equal(t, "Demo Öğretmen", accounts[1].Label)
	assert.Equal(t, "ahmet.yilmaz@uni.edu.tr", accounts[1].Email)
	assert.Equal(t, "ahmet.yilmaz@uni.edu.tr", accounts[1].Password)

	assert.Equal(t, "student", accounts[2].Role)
	assert.Equal(t, "Demo Öğrenci", accounts[2].Label)
	assert.Equal(t, "zeynep.sahin@uni.edu.tr", accounts[2].Email)
	assert.Equal(t, "zeynep.sahin@uni.edu.tr", accounts[2].Password)

	assert.Equal(t, "unknown", accounts[3].Role)
	assert.Equal(t, "Demo Kullanıcı", accounts[3].Label)
}

func TestSeedDemoAdmin_Disabled(t *testing.T) {
	cfg := &config.Config{}
	cfg.Demo.Enabled = false
	cfg.Demo.AdminEmail = "demo.admin@mydreamcampus.com"

	repo := &demoUserRepoMock{}
	s := &AuthService{config: cfg, authRepo: repo}

	err := s.SeedDemoAdmin(context.Background())
	assert.NoError(t, err)
	assert.Empty(t, repo.createdParams.Email)
}

func TestSeedDemoAdmin_AlreadyExists_EnsuresFlags(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{}
	cfg.Demo.Enabled = true
	cfg.Demo.AdminEmail = "demo.admin@mydreamcampus.com"

	repo := &demoUserRepoMock{
		existingUser: db.User{Email: "demo.admin@mydreamcampus.com"},
	}
	s := &AuthService{config: cfg, authRepo: repo}

	err := s.SeedDemoAdmin(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "demo.admin@mydreamcampus.com", repo.ensuredEmail)
	assert.Empty(t, repo.createdParams.Email)
}

func TestSeedDemoAdmin_CreatesUser(t *testing.T) {
	require.NoError(t, logger.Init("test"))
	cfg := &config.Config{}
	cfg.Demo.Enabled = true
	cfg.Demo.AdminEmail = "demo.admin@mydreamcampus.com"

	repo := &demoUserRepoMock{
		getUserErr: serviceErrors.ErrUserNotFoundRepo,
	}
	s := &AuthService{config: cfg, authRepo: repo}

	err := s.SeedDemoAdmin(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "demo.admin@mydreamcampus.com", repo.createdParams.Email)
	assert.Equal(t, "admin", repo.createdParams.Role)
	require.NotNil(t, repo.createdParams.IsDemo)
	assert.True(t, *repo.createdParams.IsDemo)
	require.NotNil(t, repo.createdParams.ForcePasswordChange)
	assert.False(t, *repo.createdParams.ForcePasswordChange)
	require.NotNil(t, repo.createdParams.IsSuperadmin)
	assert.False(t, *repo.createdParams.IsSuperadmin)
}
