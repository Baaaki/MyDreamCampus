package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/db"
	"github.com/baaaki/mydreamcampus/course-catalog-service/internal/repository"
	"github.com/baaaki/mydreamcampus/shared/audit"
	"github.com/baaaki/mydreamcampus/shared/utils"
	"github.com/google/uuid"
)

// DirectAuditLogger writes audit log entries directly to the database.
// Used by the catalog service (which owns the audit_log table).
type DirectAuditLogger struct {
	repo        *repository.AuditRepository
	serviceName string
}

func NewDirectAuditLogger(repo *repository.AuditRepository, serviceName string) *DirectAuditLogger {
	return &DirectAuditLogger{
		repo:        repo,
		serviceName: serviceName,
	}
}

func (l *DirectAuditLogger) Log(ctx context.Context, event audit.AuditEvent) error {
	event.Service = l.serviceName

	var details []byte
	if event.Details != nil {
		var err error
		details, err = json.Marshal(event.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal audit details: %w", err)
		}
	}

	params := db.InsertAuditLogParams{
		Service:      event.Service,
		ActorRole:    event.ActorRole,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		Details:      details,
	}

	if event.ActorID != "" {
		parsed, err := uuid.Parse(event.ActorID)
		if err == nil {
			params.ActorID = utils.UUIDToPgtype(parsed)
		}
	}

	if event.ResourceID != "" {
		parsed, err := uuid.Parse(event.ResourceID)
		if err == nil {
			params.ResourceID = utils.UUIDToPgtype(parsed)
		}
	}

	_, err := l.repo.InsertAuditLog(ctx, params)
	return err
}
