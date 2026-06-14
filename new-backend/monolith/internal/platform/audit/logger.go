package audit

import "context"

// AuditEvent represents an immutable audit log entry.
type AuditEvent struct {
	Service      string         `json:"service"`
	ActorID      string         `json:"actor_id"`
	ActorRole    string         `json:"actor_role"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id,omitempty"`
	Details      map[string]any `json:"details,omitempty"`
}

// Logger writes audit log entries.
// Catalog service implements this via direct DB write.
// Other services implement this via HTTP call to catalog's internal endpoint.
type Logger interface {
	Log(ctx context.Context, event AuditEvent) error
}
