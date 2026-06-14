package audit

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// SecurityEventType represents the type of security audit event
type SecurityEventType string

const (
	EventLogin           SecurityEventType = "LOGIN"
	EventLoginFailed     SecurityEventType = "LOGIN_FAILED"
	EventLogout          SecurityEventType = "LOGOUT"
	EventLogoutAll       SecurityEventType = "LOGOUT_ALL"
	EventPasswordChange  SecurityEventType = "PASSWORD_CHANGE"
	EventTokenRefresh    SecurityEventType = "TOKEN_REFRESH"
	EventAccountLocked   SecurityEventType = "ACCOUNT_LOCKED"
	EventAccountUnlocked SecurityEventType = "ACCOUNT_UNLOCKED"
	EventRoleChange      SecurityEventType = "ROLE_CHANGE"
	EventUserCreated     SecurityEventType = "USER_CREATED"
	EventUserDeactivated SecurityEventType = "USER_DEACTIVATED"
	EventAccessDenied    SecurityEventType = "ACCESS_DENIED"
	EventCSRFViolation   SecurityEventType = "CSRF_VIOLATION"
	EventBulkImport      SecurityEventType = "BULK_IMPORT"
)

// SecurityEvent represents a security audit log entry
type SecurityEvent struct {
	Timestamp time.Time         `json:"timestamp"`
	EventType SecurityEventType `json:"event_type"`
	UserID    string            `json:"user_id,omitempty"`
	Email     string            `json:"email,omitempty"`
	IP        string            `json:"ip"`
	UserAgent string            `json:"user_agent"`
	Resource  string            `json:"resource,omitempty"`
	Action    string            `json:"action,omitempty"`
	Result    string            `json:"result"` // "success" or "failure"
	Reason    string            `json:"reason,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

var securityLogger *zap.Logger

// InitSecurity initializes the security audit logger with structured JSON output
func InitSecurity(environment string) {
	var config zap.Config
	if environment == "production" {
		config = zap.NewProductionConfig()
	} else {
		config = zap.NewDevelopmentConfig()
	}

	// Security audit logs always use JSON for structured parsing
	config.Encoding = "json"
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.MessageKey = "event"

	var err error
	securityLogger, err = config.Build(zap.AddCallerSkip(1))
	if err != nil {
		panic("failed to initialize security audit logger: " + err.Error())
	}
}

// LogSecurity records a security audit event
func LogSecurity(event SecurityEvent) {
	if securityLogger == nil {
		return
	}

	fields := []zap.Field{
		zap.String("event_type", string(event.EventType)),
		zap.String("result", event.Result),
		zap.String("ip", event.IP),
		zap.String("user_agent", event.UserAgent),
	}

	if event.UserID != "" {
		fields = append(fields, zap.String("user_id", event.UserID))
	}
	if event.Email != "" {
		fields = append(fields, zap.String("email", event.Email))
	}
	if event.Resource != "" {
		fields = append(fields, zap.String("resource", event.Resource))
	}
	if event.Action != "" {
		fields = append(fields, zap.String("action", event.Action))
	}
	if event.Reason != "" {
		fields = append(fields, zap.String("reason", event.Reason))
	}
	for k, v := range event.Metadata {
		fields = append(fields, zap.String("meta_"+k, v))
	}

	securityLogger.Info("SECURITY_AUDIT", fields...)
}

// LogSecurityFromContext creates a security audit event with IP and UserAgent extracted from gin context
func LogSecurityFromContext(c *gin.Context, eventType SecurityEventType, result string, userID string) {
	LogSecurity(SecurityEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		UserID:    userID,
		IP:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Result:    result,
	})
}

// LogSecurityFromContextWithDetails creates a detailed security audit event
func LogSecurityFromContextWithDetails(c *gin.Context, eventType SecurityEventType, result, userID, reason string, metadata map[string]string) {
	LogSecurity(SecurityEvent{
		Timestamp: time.Now(),
		EventType: eventType,
		UserID:    userID,
		IP:        c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
		Result:    result,
		Reason:    reason,
		Metadata:  metadata,
	})
}

// SyncSecurity flushes any buffered security audit log entries
func SyncSecurity() {
	if securityLogger != nil {
		_ = securityLogger.Sync()
	}
}
