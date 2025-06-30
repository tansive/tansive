package session

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/stretchr/testify/require"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/runners/stdiorunner"
	"github.com/tansive/tansive/internal/tangent/test"
	"github.com/tansive/tansive/pkg/api"
)

func TestOpenAI(t *testing.T) {
	t.Skip("Skipping openai test")
	config.TestInit(t)
	SetTestMode(true)
	stdiorunner.TestInit()
	ts := test.SetupTestCatalog(t)
	token, expiresAt := test.AdoptView(t, ts.Catalog, "dev-view", ts.Token)
	serverContext := &ServerContext{
		SessionID:      uuid.New(),
		TenantID:       ts.TenantID,
		Catalog:        ts.Catalog,
		Variant:        "dev",
		SkillSet:       test.SkillsetPath(),
		Skill:          test.SkillsetAgent(),
		View:           "dev-view",
		ViewDefinition: test.GetViewDefinition("dev"),
	}
	ctx := context.Background()
	session, err := ActiveSessionManager().CreateSession(ctx, serverContext, token, expiresAt)
	require.NoError(t, err)
	err = session.fetchObjects(ctx)
	require.NoError(t, err)
	CreateSkillService()
	// wait for few milliseconds to ensure the skill service is running
	time.Sleep(100 * time.Millisecond)
	serviceEndpoint, goerr := config.GetSocketPath()
	require.NoError(t, goerr)
	client, goerr := api.NewClient(serviceEndpoint)
	require.NoError(t, goerr)
	defer client.Close()

	tools := getTools(t, ctx, client, session.GetSessionID())
	require.NotEmpty(t, tools)

	CreateOpenAISession(t, ctx, client, session)
}

func CreateOpenAISession(t *testing.T, ctx context.Context, tansiveClient *api.Client, session *session) {
	// client := openai.NewClient(
	// 	option.WithAPIKey(getOpenAIAPIKey()),
	// )

	client := openai.NewClient(
		option.WithAPIKey(getOpenAIAPIKey()),
		//option.WithBaseURL("https://api.anthropic.com/v1"),
	)

	question := "My orders are not getting processed.  Use the tools at your disposal to fix the issue."

	params := openai.ChatCompletionNewParams{
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(question),
			openai.SystemMessage(LLMBlockedByPolicyPrompt),
		},
		Tools: getTools(t, ctx, tansiveClient, session.GetSessionID()),
		Seed:  openai.Int(0),
		Model: openai.ChatModelGPT4o,
	}

	// Loop until we get a final response without tool calls
	for {
		// Make chat completion request
		completion, err := client.Chat.Completions.New(ctx, params)
		if err != nil {
			panic(err)
		}

		finishReason := completion.Choices[0].FinishReason
		if len(finishReason) > 0 && finishReason != "tool_calls" {
			fmt.Printf("Final response: %s\n", completion.Choices[0].Message.Content)
			break
		}

		// Add the assistant's message to the conversation
		params.Messages = append(params.Messages, completion.Choices[0].Message.ToParam())

		// If there are no tool calls, we have our final response
		if len(completion.Choices[0].Message.ToolCalls) == 0 {
			fmt.Println(completion.Choices[0].Message.Content)
			break
		}

		// Handle tool calls
		for _, toolCall := range completion.Choices[0].Message.ToolCalls {
			var args = make(map[string]any)
			require.NoError(t, json.Unmarshal([]byte(toolCall.Function.Arguments), &args))
			t.Logf("Tool args: %v", args)
			invocationID := uuid.New().String()
			session.invocationIDs[invocationID] = session.viewDef
			response, err := tansiveClient.InvokeSkill(ctx, session.GetSessionID(), invocationID, toolCall.Function.Name, args)
			require.NoError(t, err)
			//fmt.Printf("Tool call response: %v\n", response)
			content, err := json.Marshal(response.Output)
			require.NoError(t, err)
			params.Messages = append(params.Messages, openai.ToolMessage(string(content), toolCall.ID))
		}
	}
}

func getTools(t *testing.T, ctx context.Context, client *api.Client, sessionID string) []openai.ChatCompletionToolParam {
	tools, err := client.GetSkills(ctx, sessionID)
	if err != nil {
		t.Fatalf("failed to get tools: %v", err)
	}

	openaiTools := []openai.ChatCompletionToolParam{}
	for _, tool := range tools {
		var inputSchema openai.FunctionParameters
		if err := json.Unmarshal([]byte(tool.InputSchema), &inputSchema); err != nil {
			t.Fatalf("failed to unmarshal input schema: %v", err)
		}

		openaiTools = append(openaiTools, openai.ChatCompletionToolParam{
			Function: openai.FunctionDefinitionParam{
				Name:        tool.Name,
				Description: openai.String(tool.Description),
				Parameters:  inputSchema,
			},
		})
	}

	return openaiTools
}

const LLMBlockedByPolicyPrompt = `
All tools with tag [TansivePolicy: true] are governed by Tansive policy.
If any tool call with such tag returns an error containing "This operation is blocked by Tansive policy", you must respond to the user with:
"I tried to use Skill: <tool-name> for <reason> but it was blocked by Tansive policy. Please contact the administrator of your Tansive system to obtain access." Do not attempt to bypass, hallucinate, or reroute the request. Respect the policy boundaries.
`
const AnthropicModel = "claude-3-5-sonnet-20240620"

func getOpenAIAPIKey() string {
	apiKey := os.Getenv("OPENAI_API_KEY")
	// if apiKey == "" {
	// 	panic("OPENAI_API_KEY is not set")
	// }
	return apiKey
}

func getAnthropicAPIKey() string {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	// if apiKey == "" {
	// 	panic("ANTHROPIC_API_KEY is not set")
	// }
	return apiKey
}

var _ = getAnthropicAPIKey()
var _ = getOpenAIAPIKey()
