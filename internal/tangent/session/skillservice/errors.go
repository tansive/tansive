package skillservice

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

var (
	// ErrInvalidRequest is returned when the request is malformed or invalid.
	// Occurs when request body cannot be parsed or required fields are missing.
	ErrInvalidRequest apperrors.Error = apperrors.New("invalid request").SetStatusCode(http.StatusBadRequest)

	// ErrSkillServiceError is returned when skill service operations fail.
	// Occurs when skill execution, retrieval, or context operations encounter errors.
	ErrSkillServiceError apperrors.Error = apperrors.New("skill service error").SetStatusCode(http.StatusInternalServerError)
)
