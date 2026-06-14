package config

// Service port mappings for all microservices
// This ensures consistent port assignments across the system
const (
	// HTTP Service Ports
	AuthServicePort       = "8001"
	StaffServicePort      = "8002"
	StudentServicePort    = "8003"
	CatalogServicePort    = "8004"
	EnrollmentServicePort = "8005"
	AttendanceServicePort = "8006"
	GradesServicePort     = "8007"
	MealServicePort       = "8008"
	NotificationServicePort = "8009" // Future service

	// Database Ports (PostgreSQL)
	AuthDBPort       = "5432"
	StaffDBPort      = "5433"
	StudentDBPort    = "5434"
	CatalogDBPort    = "5435"
	EnrollmentDBPort = "5436"
	AttendanceDBPort = "5437"
	GradesDBPort     = "5438"
	MealDBPort       = "5439"
)

// ServicePortMap maps service names to their default ports
var ServicePortMap = map[string]string{
	"auth":         AuthServicePort,
	"staff":        StaffServicePort,
	"student":      StudentServicePort,
	"catalog":      CatalogServicePort,
	"enrollment":   EnrollmentServicePort,
	"attendance":   AttendanceServicePort,
	"grades":       GradesServicePort,
	"meal":         MealServicePort,
	"notification": NotificationServicePort,
}

// DBPortMap maps service names to their database ports
var DBPortMap = map[string]string{
	"auth":       AuthDBPort,
	"staff":      StaffDBPort,
	"student":    StudentDBPort,
	"catalog":    CatalogDBPort,
	"enrollment": EnrollmentDBPort,
	"attendance": AttendanceDBPort,
	"grades":     GradesDBPort,
	"meal":       MealDBPort,
}

// GetServicePort returns the default HTTP port for a service
func GetServicePort(serviceName string) string {
	if port, ok := ServicePortMap[serviceName]; ok {
		return port
	}
	return "8000" // Default fallback
}

// GetDBPort returns the default database port for a service
func GetDBPort(serviceName string) string {
	if port, ok := DBPortMap[serviceName]; ok {
		return port
	}
	return "5432" // Default PostgreSQL port
}
