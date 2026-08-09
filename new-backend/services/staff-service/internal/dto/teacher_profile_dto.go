package dto

import "time"

// Education represents education history entry
type Education struct {
	ID          string `json:"id"`
	Degree      string `json:"degree"`
	Institution string `json:"institution"`
	Department  string `json:"department"`
	Year        int    `json:"year"`
}

// Article represents a published article
type Article struct {
	ID                    string `json:"id"`
	Title                 string `json:"title"`
	Journal               string `json:"journal"`
	Year                  int    `json:"year"`
	Authors               string `json:"authors"`
	DOI                   string `json:"doi,omitempty"`
	JournalType           string `json:"journalType,omitempty"`
	DomesticInternational string `json:"domesticInternational,omitempty"`
	PublishingMonth       string `json:"publishingMonth,omitempty"`
	IssuePageYear         string `json:"issuePageYear,omitempty"`
	Language              string `json:"language,omitempty"`
	ArticleType           string `json:"articleType,omitempty"`
}

// Bulletin represents a conference bulletin
type Bulletin struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Conference string `json:"conference"`
	Year       int    `json:"year"`
	Location   string `json:"location"`
}

// Project represents a research project
type Project struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Role      string `json:"role"`
	Funder    string `json:"funder"`
	StartYear int    `json:"startYear"`
	EndYear   *int   `json:"endYear,omitempty"`
	Status    string `json:"status"` // ongoing, completed
}

// Award represents an award entry
type Award struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Institution string `json:"institution"`
	Year        int    `json:"year"`
}

// Scholarship represents a scholarship entry
type Scholarship struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Institution string `json:"institution"`
	Year        int    `json:"year"`
}

// AdminAssignment represents an administrative assignment
type AdminAssignment struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Institution string `json:"institution"`
	StartYear   int    `json:"startYear"`
	EndYear     *int   `json:"endYear,omitempty"`
}

// UpdateTeacherProfileRequest represents the request for updating teacher profile
type UpdateTeacherProfileRequest struct {
	AcademicTitle    *string            `json:"academic_title"`
	Faculty          *string            `json:"faculty"`
	ProfileImageURL  *string            `json:"profile_image_url"`
	Education        *[]Education       `json:"education"`
	Articles         *[]Article         `json:"articles"`
	Bulletins        *[]Bulletin        `json:"bulletins"`
	Projects         *[]Project         `json:"projects"`
	Awards           *[]Award           `json:"awards"`
	Scholarships     *[]Scholarship     `json:"scholarships"`
	AdminAssignments *[]AdminAssignment `json:"admin_assignments"`
}

// TeacherProfileResponse represents the response for teacher profile
type TeacherProfileResponse struct {
	ID               string            `json:"id"`
	StaffID          string            `json:"staff_id"`
	AcademicTitle    string            `json:"academic_title,omitempty"`
	FirstName        string            `json:"first_name"`
	LastName         string            `json:"last_name"`
	Faculty          string            `json:"faculty,omitempty"`
	Department       string            `json:"department,omitempty"`
	Email            string            `json:"email"`
	Phone            string            `json:"phone,omitempty"`
	OfficeLocation   string            `json:"office_location,omitempty"`
	ProfileImageURL  string            `json:"profile_image_url,omitempty"`
	Education        []Education       `json:"education"`
	Articles         []Article         `json:"articles"`
	Bulletins        []Bulletin        `json:"bulletins"`
	Projects         []Project         `json:"projects"`
	Awards           []Award           `json:"awards"`
	Scholarships     []Scholarship     `json:"scholarships"`
	AdminAssignments []AdminAssignment `json:"admin_assignments"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// TeacherProfileListResponse represents paginated teacher profile list
type TeacherProfileListResponse struct {
	Data       []TeacherProfileResponse `json:"data"`
	Pagination PaginationResponse       `json:"pagination"`
}
