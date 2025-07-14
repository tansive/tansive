package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	srvsession "github.com/tansive/tansive/internal/catalogsrv/session"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/httpclient"
	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
)

// createSession handles HTTP requests to create new interactive sessions.
// Validates request body and returns a chunked response for session execution.
// Returns an error if request validation or session creation fails.
func createSession(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()

	if r.Body == nil {
		return nil, httpx.ErrInvalidRequest("request body is required")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, httpx.ErrUnableToReadRequest()
	}

	req := &tangentcommon.SessionCreateRequest{}
	if err := json.Unmarshal(body, req); err != nil {
		return nil, httpx.ErrInvalidRequest("failed to parse request body: " + err.Error())
	}

	var rsp *httpx.Response

	switch req.SessionType {
	case tangentcommon.SessionTypeInteractive:
		return processInteractiveSession(ctx, req)
	case tangentcommon.SessionTypeMCPProxy:
		return processMCPProxySession(ctx, req)
	}

	return rsp, nil
}

// This flow is temporary until we support a full Tangent-Server SSE connection
func stopSession(r *http.Request) (*httpx.Response, error) {
	ctx := r.Context()

	if r.Body == nil {
		return nil, httpx.ErrInvalidRequest("request body is required")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, httpx.ErrUnableToReadRequest()
	}

	// We are reusing this
	req := &tangentcommon.SessionCreateRequest{}
	if err := json.Unmarshal(body, req); err != nil {
		return nil, httpx.ErrInvalidRequest("failed to parse request body: " + err.Error())
	}

	var rsp *httpx.Response

	client := getHTTPClient(&clientConfig{
		serverURL: config.Config().TansiveServer.GetURL(),
	})

	opts := httpclient.RequestOptions{
		Method: http.MethodPost,
		Path:   "sessions/stop",
		QueryParams: map[string]string{
			"code":          req.Code,
			"code_verifier": req.CodeVerifier,
		},
	}

	body, _, err = client.DoRequest(opts)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to stop session")
		return nil, ErrFailedRequestToTansiveServer.Msg("unable to stop session: " + err.Error())
	}

	stopRsp := map[string]any{}
	if err := json.Unmarshal(body, &stopRsp); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to parse stop response")
		return nil, ErrFailedRequestToTansiveServer.Msg("unable to parse stop response: " + err.Error())
	}

	sessionIDStr := stopRsp["sessionID"].(string)
	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil || sessionID == uuid.Nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to parse session ID")
		return nil, ErrFailedRequestToTansiveServer.Msg("unable to parse session ID: " + err.Error())
	}

	if err := processStopSession(ctx, sessionID); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to stop session")
		return nil, ErrFailedRequestToTansiveServer.Msg("unable to stop session: " + err.Error())
	}

	rsp = &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   nil,
	}

	return rsp, nil
}

// processMCPProxySession creates an MCP proxy session from the request.
func processMCPProxySession(ctx context.Context, req *tangentcommon.SessionCreateRequest) (rsp *httpx.Response, apperr apperrors.Error) {
	session, err := resolveSession(ctx, req)
	if err != nil {
		return nil, err
	}
	url, err := runMCPProxySession(ctx, session)
	if err != nil {
		return nil, err
	}

	rsp = &httpx.Response{
		StatusCode: http.StatusCreated,
		Location:   url,
		Response:   nil,
	}
	return rsp, nil
}

// processInteractiveSession creates an interactive session from the request.
func processInteractiveSession(ctx context.Context, req *tangentcommon.SessionCreateRequest) (rsp *httpx.Response, apperr apperrors.Error) {
	session, err := resolveSession(ctx, req)
	if err != nil {
		return nil, err
	}
	rsp = &httpx.Response{
		StatusCode:  http.StatusOK,
		ContentType: "application/x-ndjson",
		Chunked:     true,
		WriteChunks: func(w http.ResponseWriter) error {
			ctx := log.Ctx(ctx).With().Str("session_id", session.id.String()).Logger().WithContext(ctx)
			return runInteractiveSession(ctx, w, session)
		},
	}
	return rsp, nil
}

