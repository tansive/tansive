package cli

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	srvsession "github.com/tansive/tansive/internal/catalogsrv/session"
	"github.com/tansive/tansive/internal/common/httpclient"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
)

// sessionCmd represents the session command
var sessionCmd = &cobra.Command{
	Use:   "session [command]",
	Short: "Manage sessions in the Catalog",
	Long: `Create, get, and manage sessions in the Catalog.
A session represents an instance of a skill execution with its own state and context.

Available Commands:
  create         Create a new session
  list-sessions  List all sessions
  describe       Describe a specific session`,
}

// createSessionCmd represents the create subcommand
var createSessionCmd = &cobra.Command{
	Use:   "create SKILL_PATH [flags]",
	Short: "Create a new session in the Catalog",
	Long: `Create a new session in the Catalog. This will create a session with the specified skill path and view.
The command will:
1. Create a new session with the specified skill path and view
2. Optionally set session variables and input arguments
3. Return the session ID and other details

Examples:
  # Create a session with a specific view
  tansive session create /valid-skillset/test-skill --view valid-view

  # Create a session with session variables
  tansive session create /valid-skillset/test-skill --session-vars '{"key1":"value1"}'

  # Create a session with input arguments
  tansive session create /valid-skillset/test-skill --input-args '{"input":"test input"}'

  # Create a session with all options
  tansive session create /valid-skillset/test-skill --view valid-view --session-vars '{"key1":"value1"}' --input-args '{"input":"test input"}'`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		skillPath := args[0]
		client := httpclient.NewClient(GetConfig())

		var sessionVars map[string]any
		if sessionVarsStr != "" {
			if err := json.Unmarshal([]byte(sessionVarsStr), &sessionVars); err != nil {
				return fmt.Errorf("invalid session variables JSON: %v", err)
			}
		}

		var inputArgs map[string]any
		if inputArgsStr != "" {
			if err := json.Unmarshal([]byte(inputArgsStr), &inputArgs); err != nil {
				return fmt.Errorf("invalid input arguments JSON: %v", err)
			}
		}

		requestBody := map[string]any{
			"skillPath": skillPath,
			"viewName":  viewName,
		}
		if sessionVars != nil {
			requestBody["sessionVariables"] = sessionVars
		}
		if inputArgs != nil {
			requestBody["inputArgs"] = inputArgs
		}

		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %v", err)
		}

		// Generate code verifier (high-entropy random) and challenge
		codeVerifier, err := generateCodeVerifier()
		if err != nil {
			return fmt.Errorf("failed to generate code verifier: %v", err)
		}
		hashed := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hashed[:])

		opts := httpclient.RequestOptions{
			Method: http.MethodPost,
			Path:   "sessions",
			Body:   bodyBytes,
			QueryParams: map[string]string{
				"interactive":    fmt.Sprintf("%v", interactive),
				"code_challenge": codeChallenge,
			},
		}

		body, _, err := client.DoRequest(opts)
		if err != nil {
			return err
		}

		var response srvsession.InteractiveSessionRsp
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}

		if interactive {
			req := &tangentcommon.SessionCreateRequest{
				SessionType:  tangentcommon.SessionTypeInteractive,
				Code:         response.Code,
				CodeVerifier: codeVerifier,
			}
			err = createSession(req, response.TangentURL)
			if err != nil {
				return err
			}

			return nil
		}

		req := &tangentcommon.SessionCreateRequest{
			SessionType:  tangentcommon.SessionTypeMCPProxy,
			Code:         response.Code,
			CodeVerifier: codeVerifier,
		}
		tangentConfig := TangentConfig{
			ServerURL: response.TangentURL,
		}
		client = httpclient.NewClient(&tangentConfig)
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %v", err)
		}

		opts = httpclient.RequestOptions{
			Method: http.MethodPost,
			Path:   "/sessions",
			Body:   reqJSON,
		}

		_, url, err := client.DoRequest(opts)
		if err != nil {
			return err
		}

		if jsonOutput {
			output := map[string]any{
				"result": 1,
				"value": map[string]any{
					"mcp_endpoint": url,
				},
			}
			jsonBytes, err := json.MarshalIndent(output, "", "    ")
			if err != nil {
				return fmt.Errorf("failed to format JSON output: %v", err)
			}
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Println("Session created. MCP endpoint:")
			fmt.Println(url)
		}
		return nil
	},
}

