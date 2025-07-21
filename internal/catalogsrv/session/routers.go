package session

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/apis"
	"github.com/tansive/tansive/internal/catalogsrv/auth"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/db"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tansive/tansive/internal/common/uuid"
)

var sessionHandlers = []policy.ResponseHandlerParam{
	{
		Method:  http.MethodGet,
		Path:    "/execution-state",
		Handler: getExecutionState,
	},
}

var sessionTangentHandlers = []policy.ResponseHandlerParam{
	{
		Method:  http.MethodPost,
		Path:    "/execution-state",
		Handler: createExecutionState,
	},
	{
		Method:  http.MethodPut,
		Path:    "/execution-state",
		Handler: updateExecutionState,
	},
	{
		Method:  http.MethodPost,
		Path:    "/stop",
		Handler: initializeStopSession,
	},
}

var sessionUserHandlers = []policy.ResponseHandlerParam{
	{
		Method:  http.MethodPost,
		Path:    "/",
		Handler: newSession,
	},
	{
		Method:  http.MethodGet,
		Path:    "/",
		Handler: getSessions,
	},
	{
		Method:  http.MethodGet,
		Path:    "/{sessionID}",
		Handler: getSessionSummaryByID,
	},
	{
		Method:  http.MethodDelete,
		Path:    "/{sessionID}",
		Handler: stopSession,
	},
	{
		Method:  http.MethodGet,
		Path:    "/{sessionID}/auditlog",
		Handler: getAuditLogByID,
	},
	{
		Method:  http.MethodGet,
		Path:    "/{sessionID}/auditlog/verification-key",
		Handler: getAuditLogVerificationKeyByID,
	},
}

func Router() chi.Router {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(tangentAuthMiddleware)
		r.Use(sessionContextMiddleware)
		for _, handler := range sessionTangentHandlers {
			r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(handler.Handler))
		}
	})
	r.Group(func(r chi.Router) {
		r.Use(auth.ContextMiddleware)
		for _, handler := range sessionHandlers {
			r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(handler.Handler))
		}
	})
	r.Group(func(r chi.Router) {
		r.Use(auth.ContextMiddleware)
		r.Use(apis.CatalogContextLoader)
		for _, handler := range sessionUserHandlers {
			r.Method(handler.Method, handler.Path, httpx.WrapHttpRsp(handler.Handler))
		}
	})
	return r
}

func tangentAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if err := validateTangentSignature(ctx, r); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("tangent signature validation failed")
			httpx.ErrUnAuthorized("tangent signature validation failed").Send(w)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func validateTangentSignature(ctx context.Context, r *http.Request) error {
	signature := r.Header.Get("X-Tangent-Signature")
	timestamp := r.Header.Get("X-Tangent-Signature-Timestamp")
	tangentIDStr := r.Header.Get("X-TangentID")

	if signature == "" || timestamp == "" || tangentIDStr == "" {
		return ErrInvalidRequest.Msg("missing signature headers")
	}

	tangentID, err := uuid.Parse(tangentIDStr)
	if err != nil {
		return ErrInvalidRequest.Msg("invalid tangent ID format")
	}

	tangent, err := db.DB(ctx).GetTangent(ctx, tangentID)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Str("tangent_id", tangentIDStr).Msg("failed to get tangent from database")
		return ErrInvalidRequest.Msg("tangent not found")
	}

	parsedTimestamp, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		return ErrInvalidRequest.Msg("invalid timestamp format")
	}

	now := time.Now().UTC()
	if now.Sub(parsedTimestamp) > 5*time.Minute || parsedTimestamp.Sub(now) > 5*time.Minute {
		return ErrInvalidRequest.Msg("signature timestamp too old or in the future")
	}

	var body []byte

	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		r.Body = io.NopCloser(strings.NewReader(string(body)))
	}

	requestPath := strings.TrimPrefix(r.URL.Path, "/")
	requestPath = "/" + requestPath
	stringToSign := strings.Join([]string{
		r.Method,
		requestPath,
		r.URL.RawQuery,
		string(body),
		timestamp,
	}, "\n")

	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return ErrInvalidRequest.Msg("invalid signature format")
	}

	if len(tangent.PublicKey) != ed25519.PublicKeySize {
		return ErrInvalidRequest.Msg("invalid public key size")
	}

	publicKey := ed25519.PublicKey(tangent.PublicKey)
	if !ed25519.Verify(publicKey, []byte(stringToSign), signatureBytes) {
		return ErrInvalidRequest.Msg("signature verification failed")
	}

	newCtx := catcommon.WithTenantID(r.Context(), catcommon.TenantId(tangent.TenantID))
	*r = *r.WithContext(newCtx)

	return nil
}

func sessionContextMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if catcommon.GetTestContext(ctx) {
			next.ServeHTTP(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			log.Ctx(ctx).Debug().Msg("missing authorization header")
			next.ServeHTTP(w, r)
			return
		}

		if !strings.HasPrefix(authHeader, auth.AuthHeaderPrefix) {
			log.Ctx(ctx).Debug().Msg("invalid authorization header format")
			httpx.ErrUnAuthorized(auth.GenericAuthError).Send(w)
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(authHeader, auth.AuthHeaderPrefix))
		if token == "" {
			log.Ctx(ctx).Debug().Msg("empty token")
			httpx.ErrUnAuthorized(auth.GenericAuthError).Send(w)
			return
		}

		var err error
		ctx, err = auth.ValidateToken(ctx, token)
		if err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("token validation failed")
			httpx.ErrUnAuthorized(auth.GenericAuthError).Send(w)
			return
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
