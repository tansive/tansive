package session

import (
	"context"
	"encoding/json"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
)

type SessionManager interface {
	ID() uuid.UUID
	Save(ctx context.Context) apperrors.Error
	GetViewManager(ctx context.Context) (policy.ViewManager, apperrors.Error)
	GetExecutionState(ctx context.Context) *ExecutionState
	SetStatusSummary(ctx context.Context, statusSummary SessionStatus) apperrors.Error
	GetStatusSummaryInfo(ctx context.Context) *SessionSummaryInfo
	SetStatus(ctx context.Context, statusSummary SessionStatus, status ExecutionStatus) apperrors.Error
}

func (s *sessionManager) ID() uuid.UUID {
	return s.session.SessionID
}

func (s *sessionManager) GetViewManager(ctx context.Context) (policy.ViewManager, apperrors.Error) {
	return s.viewManager, nil
}

func (s *sessionManager) GetExecutionState(ctx context.Context) *ExecutionState {
	sessionInfo := SessionInfo{}
	err := json.Unmarshal(s.session.Info, &sessionInfo)
	if err != nil {
		return nil
	}
	return &ExecutionState{
		SessionID:        s.session.SessionID,
		SkillSet:         s.session.SkillSet,
		Skill:            s.session.Skill,
		View:             s.viewManager.Name(),
		ViewDefinition:   s.viewManager.GetViewDefinition(),
		SessionVariables: sessionInfo.SessionVariables,
		InputArgs:        sessionInfo.InputArgs,
		Catalog:          s.viewManager.Scope().Catalog,
		Variant:          s.viewManager.Scope().Variant,
		Namespace:        s.viewManager.Scope().Namespace,
		TenantID:         catcommon.GetTenantID(ctx),
	}
}

func (s *sessionManager) SetStatusSummary(ctx context.Context, statusSummary SessionStatus) apperrors.Error {
	s.session.StatusSummary = string(statusSummary)
	err := db.DB(ctx).UpdateSessionStatus(ctx, s.session.SessionID, string(statusSummary), s.session.Status)
	if err != nil {
		return err
	}
	return nil
}

func (s *sessionManager) SetStatus(ctx context.Context, statusSummary SessionStatus, status ExecutionStatus) apperrors.Error {
	statusJSON, err := json.Marshal(status)
	if err != nil {
		return ErrInvalidObject.Msg("failed to marshal status: " + err.Error())
	}
	err = db.DB(ctx).UpdateSessionStatus(ctx, s.session.SessionID, string(statusSummary), statusJSON)
	if err != nil {
		return ErrInvalidObject.Msg("failed to update session status: " + err.Error())
	}
	return nil
}

func (s *sessionManager) GetStatusSummaryInfo(ctx context.Context) *SessionSummaryInfo {
	var status ExecutionStatus
	if err := json.Unmarshal(s.session.Status, &status); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal status")
		status = ExecutionStatus{}
	}
	return &SessionSummaryInfo{
		SessionID:     s.session.SessionID,
		UserID:        s.session.UserID,
		CreatedAt:     s.session.CreatedAt,
		StartedAt:     s.session.StartedAt,
		UpdatedAt:     s.session.UpdatedAt,
		StatusSummary: SessionStatus(s.session.StatusSummary),
		Error:         status.Error,
	}
}
