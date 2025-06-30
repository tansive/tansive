package apperrors

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	t.Run("TestError", func(t *testing.T) {
		ErrBaseErr := New("base error")
		assert.Equal(t, "base error", ErrBaseErr.Error())
		assert.Equal(t, "msg", ErrBaseErr.New("msg").Error())
		assert.ErrorIs(t, ErrBaseErr, ErrBaseErr)

		ErrFirstLevel := ErrBaseErr.New("first level")
		assert.Equal(t, "first level", ErrFirstLevel.Error())
		assert.ErrorIs(t, ErrFirstLevel, ErrBaseErr)

		ErrAnotherErr := New("another error")
		ErrAnotherErrMsg := ErrAnotherErr.Msg("another error msg")
		ErrYetAnotherErr := New("yet another error")
		ErrYetAnotherErrMsg := ErrYetAnotherErr.Msg("yet another error msg")
		ErrWrappedErr := ErrFirstLevel.Err(ErrAnotherErrMsg, ErrYetAnotherErrMsg)
		assert.Equal(t, "first level", ErrWrappedErr.Error())
		assert.ErrorIs(t, ErrWrappedErr, ErrBaseErr)
		assert.ErrorIs(t, ErrWrappedErr, ErrFirstLevel)
		assert.ErrorIs(t, ErrWrappedErr, ErrAnotherErr)
		assert.ErrorIs(t, ErrWrappedErr, ErrAnotherErrMsg)
		assert.ErrorIs(t, ErrWrappedErr, ErrYetAnotherErr)
		assert.ErrorIs(t, ErrWrappedErr, ErrYetAnotherErrMsg)

		err := errors.New("error")
		ErrWrappedErr = ErrFirstLevel.Err(err)
		assert.Equal(t, "first level", ErrWrappedErr.Error())
		assert.ErrorIs(t, ErrWrappedErr, ErrBaseErr)
		assert.ErrorIs(t, ErrWrappedErr, err)

		ErrWrappedErr = ErrFirstLevel.MsgErr("msg", err)
		assert.Equal(t, "msg", ErrWrappedErr.Error())
		assert.ErrorIs(t, ErrWrappedErr, ErrBaseErr)
		assert.ErrorIs(t, ErrWrappedErr, err)

		ErrAnotherGoErr := fmt.Errorf("another error")
		ErrYetAnotherGoErr := fmt.Errorf("yet another error")
		ErrWrappedGoErr := ErrFirstLevel.Err(ErrAnotherGoErr, ErrYetAnotherGoErr)
		assert.Equal(t, "first level", ErrWrappedGoErr.Error())
		assert.ErrorIs(t, ErrWrappedGoErr, ErrBaseErr)
		assert.ErrorIs(t, ErrWrappedGoErr, ErrAnotherGoErr)
		assert.ErrorIs(t, ErrWrappedGoErr, ErrYetAnotherGoErr)

		ErrSchemaValidation := New("error validating schema").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
		ErrInvalidSchema := ErrSchemaValidation.New("invalid schema").SetExpandError(true).SetStatusCode(http.StatusBadRequest)
		validationErrors := ValidationErrors{
			ValidationError{
				Field:  "name",
				Value:  "invalid",
				ErrStr: "invalid name",
			},
			ValidationError{
				Field:  "description",
				Value:  "invalid",
				ErrStr: "invalid description",
			},
		}
		ErrWrappedValidationErr := ErrInvalidSchema.Err(validationErrors)
		assert.True(t, errors.Is(ErrWrappedValidationErr, ErrInvalidSchema))
	})
}

// ValidationError represents an error that occurs during validation.
type ValidationError struct {
	Field  string // The field that caused the validation error.
	Value  any    // The value that caused the validation error.
	ErrStr string // The error message.
}

// Error allows ValidationError to satisfy the error interface.
func (ve ValidationError) Error() string {
	if len(ve.Field) > 0 {
		return ve.Field + ": " + ve.ErrStr
	} else {
		return ve.ErrStr
	}
}

// ErrInvalidSchema is an error indicating that the schema is invalid.
var ErrInvalidSchema = ValidationError{
	Field:  "invalid input",
	Value:  "",
	ErrStr: "unable to parse schema",
}

// ValidationErrors represents a collection of validation errors.
type ValidationErrors []ValidationError

// Error allows ValidationErrors to satisfy the error interface.
func (ves ValidationErrors) Error() string {
	buff := bytes.NewBufferString("")

	for i := 0; i < len(ves); i++ {
		buff.WriteString(ves[i].Error())
		buff.WriteString("; ")
	}

	return strings.TrimSpace(buff.String())
}
