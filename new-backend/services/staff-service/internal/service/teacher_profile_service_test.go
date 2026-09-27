package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/logger"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/baaaki/mydreamcampus/staff/internal/db"
	"github.com/baaaki/mydreamcampus/staff/internal/dto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	_ = logger.Init("test")
}

func TestDecodeJSONBField(t *testing.T) {
	t.Run("empty or nil bytes leaves dst unchanged", func(t *testing.T) {
		var articles []dto.Article
		decodeJSONBField(nil, &articles, "articles")
		assert.Nil(t, articles)

		decodeJSONBField([]byte{}, &articles, "articles")
		assert.Nil(t, articles)
	})

	t.Run("valid JSON decodes correctly into slice", func(t *testing.T) {
		validData := []dto.Article{
			{Title: "AI in Education", Journal: "IEEE", Year: 2026},
		}
		raw, err := json.Marshal(validData)
		require.NoError(t, err)

		var result []dto.Article
		decodeJSONBField(raw, &result, "articles")
		require.Len(t, result, 1)
		assert.Equal(t, "AI in Education", result[0].Title)
		assert.Equal(t, "IEEE", result[0].Journal)
		assert.Equal(t, 2026, result[0].Year)
	})

	t.Run("corrupted JSON logs warning and preserves dst without panic", func(t *testing.T) {
		var result []dto.Article
		corrupted := []byte("{this-is-not-valid-json")
		assert.NotPanics(t, func() {
			decodeJSONBField(corrupted, &result, "articles")
		})
		assert.Nil(t, result)
	})
}

func TestToTeacherProfileResponse(t *testing.T) {
	svc := &TeacherProfileService{}

	profileID := uuid.New()
	staffID := uuid.New()
	now := time.Now().UTC()

	educationList := []dto.Education{
		{Degree: "PhD", Institution: "MIT", Year: 2020},
	}
	eduRaw, _ := json.Marshal(educationList)

	articlesList := []dto.Article{
		{Title: "Distributed Systems", Journal: "ACM", Year: 2024},
	}
	artRaw, _ := json.Marshal(articlesList)

	row := db.GetTeacherProfileByStaffIDRow{
		ID:               utils.UUIDToPgtype(profileID),
		StaffID:          utils.UUIDToPgtype(staffID),
		AcademicTitle:    utils.StringToPgText("Prof. Dr."),
		FirstName:        "Ada",
		LastName:         "Lovelace",
		Faculty:          utils.StringToPgText("Engineering"),
		Department:       utils.StringToPgText("Computer Science"),
		Email:            "ada@university.edu",
		Phone:            utils.StringToPgText("+905551112233"),
		OfficeLocation:   utils.StringToPgText("Building B Room 204"),
		ProfileImageUrl:  utils.StringToPgText("https://img.university.edu/ada.jpg"),
		Education:        eduRaw,
		Articles:         artRaw,
		Bulletins:        nil,
		Projects:         nil,
		Awards:           nil,
		Scholarships:     nil,
		AdminAssignments: nil,
		CreatedAt:        pgtype.Timestamp{Time: now, Valid: true},
		UpdatedAt:        pgtype.Timestamp{Time: now, Valid: true},
	}

	resp := svc.toTeacherProfileResponse(row)

	assert.Equal(t, profileID.String(), resp.ID)
	assert.Equal(t, staffID.String(), resp.StaffID)
	assert.Equal(t, "Prof. Dr.", resp.AcademicTitle)
	assert.Equal(t, "Ada", resp.FirstName)
	assert.Equal(t, "Lovelace", resp.LastName)
	assert.Equal(t, "Engineering", resp.Faculty)
	assert.Equal(t, "Computer Science", resp.Department)
	assert.Equal(t, "ada@university.edu", resp.Email)
	assert.Equal(t, "+905551112233", resp.Phone)
	assert.Equal(t, "Building B Room 204", resp.OfficeLocation)
	assert.Equal(t, "https://img.university.edu/ada.jpg", resp.ProfileImageURL)

	require.Len(t, resp.Education, 1)
	assert.Equal(t, "PhD", resp.Education[0].Degree)
	assert.Equal(t, "MIT", resp.Education[0].Institution)

	require.Len(t, resp.Articles, 1)
	assert.Equal(t, "Distributed Systems", resp.Articles[0].Title)

	// Ensure non-nil empty slices for unset fields
	assert.NotNil(t, resp.Bulletins)
	assert.Empty(t, resp.Bulletins)
	assert.NotNil(t, resp.Projects)
	assert.Empty(t, resp.Projects)
	assert.NotNil(t, resp.Awards)
	assert.Empty(t, resp.Awards)
	assert.NotNil(t, resp.Scholarships)
	assert.Empty(t, resp.Scholarships)
	assert.NotNil(t, resp.AdminAssignments)
	assert.Empty(t, resp.AdminAssignments)
}

func TestTeacherProfileService_InvalidUUID(t *testing.T) {
	svc := &TeacherProfileService{}
	ctx := context.Background()

	t.Run("GetTeacherProfileByStaffID rejects invalid UUID", func(t *testing.T) {
		_, err := svc.GetTeacherProfileByStaffID(ctx, "not-a-valid-uuid")
		require.ErrorIs(t, err, sharedErrors.ErrInvalidID)
	})

	t.Run("UpdateTeacherProfile rejects invalid UUID", func(t *testing.T) {
		_, err := svc.UpdateTeacherProfile(ctx, "not-a-valid-uuid", dto.UpdateTeacherProfileRequest{})
		require.ErrorIs(t, err, sharedErrors.ErrInvalidID)
	})
}