// resolveSession creates an interactive session from the request.
// Retrieves execution state from the catalog server and creates an active session.
// Returns the created session and any error encountered during creation.
func resolveSession(ctx context.Context, req *tangentcommon.SessionCreateRequest) (*session, apperrors.Error) {
	client := getHTTPClient(&clientConfig{
		serverURL: config.Config().TansiveServer.GetURL(),
	})

	opts := httpclient.RequestOptions{
		Method: http.MethodPost,
		Path:   "sessions/execution-state",
		QueryParams: map[string]string{
			"code":          req.Code,
			"code_verifier": req.CodeVerifier,
		},
	}

	body, _, err := client.DoRequest(opts)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to create execution state")
		return nil, ErrFailedRequestToTansiveServer.Msg("unable to create execution state: " + err.Error())
	}

	rsp := &srvsession.SessionTokenRsp{}
	if err := json.Unmarshal(body, rsp); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to parse token response")
		return nil, ErrFailedRequestToTansiveServer.Msg("unable to parse token response: " + err.Error())
	}

	executionState, apperr := getExecutionState(ctx, rsp)
	if apperr != nil {
		return nil, apperr
	}

	ctx = log.Ctx(ctx).With().Str("session_id", executionState.SessionID.String()).Logger().WithContext(ctx)
	session, apperr := createActiveSession(ctx, executionState, rsp.Token, rsp.Expiry, req.SessionType)
	if apperr != nil {
		log.Ctx(ctx).Error().Err(apperr).Msg("unable to create active session")
		return nil, apperr
	}
	return session, nil
}

// getExecutionState retrieves execution state from the catalog server.
// Validates token and expiry, then fetches the complete execution state.
// Returns the execution state and any error encountered during retrieval.
func getExecutionState(ctx context.Context, rsp *srvsession.SessionTokenRsp) (*srvsession.ExecutionState, apperrors.Error) {
	if rsp.Token == "" {
		return nil, ErrTokenRequired
	}
	if rsp.Expiry.Before(time.Now()) {
		return nil, ErrTokenExpired
	}

	client := getHTTPClient(&clientConfig{
		token:       rsp.Token,
		tokenExpiry: rsp.Expiry,
		serverURL:   config.Config().TansiveServer.GetURL(),
	})

	opts := httpclient.RequestOptions{
		Method: http.MethodGet,
		Path:   "sessions/execution-state",
	}

	body, _, err := client.DoRequest(opts)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to get execution state")
		return nil, ErrFailedRequestToTansiveServer.Msg("unable to get execution state: " + err.Error())
	}

	executionState := &srvsession.ExecutionState{}
	if err := json.Unmarshal(body, executionState); err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to parse execution state")
		return nil, ErrFailedRequestToTansiveServer.Msg("unable to parse execution state: " + err.Error())
	}

	log.Ctx(ctx).Info().Str("session_id", executionState.SessionID.String()).Msg("obtained execution state")

	return executionState, nil
}

// createActiveSession creates an active session from execution state.
// Converts execution state to server context and creates a session in the session manager.
// Returns the created session and any error encountered during creation.
func createActiveSession(ctx context.Context, executionState *srvsession.ExecutionState, token string, tokenExpiry time.Time, sessionType tangentcommon.SessionType) (*session, apperrors.Error) {
	serverCtx := &ServerContext{
		SessionID:        executionState.SessionID,
		SkillSet:         executionState.SkillSet,
		Skill:            executionState.Skill,
		View:             executionState.View,
		ViewDefinition:   executionState.ViewDefinition,
		SessionVariables: executionState.SessionVariables,
		InputArgs:        executionState.InputArgs,
		Catalog:          executionState.Catalog,
		Variant:          executionState.Variant,
		Namespace:        executionState.Namespace,
		TenantID:         executionState.TenantID,
	}

	session, err := ActiveSessionManager().CreateSession(ctx, serverCtx, token, tokenExpiry, sessionType)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to create session")
		return nil, err
	}

	return session, nil
}

