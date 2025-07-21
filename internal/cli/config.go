package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// DefaultConfigFile is the default name of the config file
const DefaultConfigFile = "config.yaml"

// Config represents the configuration for the Tansive CLI
// It contains server connection details and authentication information
type Config struct {
	// Version of the configuration file format
	Version string `yaml:"version"`
	// ServerURL is the URL and port of the Tansive server
	ServerURL string `yaml:"server_url"`
	// APIKey is the authentication token for the Tansive server
	APIKey string `yaml:"api_key"`
	// Password is the password for authentication (stored for convenience)
	Password string `yaml:"password"`
	// CurrentToken is the active token for the selected catalog
	CurrentToken string `yaml:"current_token"`
	// TokenExpiry is when the current token expires
	TokenExpiry string `yaml:"token_expiry"`
	// CurrentCatalog is the currently selected catalog
	CurrentCatalog string `yaml:"current_catalog"`
}

var config *Config

// GetDefaultConfigPath returns the default path for the config file
// It uses the OS-specific config directory (e.g., ~/.config/tansive on Linux)
func GetDefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "tansive", DefaultConfigFile), nil
}

// LoadConfig loads the configuration from the specified file
// If no file is specified, it uses the default config location
func LoadConfig(file string) error {
	if file == "" {
		var err error
		file, err = GetDefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get default config path: %w", err)
		}
	}

	yamlStr, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("unable to read config file: %w", err)
	}

	var c Config
	if err = yaml.Unmarshal(yamlStr, &c); err != nil {
		return fmt.Errorf("unable to parse config file: %w", err)
	}

	// Validate required fields
	if c.ServerURL == "" {
		return errors.New("server:port is required")
	}

	// Validate server port format
	if !strings.Contains(c.ServerURL, ":") {
		return errors.New("server:port must include port number")
	}

	// Morph the server URL before storing
	c.ServerURL = MorphServer(c.ServerURL)

	config = &c
	return nil
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	return config
}

// WriteConfig writes the current configuration to the specified file
// If no file is specified, it uses the default config location
func (cfg *Config) WriteConfig(file string) error {
	if file == "" {
		return errors.New("file path cannot be empty")
	}

	err := os.MkdirAll(filepath.Dir(file), os.ModePerm)
	if err != nil {
		return fmt.Errorf("unable to create config directory: %w", err)
	}

	yamlStr, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("unable to generate configuration: %w", err)
	}

	err = os.WriteFile(file, yamlStr, os.FileMode(0600))
	if err != nil {
		return fmt.Errorf("unable to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration
// Checks for required fields and proper formatting
func (cfg *Config) ValidateConfig() error {
	if cfg.ServerURL == "" {
		return errors.New("server:port is required")
	}
	if !strings.HasPrefix(cfg.ServerURL, "http://") && !strings.HasPrefix(cfg.ServerURL, "https://") {
		return errors.New("server:port must start with http:// or https://")
	}
	if !strings.Contains(cfg.ServerURL, ":") {
		return errors.New("server:port must include port number")
	}
	if cfg.APIKey == "" {
		return errors.New("API key is required")
	}
	return nil
}

// Print prints the current configuration in a human-readable format
func (cfg *Config) Print() {
	fmt.Printf("Server: %s\n", cfg.ServerURL)
}

// MorphServer ensures the server URL is properly formatted
// Adds http:// prefix if missing and removes trailing slashes
func MorphServer(server string) string {
	if server == "" {
		return server
	}

	// Remove any trailing slashes
	server = strings.TrimRight(server, "/")

	// Add http:// if no protocol is specified
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		server = "https://" + server
	}

	return server
}

// GetServerURL returns the properly formatted server URL
func (cfg *Config) GetServerURL() string {
	return MorphServer(cfg.ServerURL)
}

// GetAPIKey returns the API key from the configuration
func (cfg *Config) GetAPIKey() string {
	return cfg.APIKey
}

// GetSigningKey returns the API key from the configuration
func (cfg *Config) GetSigningKey() (string, []byte) {
	return "", nil
}

// GetToken returns the current token from the configuration
func (cfg *Config) GetToken() string {
	return cfg.CurrentToken
}

// GetTokenExpiry returns the token expiry time from the configuration
func (cfg *Config) GetTokenExpiry() time.Time {
	if cfg.TokenExpiry == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, cfg.TokenExpiry)
	if err != nil {
		return time.Time{}
	}
	return t
}

