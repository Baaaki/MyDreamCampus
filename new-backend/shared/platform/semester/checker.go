package semester

import "context"

// Checker verifies if a semester is currently active.
// Catalog service implements this via direct DB access (SemesterStatusRepository).
// Other services implement this via HTTP call to catalog service (HTTPChecker).
type Checker interface {
	IsSemesterActive(ctx context.Context, semesterName string) (bool, error)
}
