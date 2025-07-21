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
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the Tansive server",
		Long: `Login to the Tansive server to obtain an authentication token.
This command will authenticate with the server and store the token in your configuration file.

The login process requires:
- A valid server configuration
- Single user mode enabled on the server
- A password (provided via --passwd or stored in config)

Example:
  tansive login --passwd=mypassword
  tansive login  # uses password from config file`,
		RunE: runLogin,
	}

	cmd.Flags().String("passwd", "", "Password for authentication")
	return cmd
}

// runLogin handles the login command execution
func runLogin(cmd *cobra.Command, args []string) error {
	// Get the current configuration
	cfg := GetConfig()
	if cfg == nil {
		return fmt.Errorf("no configuration loaded")
	}

	passwd, _ := cmd.Flags().GetString("passwd")
	if passwd == "" {
		passwd = cfg.Password
		if passwd == "" {
			return fmt.Errorf("no password provided. Use --passwd flag or set password in config file")
		}
	}

	client := httpclient.NewClient(cfg)

	opts := httpclient.RequestOptions{
		Method: "POST",
		Path:   "auth/login",
		QueryParams: map[string]string{
			"password": passwd,
		},
	}

	body, _, err := client.DoRequest(opts)
	if err != nil {
		return fmt.Errorf("login request failed: %w", err)
	}

	// Parse response
	var loginResp loginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("failed to parse login response: %w", err)
	}

	cfg.APIKey = loginResp.Token
	cfg.CurrentToken = ""
	cfg.CurrentCatalog = ""
	cfg.TokenExpiry = ""

	if passwd != "" {
		cfg.Password = passwd
	}

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
