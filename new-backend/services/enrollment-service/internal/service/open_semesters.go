package service

import (
	"context"
	"slices"
	"time"

	"github.com/baaaki/mydreamcampus/enrollment/internal/dto"
	"github.com/baaaki/mydreamcampus/shared/platform/clock"
	sharedErrors "github.com/baaaki/mydreamcampus/shared/platform/errors"
	sharedRepo "github.com/baaaki/mydreamcampus/shared/platform/repository"
	"github.com/google/uuid"
)

// GetOpenSemesters tells the student which semesters the enrollment screen
// can work on, read off this service's own period projection and on the
// service clock, so the time machine moves it like the window check does.
func (s *EnrollmentService) GetOpenSemesters(ctx context.Context, studentID uuid.UUID) ([]dto.OpenSemester, error) {
	periods, err := s.periodRepo.GetAllPeriods(ctx)
	if err != nil {
		return nil, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}

	programs, err := s.enrollmentRepo.GetEnrollmentProgramsByStudent(ctx, studentID, nil, nil)
	if err != nil {
		return nil, sharedErrors.Wrap(sharedErrors.ErrInternal, err)
	}
	hasProgram := make(map[string]bool, len(programs))
	for _, p := range programs {
		hasProgram[p.Semester] = true
	}

	return openSemesters(periods, hasProgram, clock.Now()), nil
}

// openSemesters returns every semester whose window holds now, plus the
// current semester when the student has a program in it. The current
// semester is the one whose window closed most recently: its classes run
// after enrollment ends, and a window opening for the next semester must not
// hide the program the student is taking now. Oldest first.
func openSemesters(periods []sharedRepo.SimplePeriod, hasProgram map[string]bool, now time.Time) []dto.OpenSemester {
	out := make([]dto.OpenSemester, 0, len(periods))
	var current *sharedRepo.SimplePeriod
	for i := range periods {
		p := &periods[i]
		if !p.IsActive || now.Before(p.PeriodStart) {
			continue
		}
		if !now.After(p.PeriodEnd) {
			out = append(out, toOpenSemester(p, true, hasProgram))
			continue
		}
		if current == nil || p.PeriodEnd.After(current.PeriodEnd) {
			current = p
		}
	}
	if current != nil && hasProgram[current.Semester] {
		out = append(out, toOpenSemester(current, false, hasProgram))
	}

	slices.SortFunc(out, func(a, b dto.OpenSemester) int { return a.PeriodStart.Compare(b.PeriodStart) })
	return out
}

func toOpenSemester(p *sharedRepo.SimplePeriod, open bool, hasProgram map[string]bool) dto.OpenSemester {
	return dto.OpenSemester{
		Semester:       p.Semester,
		EnrollmentOpen: open,
		PeriodStart:    p.PeriodStart,
		PeriodEnd:      p.PeriodEnd,
		HasProgram:     hasProgram[p.Semester],
	}
}
