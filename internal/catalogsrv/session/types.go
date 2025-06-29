package session

import (
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/uuid"
)

type SessionStatus string

const (
	SessionStatusCreated    SessionStatus = "created"
	SessionStatusRunning    SessionStatus = "running"
	SessionStatusCompleted  SessionStatus = "completed"
	SessionStatusFailed     SessionStatus = "failed"
	SessionStatusExpired    SessionStatus = "expired"
	SessionStatusCancelled  SessionStatus = "cancelled"
	SessionStatusPaused     SessionStatus = "paused"
	SessionStatusResumed    SessionStatus = "resumed"
	SessionStatusSuspended  SessionStatus = "suspended"
	SessionStatusTerminated SessionStatus = "terminated"
)

var validSessionStatus = map[SessionStatus]struct{}{
	SessionStatusCreated:    {},
	SessionStatusRunning:    {},
	SessionStatusCompleted:  {},
	SessionStatusFailed:     {},
	SessionStatusExpired:    {},
	SessionStatusCancelled:  {},
	SessionStatusPaused:     {},
	SessionStatusResumed:    {},
	SessionStatusSuspended:  {},
	SessionStatusTerminated: {},
}

func IsValidSessionStatus(status SessionStatus) bool {
	_, ok := validSessionStatus[status]
	return ok
}

type InteractiveSessionRsp struct {
	Code       string `json:"code"`
	TangentURL string `json:"tangentURL"`
}

type SessionTokenRsp struct {
	Token  string    `json:"token"`
	Expiry time.Time `json:"expiry"`
}

type ExecutionState struct {
	SessionID        uuid.UUID              `json:"sessionID"`
	SkillSet         string                 `json:"skillSet"`
	Skill            string                 `json:"skill"`
	View             string                 `json:"view"`
	ViewDefinition   *policy.ViewDefinition `json:"viewDefinition"`
	SessionVariables map[string]any         `json:"sessionVariables"`
	InputArgs        map[string]any         `json:"inputArgs"`
	Catalog          string                 `json:"catalog"`
	Variant          string                 `json:"variant"`
	Namespace        string                 `json:"namespace"`
	TenantID         catcommon.TenantId     `json:"tenantID"`
}

type ExecutionStatus struct {
	AuditLog                string         `json:"auditLog"`
	AuditLogVerificationKey []byte         `json:"auditLogVerificationKey"`
	Error                   map[string]any `json:"error"`
}

type ExecutionStatusUpdate struct {
	StatusSummary SessionStatus   `json:"statusSummary"`
	Status        ExecutionStatus `json:"status"`
}

type SessionSummaryInfo struct {
	SessionID     uuid.UUID      `json:"sessionID"`
	UserID        string         `json:"userID"`
	CreatedAt     time.Time      `json:"createdAt"`
	StartedAt     time.Time      `json:"startedAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	StatusSummary SessionStatus  `json:"statusSummary"`
	Error         map[string]any `json:"error"`
}

type AuditLogVerificationKey struct {
	Key []byte `json:"key"`
}

type SessionList struct {
	SessionSummaryInfo []SessionSummaryInfo `json:"sessionSummaryInfo"`
}
