package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/auth"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/db/models"
	"github.com/tansive/tansive/internal/catalogsrv/tangent"
	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tansive/tansive/internal/common/uuid"
)

func newSession(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()

	if r.Body == nil {
		return nil, httpx.ErrInvalidRequest("request body is required")
	}

	req, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, httpx.ErrUnableToReadRequest()
	}

	interactive := r.URL.Query().Get("interactive") == "true"
	// Until we support a full Tangent-Server SSE connection, we use the user to mediate
	temp_oauth := true
	codeChallenge := ""
	if interactive || temp_oauth {
		//We need to create a oauth2.0 session, so look for a code challenge
		codeChallenge = r.URL.Query().Get("code_challenge")
		if codeChallenge == "" {
			return nil, httpx.ErrInvalidRequest("code_challenge is required")
		}
	}

	session, tangent, err := NewSession(ctx, req, WithInteractive(interactive), WithCodeChallenge(codeChallenge))
	if err != nil {
		return nil, err
	}

	session.Save(ctx)

	if interactive || temp_oauth {
		log.Ctx(ctx).Info().Msgf("Creating auth code for session %s", session.ID().String())
		authCode, err := CreateAuthCode(ctx, session, codeChallenge)
		if err != nil {
			return nil, err
		}

		resp := &httpx.Response{
			StatusCode: http.StatusOK,
			Response: &InteractiveSessionRsp{
				Code:       authCode,
				TangentURL: tangent.URL,
			},
		}
		return resp, nil
	}

	return &httpx.Response{
		StatusCode: http.StatusCreated,
		Location:   "/sessions/" + session.ID().String(),
		Response:   nil,
	}, nil
}

func getExecutionState(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	sessionID := catcommon.GetSessionID(ctx)
	if sessionID == uuid.Nil {
		return nil, httpx.ErrInvalidRequest("invalid session ID")
	}
	session, err := GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	executionState := session.GetExecutionState(ctx)
	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   executionState,
	}, nil
}

func createExecutionState(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, httpx.ErrInvalidRequest("code is required")
	}
	codeVerifier := r.URL.Query().Get("code_verifier")
	if codeVerifier == "" {
		return nil, httpx.ErrInvalidRequest("code_verifier is required")
	}

	authCodeMetadata, err := GetAuthCode(ctx, code, codeVerifier)
	if err != nil {
		return nil, ErrNotAuthorized
	}

	ctx = setContextObjects(ctx, &authCodeMetadata)

	session, err := GetSession(ctx, authCodeMetadata.SessionID)
	if err != nil {
		return nil, err
	}

	token, expiry, err := createSessionToken(ctx, session)
	if err != nil {
		return nil, err
	}

	session.SetStatusSummary(ctx, SessionStatusRunning)

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response: &SessionTokenRsp{
			Token:  token,
			Expiry: expiry,
		},
	}, nil
}

var _ = createSessionToken

func createSessionToken(ctx context.Context, session SessionManager) (string, time.Time, error) {
	var view *models.View
	if vm, err := session.GetViewManager(ctx); err == nil {
		view, err = vm.GetViewModel()
		if err != nil {
			return "", time.Time{}, err
		}
		if view == nil {
			return "", time.Time{}, httpx.ErrInvalidRequest("view is nil")
		}
	} else {
		return "", time.Time{}, err
	}

	subjectType := catcommon.GetSubjectType(ctx)

	additionalClaims := map[string]any{
		"sub": "session/" + session.ID().String(),
	}

	if subjectType == catcommon.SubjectTypeUser {
		userID := catcommon.GetUserID(ctx)
		if userID == "" {
			return "", time.Time{}, httpx.ErrInvalidRequest("user ID is required")
		}
		additionalClaims["created_by"] = "user/" + userID
	}

	token, expiry, err := auth.CreateAccessToken(ctx, view, auth.WithAdditionalClaims(additionalClaims))
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiry, nil
}

