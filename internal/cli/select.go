package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tansive/tansive/internal/common/httpclient"
)

// setCatalogCmd represents the set-catalog command
var setCatalogCmd = &cobra.Command{
	Use:   "set-catalog CATALOG_NAME [flags]",
	Short: "Set your default view in the Catalog",
	Long: `Set your default view in the Catalog. This view will be used for all subsequent operations until you switch views.
The command will:
1. Adopt the default view for the specified catalog
2. Store the authentication token for this view
3. Use this token for all subsequent operations until you switch views or the token expires

Examples:
  # Select a catalog as your default
  tansive set-catalog my-catalog

  # Select a catalog and output in JSON format
  tansive set-catalog my-catalog -j`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		catalogName := args[0]
		client := httpclient.NewClient(GetConfig())

		// Make the POST request to get the token
		opts := httpclient.RequestOptions{
			Method: http.MethodPost,
			Path:   fmt.Sprintf("auth/default-view-adoptions/%s", catalogName),
		}

		body, _, err := client.DoRequest(opts)
		if err != nil {
			return err
		}

		// Parse the response
		var response struct {
			Token     string    `json:"token"`
			ExpiresAt time.Time `json:"expires_at"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}

		// Update the config with the new token
		cfg := GetConfig()
		cfg.CurrentToken = response.Token
		cfg.TokenExpiry = response.ExpiresAt.Format(time.RFC3339)
		cfg.CurrentCatalog = catalogName

		// Save the config
		configFile, err := GetDefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %v", err)
		}
		if err := cfg.WriteConfig(configFile); err != nil {
			return fmt.Errorf("failed to save config: %v", err)
		}

		if jsonOutput {
			printJSON(map[string]int{"result": 1})
		} else {
			fmt.Printf("Catalog set to %s\n", catalogName)
		}

		return nil
	},
}

// selectCatalogCmd represents the select-catalog command
var adoptViewCmd = &cobra.Command{
	Use:   "adopt-view CATALOG_REF/VIEW_LABEL [flags]",
	Short: "Adopt a view in the Catalog",
	Long: `Adopt a view in the Catalog. This view will be used for all subsequent operations until you switch views.
The command will:
1. Adopt the view for the specified catalog
2. Store the authentication token for this view
3. Use this token for all subsequent operations until you switch views or the token expires

Examples:
  # Adopt a view in a specific catalog
  tansive adopt-view my-catalog/my-view

  # Adopt a view in the current catalog
  tansive adopt-view my-view

  # Adopt a view and output in JSON format
  tansive adopt-view my-catalog/my-view -j`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parts := strings.SplitN(args[0], "/", 2)
		if len(parts) != 2 {
			config := GetConfig()
			if config.CurrentCatalog == "" {
				return fmt.Errorf("invalid view format. Expected <catalog-ref>/<view-label>")
			}
			parts = []string{config.CurrentCatalog, args[0]}
		}
		catalogRef := parts[0]
		viewLabel := parts[1]
		client := httpclient.NewClient(GetConfig())

		// Make the POST request to get the token
		opts := httpclient.RequestOptions{
			Method: http.MethodPost,
			Path:   fmt.Sprintf("auth/view-adoptions/%s/%s", catalogRef, viewLabel),
		}

		body, _, err := client.DoRequest(opts)
		if err != nil {
			return err
		}

		// Parse the response
		var response struct {
			Token     string    `json:"token"`
			ExpiresAt time.Time `json:"expires_at"`
		}
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("failed to parse response: %v", err)
		}

		// Update the config with the new token
		cfg := GetConfig()
		cfg.CurrentToken = response.Token
		cfg.TokenExpiry = response.ExpiresAt.Format(time.RFC3339)
		cfg.CurrentCatalog = catalogRef

		// Save the config
		configFile, err := GetDefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %v", err)
		}
		if err := cfg.WriteConfig(configFile); err != nil {
			return fmt.Errorf("failed to save config: %v", err)
		}

		if jsonOutput {
			printJSON(map[string]int{"result": 1})
		} else {
			fmt.Printf("View set to %s\n", viewLabel)
		}

		return nil
	},
}

// unsetCatalogCmd represents the unset-catalog command
var unsetCatalogCmd = &cobra.Command{
	Use:   "unset-catalog [flags]",
	Short: "Clear your default catalog selection",
	Long: `Clear your default catalog view. This will remove the current catalog, token, and token expiry from your configuration.
The command will:
1. Clear the current catalog name
2. Clear the authentication token
3. Clear the token expiry
4. Save the updated configuration

Examples:
  # Clear the current catalog configuration
  tansive unset-catalog

  # Clear the current catalog configuration and output in JSON format
  tansive unset-catalog -j`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Update the config by clearing the catalog-related fields
		cfg := GetConfig()
		cfg.CurrentToken = ""
		cfg.TokenExpiry = ""
		cfg.CurrentCatalog = ""

		// Save the config
		configFile, err := GetDefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get config path: %v", err)
		}
		if err := cfg.WriteConfig(configFile); err != nil {
			return fmt.Errorf("failed to save config: %v", err)
		}

		if jsonOutput {
			printJSON(map[string]int{"result": 1})
		} else {
			fmt.Println("Catalog configuration cleared")
		}

		return nil
	},
}

// init initializes the select-catalog and adopt-view commands and adds them to the root command
func init() {
	rootCmd.AddCommand(setCatalogCmd)
	rootCmd.AddCommand(adoptViewCmd)
	rootCmd.AddCommand(unsetCatalogCmd)
}
