package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/common/httpclient"
)

// StatusResponse represents the response from the /status endpoint
type StatusResponse struct {
	UserID        string          `json:"userID,omitempty"`
	ServerTime    string          `json:"serverTime,omitempty"`
	ServerVersion string          `json:"serverVersion"`
	ApiVersion    string          `json:"apiVersion"`
	ViewDef       *ViewDefinition `json:"viewDef,omitempty"`
}

// ViewDefinition represents the view definition structure
type ViewDefinition struct {
	Scope Scope  `json:"scope"`
	Rules []Rule `json:"rules"`
}

// Scope represents the scope of a view
type Scope struct {
	Catalog   string `json:"catalog"`
	Variant   string `json:"variant"`
	Namespace string `json:"namespace"`
}

// Rule represents a policy rule
type Rule struct {
	Intent  string   `json:"intent"`
	Actions []string `json:"actions"`
	Targets []string `json:"targets"`
}

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Get server status and view information",
	Long: `Get server status and view information. This command returns information about the server,
including version, API version, server time, user ID, and current view definition.

Examples:
  # Get server status
  tansive status

  # Get server status in JSON format
  tansive status -j`,
	RunE: getStatus,
}

// getStatus handles retrieving server status information
func getStatus(cmd *cobra.Command, args []string) error {
	// Load the config file first
	LoadConfig(configFile)

	config := GetConfig()
	if config == nil {
		if jsonOutput {
			kv := map[string]string{
				"version_cli": getCLIVersion(),
				"error":       "Config file cannot be loaded",
			}
			printJSON(kv)
		} else {
			fmt.Printf("tansive CLI %s\n", getCLIVersion())
			fmt.Println("Error: Config file cannot be loaded")
		}
		return ErrAlreadyHandled
	}

	client := httpclient.NewClient(config)

	opts := httpclient.RequestOptions{
		Method: "GET",
		Path:   "status",
	}

	response, _, err := client.DoRequest(opts)
	if err != nil {
		if jsonOutput {
			kv := map[string]string{
				"version_cli": getCLIVersion(),
				"error":       "Unable to connect to server: " + err.Error(),
			}
			printJSON(kv)
		} else {
			fmt.Printf("tansive CLI %s\n", getCLIVersion())
			fmt.Println("Error: Unable to connect to server: " + err.Error())
		}
		return ErrAlreadyHandled
	}

	var statusResp StatusResponse
	if err := json.Unmarshal(response, &statusResp); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if jsonOutput {
		// Format as JSON with result and value
		output := map[string]any{
			"result":      1,
			"version_cli": getCLIVersion(),
			"value":       statusResp,
		}

		jsonBytes, err := json.MarshalIndent(output, "", "    ")
		if err != nil {
			return fmt.Errorf("failed to format JSON output: %v", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		// Pretty print the status information
		fmt.Printf("tansive CLI %s\n", getCLIVersion())
		printStatusPretty(statusResp)
	}

	return nil
}

// printStatusPretty prints the status information in a human-readable format
func printStatusPretty(status StatusResponse) {
	// Server information
	fmt.Printf("Server Version: %s\n", status.ServerVersion)
	fmt.Printf("API Version: %s\n", status.ApiVersion)
	if status.ServerTime != "" {
		// Parse the server time and convert to local time
		if serverTime, err := time.Parse(time.RFC3339, status.ServerTime); err == nil {
			localTime := serverTime.Local()
			fmt.Printf("Server Time: %s\n", localTime.Format("2006-01-02 15:04:05 MST"))
		} else {
			// Fallback to original format if parsing fails
			fmt.Printf("Server Time: %s\n", status.ServerTime)
		}
	}
	if status.UserID != "" {
		fmt.Printf("User ID: %s\n", status.UserID)
	}

	// View definition information
	fmt.Println()
	if status.ViewDef != nil {
		fmt.Println("View Definition:")
		fmt.Printf("  Scope:\n")
		if status.ViewDef.Scope.Catalog != "" {
			fmt.Printf("    Catalog: %s\n", status.ViewDef.Scope.Catalog)
		}
		if status.ViewDef.Scope.Variant != "" {
			fmt.Printf("    Variant: %s\n", status.ViewDef.Scope.Variant)
		}
		if status.ViewDef.Scope.Namespace != "" {
			fmt.Printf("    Namespace: %s\n", status.ViewDef.Scope.Namespace)
		}

		if len(status.ViewDef.Rules) > 0 {
			fmt.Printf("  Rules:\n")
			for i, rule := range status.ViewDef.Rules {
				fmt.Printf("    Rule %d:\n", i+1)
				fmt.Printf("      Intent: %s\n", rule.Intent)
				if len(rule.Actions) > 0 {
					fmt.Printf("      Actions: %s\n", strings.Join(rule.Actions, ", "))
				}
				if len(rule.Targets) > 0 {
					fmt.Printf("      Targets: %s\n", strings.Join(rule.Targets, ", "))
				}
			}
		}
	} else {
		fmt.Println("No Catalog is set")
	}
}

// init initializes the status command and adds it to the root command
func init() {
	rootCmd.AddCommand(statusCmd)
}
