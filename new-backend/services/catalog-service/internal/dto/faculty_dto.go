package dto

// FacultyResponse follows the frontend's Faculty type rather than the table:
// id is the slug, because the frontend routes and filters on it and the slug
// is what survives a re-seed.
type FacultyResponse struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Code        string               `json:"code"`
	Departments []DepartmentResponse `json:"departments"`
}

// DepartmentResponse carries its faculty as the faculty's slug, camelCase to
// match the frontend's Department type.
type DepartmentResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	FacultyID   string  `json:"facultyId"`
	Code        string  `json:"code"`
	Description *string `json:"description,omitempty"`
}

type ListFacultiesResponse struct {
	Data []FacultyResponse `json:"data"`
}