// createSession creates a new session with the given request parameters
// It streams the session output and formats it based on the output format
func createSession(req *tangentcommon.SessionCreateRequest, serverURL string) error {
	tangentConfig := TangentConfig{
		ServerURL: serverURL,
	}
	client := httpclient.NewClient(&tangentConfig)
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	opts := httpclient.RequestOptions{
		Method: http.MethodPost,
		Path:   "/sessions",
		Body:   reqJSON,
	}

	reader, err := client.StreamRequest(opts)
	if err != nil {
		return err
	}
	defer reader.Close()

	bufReader := bufio.NewReader(reader)

	// Read chunks until EOF
	for {
		line, err := bufReader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if jsonOutput {
			fmt.Print(line)
		} else {
			PrettyPrintNDJSONLine([]byte(line))
		}
	}
	return nil
}

// printSessions formats and prints a list of sessions in either JSON or table format
func printSessions(response []byte) error {
	sessions := []srvsession.SessionSummaryInfo{}
	if err := json.Unmarshal(response, &sessions); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if jsonOutput {
		output := map[string]any{
			"result": 1,
			"value":  sessions,
		}

		jsonBytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %v", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		// Print header
		fmt.Printf("%-36s %-12s %-25s %-25s %-20s\n", "SESSION ID", "STATUS", "STARTED", "UPDATED", "CREATED BY")
		fmt.Println(strings.Repeat("-", 120))

		// Print each session (max 10)
		count := 0
		for _, session := range sessions {
			if count >= 10 {
				break
			}

			// Format timestamps
			startedAt := formatTimestampInLocalTimezone(session.StartedAt)
			endedAt := "N/A"
			// Check for invalid/zero dates (year 0000 or empty)
			if !session.UpdatedAt.IsZero() && session.UpdatedAt.Year() > 0 {
				endedAt = formatTimestampInLocalTimezone(session.UpdatedAt)
			}

			fmt.Printf("%-36s %-12s %-25s %-25s %-20s\n",
				session.SessionID,
				session.StatusSummary,
				startedAt,
				endedAt,
				session.UserID)
			count++
		}
	}
	return nil
}

// listSessionsCmd represents the list-sessions subcommand
var listSessionsCmd = &cobra.Command{
	Use:   "list [flags]",
	Short: "List all sessions in the Catalog",
	Long: `List all sessions in the Catalog, showing their status, timestamps, and other details.
The output includes:
- Session ID
- Status
- Start time
- Last update time
- Created by

Examples:
  # List all sessions
  tansive session list

  # List sessions in JSON format
  tansive session list -j`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := httpclient.NewClient(GetConfig())

		response, err := client.ListResources("sessions", nil)
		if err != nil {
			return err
		}

		return printSessions(response)
	},
}

// describeSessionCmd represents the describe subcommand
var describeSessionCmd = &cobra.Command{
	Use:   "describe SESSION_ID [flags]",
	Short: "Describe a session in the Catalog",
	Long: `Describe a session in the Catalog by its ID. This will show detailed information about the session,
including its status, timestamps, and other metadata.

Examples:
  # Describe a specific session
  tansive session describe 123e4567-e89b-12d3-a456-426614174000

  # Describe a session in JSON format
  tansive session describe 123e4567-e89b-12d3-a456-426614174000 -j`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		client := httpclient.NewClient(GetConfig())

		queryParams := map[string]string{
			"sessionID": sessionID,
		}
		response, err := client.GetResource("sessions", sessionID, queryParams, "")
		if err != nil {
			return err
		}

		var session srvsession.SessionSummaryInfo
		if err := json.Unmarshal(response, &session); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}

		if jsonOutput {
			output := map[string]any{
				"result": 1,
				"value":  session,
			}

			jsonBytes, err := json.MarshalIndent(output, "", "    ")
			if err != nil {
				return fmt.Errorf("failed to format JSON output: %v", err)
			}
			fmt.Println(string(jsonBytes))
		} else {
			// Print session details in a readable format
			fmt.Printf("Session ID: %s\n", session.SessionID)
			fmt.Printf("Status: %s\n", session.StatusSummary)
			fmt.Printf("Created At: %s\n", formatTimestampInLocalTimezone(session.CreatedAt))
			fmt.Printf("Started At: %s\n", formatTimestampInLocalTimezone(session.StartedAt))
			if !session.UpdatedAt.IsZero() && session.UpdatedAt.Year() > 0 {
				fmt.Printf("Updated At: %s\n", formatTimestampInLocalTimezone(session.UpdatedAt))
			}
			fmt.Printf("Created By: %s\n", session.UserID)
			if len(session.Error) > 0 {
				fmt.Printf("Error: %v\n", session.Error)
			}
		}
		return nil
	},
}

