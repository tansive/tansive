package session

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/runners/stdiorunner"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/internal/tangent/test"
	"github.com/tansive/tansive/pkg/api"
)

func TestCreateSession(t *testing.T) {
	config.SetTestMode(true)
	ts := test.SetupTestCatalog(t)
	config.TestInit(t)
	SetTestMode(true)
	Init()
	stdiorunner.TestInit()
	token, expiresAt := test.AdoptView(t, ts.Catalog, "prod-view", ts.Token)
	serverContext := &ServerContext{
		SessionID:      uuid.New(),
		TenantID:       ts.TenantID,
		Catalog:        ts.Catalog,
		Variant:        "prod",
		SkillSet:       test.SkillsetPath(),
		Skill:          test.SkillsetAgent(),
		View:           "prod-view",
		ViewDefinition: test.GetViewDefinition("prod"),
	}
	ctx := context.Background()
	session, err := ActiveSessionManager().CreateSession(ctx, serverContext, token, expiresAt)
	require.NoError(t, err)
	err = session.fetchObjects(ctx)
	auditCtx, auditCancel := context.WithCancel(ctx)
	InitAuditLog(auditCtx, session)
	defer auditCancel()
	require.NoError(t, err)
	tCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	outWriter := tangentcommon.NewBufferedWriter()
	errWriter := tangentcommon.NewBufferedWriter()
	err = session.Run(tCtx, "", "k8s_troubleshooter", map[string]any{
		"prompt": "I'm getting a 500 error when I try to access the API",
	}, &tangentcommon.IOWriters{
		Out: outWriter,
		Err: errWriter,
	})
	require.NoError(t, err)
	t.Logf("outWriter: %s", outWriter.String())
	t.Logf("errWriter: %s", errWriter.String())
	CreateSkillService()
	// wait for few milliseconds to ensure the skill service is running
	time.Sleep(100 * time.Millisecond)
	serviceEndpoint, goerr := config.GetSocketPath()
	require.NoError(t, goerr)
	client, goerr := api.NewClient(serviceEndpoint)
	require.NoError(t, goerr)
	defer client.Close()

	// // empty invocationID should return an error
	_, goerr = client.InvokeSkill(ctx, session.GetSessionID(), "", "k8s_troubleshooter", map[string]any{
		"prompt": "I'm getting a 500 error when I try to access the API",
	})
	require.Error(t, goerr)

	// reusing invocationID should return a loop error
	var invocationID string
	for k := range session.invocationIDs {
		invocationID = k
		break
	}
	response, goerr := client.InvokeSkill(ctx, session.GetSessionID(), invocationID, "k8s_troubleshooter", map[string]any{
		"prompt": "I'm getting a 500 error when I try to access the API",
	})
	require.NoError(t, goerr)
	require.True(t, isError(response))

	// for testing, append a new invocationID to the session
	invocationID = uuid.New().String()
	session.invocationIDs[invocationID] = session.viewDef
	response, goerr = client.InvokeSkill(ctx, session.GetSessionID(), invocationID, "list_pods", map[string]any{
		"labelSelector": "app=my-app",
	})
	if !assert.NoError(t, goerr) {
		t.Logf("error: %v", goerr.Error())
		if response != nil {
			t.Logf("response: %v", response)
		}
		t.Fail()
	}
	if response != nil {
		t.Logf("response output: %s", response.Output["content"])
	}

	response, goerr = client.InvokeSkill(ctx, session.GetSessionID(), invocationID, "list_pods", map[string]any{
		"labelSelector": "app=my-app",
	})
	require.NoError(t, goerr)
	require.False(t, isError(response))
	t.Logf("response output: %v", response)

	response, goerr = client.InvokeSkill(ctx, session.GetSessionID(), invocationID, "restart_deployment", map[string]any{
		"deployment": "my-app",
	})
	require.NoError(t, goerr)
	require.True(t, isError(response))
	t.Logf("response outpu: %v", response)

	// // test get tools
	skills, goerr := client.GetSkills(ctx, session.GetSessionID())
	require.NoError(t, goerr)
	require.NotNil(t, skills)
	t.Logf("skills: %v", skills)
}

func isError(response *api.SkillResult) bool {
	_, ok := response.Output["error"]
	return ok
}