type TangentConfig struct {
	ServerURL   string
	Token       string
	TokenExpiry time.Time
}

func (cfg *TangentConfig) GetServerURL() string {
	return MorphServer(cfg.ServerURL)
}

func (cfg *TangentConfig) GetAPIKey() string {
	return ""
}

func (cfg *TangentConfig) GetSigningKey() (string, []byte) {
	return "", nil
}

func (cfg *TangentConfig) GetToken() string {
	return cfg.Token
}

func (cfg *TangentConfig) GetTokenExpiry() time.Time {
	return cfg.TokenExpiry
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `Manage CLI configuration settings like server connection and authentication.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if --server flag is provided
		serverFlag, _ := cmd.Flags().GetString("server")
		if serverFlag != "" {
			return setServerConfig(serverFlag)
		}

		// If no specific flag is provided, show help
		cmd.Help()
		return nil
	},
}

// configClearCmd represents the config clear command
var configClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the current token and catalog configuration",
	Long: `Clear the current token and catalog configuration. This will remove:
1. The current authentication token
2. The token expiry time
3. The current catalog selection

This is useful when you want to reset your configuration or switch to a different catalog.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := LoadConfig(configFile); err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Tansive config file not found. Configure tansive with \"tansive config create\" first.")
				os.Exit(1)
			} else {
				fmt.Printf("Unable to load config file: %s\n", err.Error())
				os.Exit(1)
			}
		}
		cfg := GetConfig()
		// Clear the token-related fields
		cfg.CurrentToken = ""
		cfg.TokenExpiry = ""
		cfg.CurrentCatalog = ""
		// Note: We don't clear Password here as it might be needed for future logins

		// Save the config
		if err := cfg.WriteConfig(configFile); err != nil {
			return fmt.Errorf("failed to save config: %v", err)
		}

		if jsonOutput {
			printJSON(map[string]int{"result": 1})
		} else {
			fmt.Println("Choose a new catalog with \"tansive set-catalog <catalog>\"")
		}

		return nil
	},
}

func init() {
	// Add flags to config command
	configCmd.Flags().String("server", "", "Set the server URL and port (e.g., example.com:8080)")

	configCmd.AddCommand(configClearCmd)
	rootCmd.AddCommand(configCmd)
}

// setServerConfig sets the server configuration in the config file
func setServerConfig(server string) error {
	configPath := configFile
	if configPath == "" {
		var err error
		configPath, err = GetDefaultConfigPath()
		if err != nil {
			return fmt.Errorf("failed to get default config path: %w", err)
		}
	}

	cfg := &Config{
		Version: "0.1.0",
	}

	if !strings.Contains(server, ":") {
		return errors.New("server must include port number (e.g., example.com:8080)")
	}

	cfg.Version = "0.1.0"
	cfg.ServerURL = MorphServer(server)

	// Clear the token-related fields
	cfg.APIKey = ""
	cfg.Password = ""
	cfg.CurrentToken = ""
	cfg.TokenExpiry = ""
	cfg.CurrentCatalog = ""

	if err := cfg.WriteConfig(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	if jsonOutput {
		printJSON(map[string]string{
			"server":      cfg.ServerURL,
			"config_file": configPath,
		})
	} else {
		fmt.Printf("Server configured: %s\n", cfg.ServerURL)
		fmt.Printf("Config file: %s\n", configPath)
	}

	return nil
}