// stopSessionCmd represents the stop subcommand
var stopSessionCmd = &cobra.Command{
	Use:   "stop SESSION_ID [flags]",
	Short: "Stop a session in the Catalog",
	Long: `Stop a session in the Catalog by its ID. This will terminate the session and clean up resources.

Examples:
  # Stop a specific session
tansive session stop 123e4567-e89b-12d3-a456-426614174000

  # Stop a session and output in JSON format
tansive session stop 123e4567-e89b-12d3-a456-426614174000 -j`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sessionID := args[0]
		client := httpclient.NewClient(GetConfig())

		// Generate code verifier (high-entropy random) and challenge
		codeVerifier, err := generateCodeVerifier()
		if err != nil {
			return fmt.Errorf("failed to generate code verifier: %v", err)
		}
		hashed := sha256.Sum256([]byte(codeVerifier))
		codeChallenge := base64.RawURLEncoding.EncodeToString(hashed[:])

		// Step 1: Request stop from the catalog server (DELETE)
		opts := httpclient.RequestOptions{
			Method: http.MethodDelete,
			Path:   "sessions/" + sessionID,
			QueryParams: map[string]string{
				"code_challenge": codeChallenge,
			},
		}

		body, _, err := client.DoRequest(opts)
		if err != nil {
			return err
		}

		var response srvsession.InteractiveSessionRsp
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}

		// Step 2: Send stop request to the tangent (DELETE)
		req := &tangentcommon.SessionCreateRequest{
			SessionType:  tangentcommon.SessionTypeInteractive, // SessionType is not used for stop, but keep for consistency
			Code:         response.Code,
			CodeVerifier: codeVerifier,
		}
		tangentConfig := TangentConfig{
			ServerURL: response.TangentURL,
		}
		client = httpclient.NewClient(&tangentConfig)
		reqJSON, err := json.Marshal(req)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %v", err)
		}

		opts = httpclient.RequestOptions{
			Method: http.MethodDelete,
			Path:   "/sessions",
			Body:   reqJSON,
		}

		_, _, err = client.DoRequest(opts)
		if err != nil {
			return err
		}

		if jsonOutput {
			output := map[string]any{
				"result": 1,
				"value": map[string]any{
					"status":     "stopped",
					"session_id": sessionID,
				},
			}
			jsonBytes, err := json.MarshalIndent(output, "", "    ")
			if err != nil {
				return fmt.Errorf("failed to format JSON output: %v", err)
			}
			fmt.Println(string(jsonBytes))
		} else {
			fmt.Printf("Session %s stopped successfully.\n", sessionID)
		}
		return nil
	},
}

// formatTimestampInLocalTimezone formats a timestamp in local timezone
// It handles the case where the timestamp might already be in local timezone
func formatTimestampInLocalTimezone(t time.Time) string {
	if t.Location() == time.Local {
		return t.Format("2006-01-02 15:04:05 MST")
	}
	return t.Local().Format("2006-01-02 15:04:05 MST")
}

// generateCodeVerifier generates a high-entropy cryptographically random code verifier for PKCE
func generateCodeVerifier() (string, error) {
	b := make([]byte, 32) // 32 bytes = 256 bits
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

var (
	sessionVarsStr string
	inputArgsStr   string
	viewName       string
	interactive    bool
)

// init initializes the session command and its subcommands
func init() {
	rootCmd.AddCommand(sessionCmd)
	sessionCmd.AddCommand(createSessionCmd)
	sessionCmd.AddCommand(listSessionsCmd)
	sessionCmd.AddCommand(describeSessionCmd)
	sessionCmd.AddCommand(stopSessionCmd)

	createSessionCmd.Flags().StringVar(&viewName, "view", "", "Name of the view to use (required)")
	createSessionCmd.MarkFlagRequired("view")
	createSessionCmd.Flags().StringVar(&sessionVarsStr, "session-vars", "", "JSON string of session variables")
	createSessionCmd.Flags().StringVar(&inputArgsStr, "input-args", "", "JSON string of input arguments")
	createSessionCmd.Flags().BoolVar(&interactive, "interactive", false, "Set interactive mode for the session (default: false)")
}
