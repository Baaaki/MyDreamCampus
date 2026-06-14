package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServicePortMap_AllExpectedServicesPresent(t *testing.T) {
	expected := map[string]string{
		"auth":         "8001",
		"staff":        "8002",
		"student":      "8003",
		"catalog":      "8004",
		"enrollment":   "8005",
		"attendance":   "8006",
		"grades":       "8007",
		"meal":         "8008",
		"notification": "8009",
	}
	for svc, port := range expected {
		assert.Equal(t, port, ServicePortMap[svc], "service %s port mismatch", svc)
		assert.Equal(t, port, GetServicePort(svc))
	}
}

func TestDBPortMap_NoCollisions(t *testing.T) {
	seen := map[string]string{}
	for svc, port := range DBPortMap {
		if other, exists := seen[port]; exists {
			t.Fatalf("port %s reused: %s and %s collide", port, other, svc)
		}
		seen[port] = svc
		assert.Equal(t, port, GetDBPort(svc))
	}
}

func TestServicePortMap_NoHTTPPortCollisions(t *testing.T) {
	seen := map[string]string{}
	for svc, port := range ServicePortMap {
		if other, exists := seen[port]; exists {
			t.Fatalf("port %s reused: %s and %s collide", port, other, svc)
		}
		seen[port] = svc
	}
}

func TestGetServicePort_FallbackForUnknown(t *testing.T) {
	assert.Equal(t, "8000", GetServicePort("nonexistent-service"))
}

func TestGetDBPort_FallbackForUnknown(t *testing.T) {
	assert.Equal(t, "5432", GetDBPort("nonexistent-service"))
}

func TestPortConstants_MatchMap(t *testing.T) {
	// Sentinel: ensure constants and map agree
	cases := map[string]string{
		AuthServicePort:       ServicePortMap["auth"],
		StaffServicePort:      ServicePortMap["staff"],
		StudentServicePort:    ServicePortMap["student"],
		CatalogServicePort:    ServicePortMap["catalog"],
		EnrollmentServicePort: ServicePortMap["enrollment"],
		AttendanceServicePort: ServicePortMap["attendance"],
		GradesServicePort:     ServicePortMap["grades"],
		MealServicePort:       ServicePortMap["meal"],
	}
	for constant, mapped := range cases {
		assert.Equal(t, constant, mapped)
	}
}
