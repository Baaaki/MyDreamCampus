package repository

import (
	"slices"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCourseLockIDs_SameOrderWhateverTheRequestOrder(t *testing.T) {
	a := ProgramCourseSnapshot{CourseID: uuid.New()}
	b := ProgramCourseSnapshot{CourseID: uuid.New()}

	forward := courseLockIDs([]ProgramCourseSnapshot{a, b, a})
	backward := courseLockIDs([]ProgramCourseSnapshot{b, a})

	assert.Equal(t, backward, forward)
	assert.Len(t, forward, 2, "a repeated course is locked once")
	assert.True(t, slices.IsSorted(forward))
}
