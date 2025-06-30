package session

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	"github.com/tansive/tansive/internal/catalogsrv/policy"
	"github.com/tansive/tansive/internal/common/uuid"
)

// This is a PKCE flow implementation in-memory for single instance deployment. This will need to move in to a
// distributed cache for multi-instance deployment.

type AuthCodeMetadata struct {
	SessionID     uuid.UUID
	ViewScope     policy.Scope
	Code          string
	CodeChallenge string
	CatalogID     uuid.UUID
	TenantID      catcommon.TenantId
	ExpiresAt     time.Time
}

var (
	authCodes = make(map[string]AuthCodeMetadata)
	mu        sync.RWMutex
)

func generateRandomCode(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func CreateAuthCode(ctx context.Context, session SessionManager, codeChallenge string) (string, error) {
	code, err := generateRandomCode(32) // 32 bytes -> 43-char base64 string
	if err != nil {
		return "", err
	}

	viewManager, err := session.GetViewManager(ctx)
	if err != nil {
		return "", err
	}
	viewScope := viewManager.Scope()

	mu.Lock()
	authCodes[code] = AuthCodeMetadata{
		SessionID:     session.ID(),
		ViewScope:     viewScope,
		Code:          code,
		CodeChallenge: codeChallenge,
		CatalogID:     viewManager.CatalogID(),
		TenantID:      catcommon.GetTenantID(ctx),
		ExpiresAt:     time.Now().Add(10 * time.Minute),
	}
	mu.Unlock()
	return code, nil
}

func GetAuthCode(ctx context.Context, code, codeVerifier string) (AuthCodeMetadata, error) {
	mu.Lock()
	defer mu.Unlock()

	authCode, ok := authCodes[code]
	if !ok {
		return AuthCodeMetadata{}, errors.New("invalid auth code")
	}
	defer delete(authCodes, code)

	if time.Now().After(authCode.ExpiresAt) {
		return AuthCodeMetadata{}, errors.New("auth code expired")
	}

	hashed := sha256.Sum256([]byte(codeVerifier))
	expectedChallenge := base64.RawURLEncoding.EncodeToString(hashed[:])

	if authCode.CodeChallenge != expectedChallenge {
		return AuthCodeMetadata{}, errors.New("invalid code verifier")
	}

	return authCode, nil
}
