package service

import (
	"context"

	"github.com/baaaki/mydreamcampus/catalog/internal/db"
	"github.com/baaaki/mydreamcampus/catalog/internal/dto"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	"github.com/baaaki/mydreamcampus/shared/platform/utils"
	"github.com/jackc/pgx/v5/pgtype"
)

// FacultyReader is the slice of FacultyRepository the service reads.
type FacultyReader interface {
	ListFaculties(ctx context.Context) ([]db.Faculty, error)
	ListDepartments(ctx context.Context) ([]db.Department, error)
}

type FacultyService struct {
	reader FacultyReader
}

func NewFacultyService(reader FacultyReader) *FacultyService {
	return &FacultyService{reader: reader}
}

// ListFaculties returns every faculty with its departments nested. Two flat
// queries instead of a JOIN: a faculty without departments must still appear,
// and the set is small enough (~20 faculties, ~100 departments) to group here.
func (s *FacultyService) ListFaculties(ctx context.Context) (dto.ListFacultiesResponse, error) {
	faculties, err := s.reader.ListFaculties(ctx)
	if err != nil {
		return dto.ListFacultiesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	departments, err := s.reader.ListDepartments(ctx)
	if err != nil {
		return dto.ListFacultiesResponse{}, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	return dto.ListFacultiesResponse{Data: buildFacultyResponses(faculties, departments)}, nil
}

// buildFacultyResponses keeps the faculty order of its input and the
// department order within each faculty.
func buildFacultyResponses(faculties []db.Faculty, departments []db.Department) []dto.FacultyResponse {
	byFaculty := make(map[pgtype.UUID][]dto.DepartmentResponse, len(faculties))
	slugs := make(map[pgtype.UUID]string, len(faculties))
	for _, f := range faculties {
		slugs[f.ID] = f.Slug
	}
	for _, d := range departments {
		facultySlug, ok := slugs[d.FacultyID]
		if !ok {
			continue
		}
		byFaculty[d.FacultyID] = append(byFaculty[d.FacultyID], dto.DepartmentResponse{
			ID:          d.Slug,
			Name:        d.Name,
			FacultyID:   facultySlug,
			Code:        d.Code,
			Description: utils.PgtypeTextToStringPtr(d.Description),
		})
	}

	out := make([]dto.FacultyResponse, 0, len(faculties))
	for _, f := range faculties {
		depts := byFaculty[f.ID]
		if depts == nil {
			depts = []dto.DepartmentResponse{}
		}
		out = append(out, dto.FacultyResponse{
			ID:          f.Slug,
			Name:        f.Name,
			Code:        f.Code,
			Departments: depts,
		})
	}
	return out
}
