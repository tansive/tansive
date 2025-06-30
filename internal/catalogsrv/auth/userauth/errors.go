package userauth

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

// Base idtoken error
var (
	ErrIDToken apperrors.Error = apperrors.New("identity token error").SetStatusCode(http.StatusInternalServerError)
)

// Token errors
var (
	ErrTokenGeneration            apperrors.Error = ErrIDToken.New("failed to generate token").SetStatusCode(http.StatusInternalServerError)
	ErrUnableToParseTokenDuration apperrors.Error = ErrIDToken.New("unable to parse token duration").SetStatusCode(http.StatusInternalServerError)
	ErrUnableToParseToken         apperrors.Error = ErrIDToken.New("unable to parse token").SetStatusCode(http.StatusUnauthorized)
	ErrInvalidToken               apperrors.Error = ErrIDToken.New("invalid token").SetStatusCode(http.StatusUnauthorized)
)

// Misc errors
var (
	ErrLoginNotSupported apperrors.Error = ErrIDToken.New("login is not supported").SetStatusCode(http.StatusBadRequest)
)
