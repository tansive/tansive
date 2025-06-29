package jsruntime

import (
	"net/http"

	"github.com/tansive/tansive/internal/common/apperrors"
)

var (
	ErrJSRuntime         = apperrors.New("jsruntime error")
	ErrJSRuntimeTimeout  = ErrJSRuntime.New("jsruntime timeout")
	ErrInvalidJSFunction = ErrJSRuntime.New("invalid javascript function")
	ErrJSRuntimeError    = ErrJSRuntime.New("jsruntime error").SetStatusCode(http.StatusBadRequest).SetExpandError(true)
	ErrJSExecutionError  = ErrJSRuntime.New("js execution error").SetStatusCode(http.StatusUnprocessableEntity).SetExpandError(true)
)
