package apis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/httpx"
)

func TestError(t *testing.T) {
	err := ToHTTPXError(nil)
	assert.Nil(t, err)
	appError := apperrors.New("test error").SetStatusCode(500)
	herr := ToHTTPXError(appError)
	assert.NotNil(t, herr)
	assert.Equal(t, 500, herr.(*httpx.Error).StatusCode)
	assert.Equal(t, "test error", herr.(*httpx.Error).Description)
}