func setContextObjects(ctx context.Context, m *AuthCodeMetadata) context.Context {
	ctx = catcommon.WithTenantID(ctx, m.TenantID)
	catalogContext := catcommon.GetCatalogContext(ctx)
	if catalogContext == nil {
		catalogContext = &catcommon.CatalogContext{}
	}
	catalogContext.CatalogID = m.CatalogID
	catalogContext.Catalog = m.ViewScope.Catalog
	if m.ViewScope.Variant != "" {
		catalogContext.Variant = m.ViewScope.Variant
	}
	if m.ViewScope.Namespace != "" {
		catalogContext.Namespace = m.ViewScope.Namespace
	}
	sessionContext := &catcommon.SessionContext{
		SessionID: m.SessionID,
	}
	catalogContext.SessionContext = sessionContext
	ctx = catcommon.WithCatalogContext(ctx, catalogContext)
	return ctx
}

func updateExecutionState(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	sessionID := catcommon.GetSessionID(ctx)
	if sessionID == uuid.Nil {
		return nil, ErrInvalidRequest.Msg("invalid session ID")
	}
	log.Ctx(ctx).Info().Msgf("updating execution state for session %s", sessionID.String())
	session, err := GetSession(ctx, sessionID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get session")
		return nil, ErrUnableToGetSession
	}
	var body []byte
	if r.Body != nil {
		var err error
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, ErrInvalidRequest.Msg("invalid request body")
		}
	} else {
		return nil, ErrInvalidRequest.Msg("request body is required")
	}

	var update ExecutionStatusUpdate
	if err := json.Unmarshal(body, &update); err != nil {
		return nil, ErrInvalidRequest.Msg("invalid request body")
	}

	if update.Status.AuditLog != "" {
		logFilePath, err := WriteAuditLogFile(ctx, session.ID(), update.Status.AuditLog)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to write audit log")
			// continue anyway
		}
		log.Ctx(ctx).Info().Msgf("wrote audit log to %s", logFilePath)
		update.Status.AuditLog = logFilePath // replace the audit log with the file path
	}

	if !IsValidSessionStatus(update.StatusSummary) {
		return nil, ErrInvalidRequest.Msg("invalid status summary")
	}

	session.SetStatus(ctx, update.StatusSummary, update.Status)
	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   &ExecutionStatusUpdate{},
	}, nil
}

func getSessions(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	sessionList, err := db.DB(ctx).ListSessionsByCatalog(ctx, catcommon.GetCatalogID(ctx))
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get session")
		return nil, ErrUnableToGetSession
	}

	sessionListInfo := make([]SessionSummaryInfo, len(sessionList))
	for i, session := range sessionList {
		var status ExecutionStatus
		if err := json.Unmarshal([]byte(session.Status), &status); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal status")
			status = ExecutionStatus{}
		}
		sessionListInfo[i] = SessionSummaryInfo{
			SessionID:     session.SessionID,
			UserID:        session.UserID,
			CreatedAt:     session.CreatedAt,
			StartedAt:     session.StartedAt,
			UpdatedAt:     session.UpdatedAt,
			StatusSummary: SessionStatus(session.StatusSummary),
			Error:         status.Error,
		}
	}

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   sessionListInfo,
	}, nil
}

func getSessionSummaryByID(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()

	sessionID := r.URL.Query().Get("sessionID")
	if sessionID == "" {
		return nil, httpx.ErrInvalidRequest("sessionID is required")
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, httpx.ErrInvalidRequest("invalid sessionID")
	}

	session, err := db.DB(ctx).GetSession(ctx, sessionUUID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get session")
		return nil, ErrUnableToGetSession
	}

	var status ExecutionStatus
	if err := json.Unmarshal([]byte(session.Status), &status); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal status")
		status = ExecutionStatus{}
	}

	sessionSummaryInfo := SessionSummaryInfo{
		SessionID:     session.SessionID,
		UserID:        session.UserID,
		CreatedAt:     session.CreatedAt,
		StartedAt:     session.StartedAt,
		UpdatedAt:     session.UpdatedAt,
		StatusSummary: SessionStatus(session.StatusSummary),
		Error:         status.Error,
	}
	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   sessionSummaryInfo,
	}, nil
}

