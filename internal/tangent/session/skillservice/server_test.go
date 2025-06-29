package skillservice

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/internal/tangent/test"
	"github.com/tansive/tansive/pkg/api"
)

// mockSession implements tangentcommon.Session interface for testing
type mockSession struct {
	id string
}

func (m *mockSession) GetSessionID() string {
	return m.id
}

func (m *mockSession) Run(ctx context.Context, params *tangentcommon.RunParams) (map[string]any, apperrors.Error) {
	return map[string]any{
		"status":  "success",
		"message": "test skill executed successfully",
	}, nil
}

func (m *mockSession) GetSkills(ctx context.Context, sessionID string) ([]api.LLMTool, apperrors.Error) {
	return []api.LLMTool{
		{
			Name:         "test-skill",
			Description:  "test skill description",
			InputSchema:  []byte("{}"),
			OutputSchema: []byte("{}"),
		},
	}, nil
}

func (m *mockSession) GetContext(ctx context.Context, sessionID, invocationID, name string) (any, apperrors.Error) {
	return 5, nil
}

func TestSkillService(t *testing.T) {
	test.SetupTestCatalog(t)
	config.SetTestMode(true)
	config.TestInit(t)
	skillService := NewSkillService(&mockSession{id: "test-session"})

	go func() {
		err := skillService.StartServer()
		require.NoError(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	serviceEndpoint, goerr := config.GetSocketPath()
	require.NoError(t, goerr)
	client, err := api.NewClient(serviceEndpoint)
	require.NoError(t, err)
	defer client.Close()

	t.Run("InvokeSkill", func(t *testing.T) {
		ctx := context.Background()
		sessionID := "test-session"
		skillName := "test-skill"
		args := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}

		result, err := client.InvokeSkill(ctx, "some-invocation-id", sessionID, skillName, args)
		require.NoError(t, err)
		require.NotNil(t, result)
		require.NotNil(t, result.Output)
	})

	t.Run("GetTools", func(t *testing.T) {
		ctx := context.Background()
		sessionID := "test-session"

		skills, err := client.GetSkills(ctx, sessionID)
		require.NoError(t, err)
		require.NotNil(t, skills)
		require.Len(t, skills, 1)
		require.Equal(t, "test-skill", skills[0].Name)
		require.Equal(t, "test skill description", skills[0].Description)
		require.Equal(t, json.RawMessage("{}"), skills[0].InputSchema)
		require.Equal(t, json.RawMessage("{}"), skills[0].OutputSchema)
	})

	t.Run("GetContext", func(t *testing.T) {
		ctx := context.Background()
		sessionID := "test-session"
		invocationID := "test-invocation-id"
		name := "test-context"
		context, err := client.GetContext(ctx, sessionID, invocationID, name)
		require.NoError(t, err)
		require.NotNil(t, context)
	})
}

func TestServerStartStop(t *testing.T) {
	test.SetupTestCatalog(t)
	config.SetTestMode(true)
	config.TestInit(t)
	skillService := NewSkillService(&mockSession{id: "test-session"})
	require.NotNil(t, skillService)

	socketPath, err := config.GetSocketPath()
	require.NoError(t, err)

	if _, err := os.Stat(socketPath); err == nil {
		os.Remove(socketPath)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- skillService.StartServer()
	}()

	time.Sleep(100 * time.Millisecond)

	_, err = os.Stat(socketPath)
	require.NoError(t, err, "socket file should exist after server start")

	skillService.StopServer()

	select {
	case err := <-errChan:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down in time")
	}

	_, err = os.Stat(socketPath)
	require.Error(t, err, "socket file should be removed after server stop")
	require.True(t, os.IsNotExist(err), "error should be 'file does not exist'")
}

func TestGetSocketPath(t *testing.T) {
	test.SetupTestCatalog(t)
	config.SetTestMode(true)
	config.TestInit(t)
	path, err := config.GetSocketPath()
	require.NoError(t, err)
	require.NotEmpty(t, path)
	require.Contains(t, path, config.DefaultSocketName)
}