// runInteractiveSession executes a session and streams results to the HTTP response.
// Initializes audit logging, subscribes to event streams, and runs the session.
// Returns any error encountered during session execution.
func runInteractiveSession(ctx context.Context, w http.ResponseWriter, session *session) (apperr apperrors.Error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Ctx(ctx).Error().Msg("response writer does not support flushing")
		return ErrSessionError.Msg("response writer does not support flushing")
	}

	defer session.Finalize(ctx, apperr)

	auditLogCtx, cancelAuditLog := context.WithCancel(context.Background())
	defer cancelAuditLog()
	session.auditLogInfo.auditLogCancel = cancelAuditLog

	apperr = InitAuditLog(auditLogCtx, session)
	if apperr != nil {
		log.Ctx(ctx).Error().Err(apperr).Msg("unable to initialize audit log")
	}

	sessionLog, unsubSessionLog := GetEventBus().Subscribe(session.getTopic(TopicSessionLog), 100)
	defer unsubSessionLog()
	interactiveLog, unsubInteractiveLog := GetEventBus().Subscribe(session.getTopic(TopicInteractiveLog), 100)
	defer unsubInteractiveLog()

	logCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func(ctx context.Context) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-sessionLog:
				data, ok := event.Data.([]byte)
				if !ok {
					continue
				}
				w.Write(data)
				flusher.Flush()
			case event := <-interactiveLog:
				data, ok := event.Data.([]byte)
				if !ok {
					continue
				}
				w.Write(data)
				flusher.Flush()
			}
		}
	}(logCtx)

	// Run will block until the session is complete
	session.auditLogInfo.auditLogger.Info().
		Str("event", "session_start").
		Any("session_variables", session.context.SessionVariables).
		Msg("starting session")

	log.Ctx(ctx).Info().Str("skill", session.context.Skill).Msg("running session")
	runCtx := session.getLogger(TopicSessionLog).With().Str("skill", session.context.Skill).Str("actor", "system").Logger().WithContext(ctx)

	apperr = session.Run(runCtx, "", session.context.Skill, session.context.InputArgs)

	if apperr != nil {
		log.Ctx(ctx).Error().Err(apperr).Msg("session failed")
		session.auditLogInfo.auditLogger.Error().Str("event", "session_end").Err(apperr).Msg("session failed")
		return apperr
	}

	session.auditLogInfo.auditLogger.Info().Str("event", "session_end").Msg("session completed")

	// cancel should be called before wg.Wait() or we'll wait forever
	cancel()
	wg.Wait()

	log.Ctx(ctx).Info().Msg("session completed")

	return nil
}

func runMCPProxySession(ctx context.Context, session *session) (url string, apperr apperrors.Error) {
	auditLogCtx, cancelAuditLog := context.WithCancel(context.Background())
	session.auditLogInfo.auditLogCancel = cancelAuditLog
	defer func() {
		if apperr != nil {
			session.auditLogInfo.auditLogCancel()
		}
	}()

	apperr = InitAuditLog(auditLogCtx, session)
	if apperr != nil {
		log.Ctx(ctx).Error().Err(apperr).Msg("unable to initialize audit log")
	}

	// Run will block until the session is complete
	session.auditLogInfo.auditLogger.Info().
		Str("event", "session_start").
		Any("session_variables", session.context.SessionVariables).
		Msg("starting session")

	log.Ctx(ctx).Info().Str("skill", session.context.Skill).Msg("running session")

	url, apperr = session.RunMCPProxy(ctx, "", session.context.Skill, session.context.InputArgs)
	if apperr != nil {
		log.Ctx(ctx).Error().Err(apperr).Msg("session failed")
		session.auditLogInfo.auditLogger.Error().Str("event", "session_end").Err(apperr).Msg("session failed")
		return "", apperr
	}

	log.Ctx(ctx).Info().Msg("session started")
	return url, nil
}

func processStopSession(ctx context.Context, sessionID uuid.UUID) apperrors.Error {
	session, err := ActiveSessionManager().GetSession(sessionID)
	if err != nil {
		return err
	}
	session.Stop(ctx, nil)
	ActiveSessionManager().DeleteSession(sessionID)
	return nil
}
