package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tansive/tansive/internal/tangent/test"
)

func TestRegisterTangent(t *testing.T) {
	_ = test.SetupTest(t)
	SetTestMode(true)
	TestInit(t)
	RegisterTangent()
	runtimeConfig := GetRuntimeConfig()
	assert.NotNil(t, runtimeConfig)
	assert.True(t, runtimeConfig.Registered)
	assert.NotNil(t, runtimeConfig.TangentID)
	assert.NotNil(t, runtimeConfig.AccessKey)
	assert.NotNil(t, runtimeConfig.LogSigningKey)
}
