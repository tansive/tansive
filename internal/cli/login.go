package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/common/httpclient"
)

// loginResponse represents the response from the login endpoint
type loginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

// newLoginCmd creates and returns a new login command
func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the Tansive server",
		Long: `Login to the Tansive server to obtain an authentication token.
This command will authenticate with the server and store the token in your configuration file.

The login process requires:
- A valid server configuration
- Single user mode enabled on the server

Example:
  tansive login`,
		RunE: runLogin,
	}
}

// runLogin handles the login command execution
func runLogin(cmd *cobra.Command, args []string) error {
	// Get the current configuration
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	// Create HTTP client using the same approach as create.go
	client := httpclient.NewClient(cfg)

	// Prepare login request using the client's DoRequest method
	opts := httpclient.RequestOptions{
		Method: "POST",
		Path:   "auth/login",
	}

	// Make the login request
	body, _, err := client.DoRequest(opts)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}

	// Parse response
	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	// Update configuration with new token
	cfg.APIKey = loginResp.Token
	cfg.CurrentToken = ""
	cfg.CurrentCatalog = ""
	cfg.TokenExpiry = ""

	// Save updated configuration
	configPath, err := GetDefaultConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	if err := cfg.WriteConfig(configPath); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Print success message
	if jsonOutput {
		kv := map[string]interface{}{
			"status":     "success",
			"message":    "Login successful",
			"expires_at": loginResp.ExpiresAt.Format(time.RFC3339),
		}
		printJSON(kv)
	} else {
		okLabel.Println("✓ Login successful")
		fmt.Printf("Token expires at: %s\n", loginResp.ExpiresAt.Format(time.RFC3339))
	}

	return nil
}