// This flow is temporary until we support a full Tangent-Server SSE connection
func initializeStopSession(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, httpx.ErrInvalidRequest("code is required")
	}
	codeVerifier := r.URL.Query().Get("code_verifier")
	if codeVerifier == "" {
		return nil, httpx.ErrInvalidRequest("code_verifier is required")
	}

	authCodeMetadata, err := GetAuthCode(ctx, code, codeVerifier)
	if err != nil {
		return nil, ErrNotAuthorized
	}

	type StopSessionRsp struct {
		SessionID uuid.UUID `json:"sessionID"`
	}

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response: &StopSessionRsp{
			SessionID: authCodeMetadata.SessionID,
		},
	}, nil

}

// This flow is temporary until we support a full Tangent-Server SSE connection
func stopSession(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()

	sessionID := chi.URLParam(r, "sessionID")
	if sessionID == "" {
		return nil, httpx.ErrInvalidRequest("sessionID is required")
	}

	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, httpx.ErrInvalidRequest("invalid sessionID")
	}

	codeChallenge := r.URL.Query().Get("code_challenge")
	if codeChallenge == "" {
		return nil, httpx.ErrInvalidRequest("code_challenge is required")
	}

	session, err := GetSession(ctx, sessionUUID)
	if err != nil {
		return nil, err
	}

	userID := catcommon.GetUserID(ctx)
	if session.UserID() != userID {
		return nil, httpx.ErrInvalidRequest("user not authorized to stop session")
	}

	tangentID := session.TangentID()
	if tangentID == uuid.Nil {
		return nil, httpx.ErrInvalidRequest("no tangent found for session")
	}

	// get the tangent
	t, err := tangent.GetTangentByID(ctx, tangentID)
	if err != nil {
		return nil, err
	}

	log.Ctx(ctx).Info().Msgf("Creating auth code for session %s", session.ID().String())
	authCode, err := CreateAuthCode(ctx, session, codeChallenge)
	if err != nil {
		return nil, err
	}

	resp := &httpx.Response{
		StatusCode: http.StatusOK,
		Response: &InteractiveSessionRsp{
			Code:       authCode,
			TangentURL: t.URL,
		},
	}
	return resp, nil
}

func getAuditLogByID(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")
	if sessionID == "" {
		return nil, httpx.ErrInvalidRequest("sessionID is required")
	}
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, httpx.ErrInvalidRequest("invalid sessionID")
	}

	auditLog, err := EncodeAuditLogFile(ctx, sessionUUID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to encode audit log")
		return nil, err
	}

	return &httpx.Response{
		StatusCode:  http.StatusOK,
		ContentType: "text/plain",
		Response:    auditLog,
	}, nil
}

func getAuditLogVerificationKeyByID(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()
	sessionID := chi.URLParam(r, "sessionID")
	if sessionID == "" {
		return nil, httpx.ErrInvalidRequest("sessionID is required")
	}
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, httpx.ErrInvalidRequest("invalid sessionID")
	}

	session, err := db.DB(ctx).GetSession(ctx, sessionUUID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to get session")
		return nil, ErrUnableToGetSession
	}

	var status ExecutionStatus
	if err := json.Unmarshal([]byte(session.Status), &status); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to unmarshal status")
		return nil, ErrInvalidRequest.Msg("invalid status")
	}

	if status.AuditLog == "" {
		return nil, ErrInvalidRequest.Msg("audit log is not available")
	}

	if len(status.AuditLogVerificationKey) == 0 {
		return nil, ErrInvalidRequest.Msg("audit log verification key is not available")
	}

	auditLogVerificationKey := AuditLogVerificationKey{
		Key: status.AuditLogVerificationKey,
	}

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   auditLogVerificationKey,
	}, nil
}
