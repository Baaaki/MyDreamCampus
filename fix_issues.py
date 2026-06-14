import os
import re

def fix_main_go():
    path = "/home/nautilus/Desktop/Playground/mydreamcampus/new-backend/monolith/cmd/main.go"
    with open(path, "r") as f:
        content = f.read()

    # Add utils import
    if '"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"' not in content:
        content = content.replace(
            '"go.uber.org/zap"\n)',
            '"go.uber.org/zap"\n\t"github.com/baaaki/mydreamcampus/monolith/internal/platform/utils"\n)'
        )

    # Replace os.Setenv
    old_code = """\t// Export JWT_SECRET to environment for backward compatibility with utils.GetJWTSecret()
	os.Setenv("JWT_SECRET", cfg.JWT.Secret)"""
    new_code = """\t// Initialize JWT secret globally to avoid os.Setenv anti-pattern
	utils.InitJWTSecret(cfg.JWT.Secret)"""
    content = content.replace(old_code, new_code)
    
    with open(path, "w") as f:
        f.write(content)

def fix_jwt_go():
    path = "/home/nautilus/Desktop/Playground/mydreamcampus/new-backend/monolith/internal/platform/utils/jwt.go"
    with open(path, "r") as f:
        content = f.read()

    # Add var jwtSecret []byte
    if "jwtSecret" not in content:
        content = content.replace(
            'ErrExpiredToken = errors.New("token has expired")\n)',
            'ErrExpiredToken = errors.New("token has expired")\n\tjwtSecret       []byte\n)'
        )
        content = content.replace(
            "// TokenType represents the type of JWT token",
            "// InitJWTSecret sets the global JWT secret.\nfunc InitJWTSecret(secret string) {\n\tjwtSecret = []byte(secret)\n}\n\n// TokenType represents the type of JWT token"
        )
        
        old_get = """func GetJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is not set")
	}
	return []byte(secret)
}"""
        new_get = """func GetJWTSecret() []byte {
	if len(jwtSecret) > 0 {
		return jwtSecret
	}
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		panic("JWT_SECRET environment variable is not set and InitJWTSecret was not called")
	}
	return []byte(secret)
}"""
        content = content.replace(old_get, new_get)

    with open(path, "w") as f:
        f.write(content)

def fix_enrollment_service():
    path = "/home/nautilus/Desktop/Playground/mydreamcampus/new-backend/monolith/internal/modules/enrollment/service/enrollment_service.go"
    with open(path, "r") as f:
        content = f.read()

    old_schedules = """		// Convert to DTO format via JSON
		var scheduleSessions []dto.ScheduleSession
		if len(course.ScheduleSessions) > 0 {
			b, _ := json.Marshal(course.ScheduleSessions)
			_ = json.Unmarshal(b, &scheduleSessions)
		}"""
    new_schedules = """		// Map course catalog sessions to enrollment DTO
		var scheduleSessions []dto.ScheduleSession
		for _, s := range course.ScheduleSessions {
			var intSlots []int
			for _, sl := range s.SlotNumbers {
				intSlots = append(intSlots, int(sl))
			}
			scheduleSessions = append(scheduleSessions, dto.ScheduleSession{
				DayOfWeek:   s.DayOfWeek,
				SlotNumbers: intSlots,
				SessionType: s.SessionType,
			})
		}"""
    content = content.replace(old_schedules, new_schedules)

    old_prereq = """	// Parse prerequisites from JSONB - actually it's now a struct array in DTO
	var prerequisites []dto.PrerequisiteCourse
	if len(course.Prerequisites) > 0 {
		b, _ := json.Marshal(course.Prerequisites)
		_ = json.Unmarshal(b, &prerequisites)
	}"""
    new_prereq = """	// Map prerequisites
	var prerequisites []dto.PrerequisiteCourse
	for _, p := range course.Prerequisites {
		prerequisites = append(prerequisites, dto.PrerequisiteCourse{
			ID:         p.ID,
			CourseCode: p.CourseCode,
			CourseName: p.CourseName,
		})
	}"""
    content = content.replace(old_prereq, new_prereq)

    with open(path, "w") as f:
        f.write(content)

def fix_enrollment_program():
    path = "/home/nautilus/Desktop/Playground/mydreamcampus/new-backend/monolith/internal/modules/enrollment/service/enrollment_service_program.go"
    with open(path, "r") as f:
        content = f.read()

    if "course_catalog/dto" not in content:
        content = content.replace(
            '"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/dto"',
            '"github.com/baaaki/mydreamcampus/monolith/internal/modules/enrollment/dto"\n\tcatalogDTO "github.com/baaaki/mydreamcampus/monolith/internal/modules/course_catalog/dto"'
        )

    old_json = """			if val, ok := catalogMap[cID]; ok {
				// Convert catalog ScheduleSession to enrollment ScheduleSession
				// catalog uses course_catalog/dto.ScheduleSession
				// enrollment uses enrollment/dto.ScheduleSession
				// we use reflection or just parse via JSON for simplicity, 
				// or better yet, since both are defined locally, we map them manually.
				// Since we don't import course_catalog/dto here, we assume it has the same structure.
				// Let's use JSON marshal/unmarshal for safe conversion between identical structs in different packages.
				b, _ := json.Marshal(val)
				_ = json.Unmarshal(b, &scheduleSessions)
			}"""
    
    new_json = """			if val, ok := catalogMap[cID]; ok {
				if catalogSessions, ok := val.([]catalogDTO.ScheduleSession); ok {
					for _, s := range catalogSessions {
						var intSlots []int
						for _, sl := range s.SlotNumbers {
							intSlots = append(intSlots, int(sl))
						}
						scheduleSessions = append(scheduleSessions, dto.ScheduleSession{
							DayOfWeek:   s.DayOfWeek,
							SlotNumbers: intSlots,
							SessionType: s.SessionType,
						})
					}
				}
			}"""
    content = content.replace(old_json, new_json)

    with open(path, "w") as f:
        f.write(content)

def fix_docker_compose():
    path = "/home/nautilus/Desktop/Playground/mydreamcampus/new-backend/infrastructure/docker-compose.yml"
    with open(path, "r") as f:
        content = f.read()

    content = content.replace("POSTGRES_USER: postgres", "POSTGRES_USER: ${POSTGRES_USER:-postgres}")
    content = content.replace("POSTGRES_PASSWORD: postgres", "POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}")
    
    with open(path, "w") as f:
        f.write(content)

fix_main_go()
fix_jwt_go()
fix_enrollment_service()
fix_enrollment_program()
fix_docker_compose()
print("Python fixes complete")
