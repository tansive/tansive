package config

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/catalogsrv/catcommon"
	srvtangent "github.com/tansive/tansive/internal/catalogsrv/tangent"
	"github.com/tansive/tansive/internal/common/httpclient"
	"github.com/tansive/tansive/internal/common/uuid"
)

// KeyPair represents a cryptographic key pair for authentication and signing.
// Contains private and public keys for Ed25519 cryptographic operations.
type KeyPair struct {
	PrivateKey []byte `json:"private_key"` // Ed25519 private key (32 bytes)
	PublicKey  []byte `json:"public_key"`  // Ed25519 public key (32 bytes)
}

// RuntimeConfig holds runtime configuration for the Tangent instance.
// Contains registration status, cryptographic keys, and runtime identifiers.
type RuntimeConfig struct {
	TangentID     uuid.UUID `json:"tangent_id"`      // unique identifier for this tangent instance
	Registered    bool      `json:"registered"`      // whether this tangent is registered with catalog server
	RegisteredAt  time.Time `json:"registered_at"`   // timestamp when registration occurred
	AccessKey     KeyPair   `json:"access_key"`      // key pair for API access authentication
	LogSigningKey KeyPair   `json:"log_signing_key"` // key pair for audit log signing
}

var runtimeConfig *RuntimeConfig

// GetRuntimeConfig returns the current runtime configuration instance.
// Provides access to runtime settings and cryptographic keys.
func GetRuntimeConfig() *RuntimeConfig {
	return runtimeConfig
}

// GetAuditLogDir returns the directory path for audit log storage.
// Constructs the path relative to the working directory configuration.
func GetAuditLogDir() string {
	appDataDir := Config().WorkingDir
	return filepath.Join(appDataDir, "auditlogs")
}

// CreateAuditLogDir creates the audit log directory if it doesn't exist.
// Creates the directory with appropriate permissions for log storage.
func CreateAuditLogDir() {
	dir := GetAuditLogDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal().Err(err).Msg("failed to create audit log dir")
		}
	}
}

// GetRuntimeConfigDir returns the directory path for runtime configuration storage.
// Uses test-specific directory when in test mode for isolation.
func GetRuntimeConfigDir() string {
	appDataDir := Config().WorkingDir
	if isTestMode {
		return filepath.Join(appDataDir, "runtime-test")
	}
	return filepath.Join(appDataDir, "runtime")
}

// CreateRuntimeConfigDir creates the runtime configuration directory if it doesn't exist.
// Creates the directory with appropriate permissions for configuration storage.
func CreateRuntimeConfigDir() {
	dir := GetRuntimeConfigDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal().Err(err).Msg("failed to create runtime dir")
		}
	}
}

// LoadRuntimeConfig loads runtime configuration from persistent storage.
// Creates new configuration with generated keys if no existing configuration is found.
// Returns an error if loading or key generation fails.
func LoadRuntimeConfig() error {
	runtimeDir := GetRuntimeConfigDir()
	configPath := filepath.Join(runtimeDir, "runtime.json")

	runtimeConfig = &RuntimeConfig{}

	if _, err := os.Stat(configPath); err == nil {
		f, err := os.Open(configPath)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to open runtime config file")
		}
		defer f.Close()

		if err := json.NewDecoder(f).Decode(runtimeConfig); err != nil {
			log.Fatal().Err(err).Msg("failed to decode runtime config")
		}
		return nil
	} else {
		runtimeConfig.TangentID = uuid.New()
		runtimeConfig.Registered = false
		accessKey, err := createKeyPair()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create access key")
		}
		logKey, err := createKeyPair()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create log key")
		}
		runtimeConfig.AccessKey = accessKey
		runtimeConfig.LogSigningKey = logKey
		if err := saveRuntimeConfig(); err != nil {
			log.Fatal().Err(err).Msg("failed to save runtime config")
		}
	}
	return nil
}

// RuntimeInit initializes runtime configuration and creates necessary directories.
// Must be called after configuration loading to set up runtime environment.
func RuntimeInit() {
	CreateRuntimeConfigDir()
	CreateAuditLogDir()
	LoadRuntimeConfig()
}

// RegisterTangent registers this Tangent instance with the catalog server.
// Sends registration request with capabilities and public keys.
// Returns an error if registration fails after retry attempts.
func RegisterTangent() error {
	if runtimeConfig.Registered {
		log.Info().Msg("tangent already registered. Updating...")
	}

	tangentInfo := &srvtangent.TangentInfo{
		ID:                     runtimeConfig.TangentID,
		URL:                    GetURL(),
		PublicKeyAccessKey:     runtimeConfig.AccessKey.PublicKey,
		PublicKeyLogSigningKey: runtimeConfig.LogSigningKey.PublicKey,
		Capabilities: []catcommon.RunnerID{
			catcommon.StdioRunnerID,
		},
	}

	client := getHTTPClient(&clientConfig{
		serverURL: Config().TansiveServer.GetURL(),
	})

	req, err := json.Marshal(tangentInfo)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to marshal tangent info")
	}

	opts := httpclient.RequestOptions{
		Method: http.MethodPost,
		Path:   "/tangents",
		Body:   req,
	}

	var location string
	retry.Do(func() error {
		_, location, err = client.DoRequest(opts)
		if err != nil {
			return err
		}
		return nil
	}, retry.Attempts(5),
		retry.Delay(1*time.Second),
		retry.DelayType(retry.BackOffDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			log.Error().Err(err).Msg("failed to register tangent")
		}))
	if err != nil {
		log.Fatal().Err(err).Msg("exhausted retries")
		return err
	}

	runtimeConfig.TangentID = uuid.MustParse(strings.TrimPrefix(string(location), "/tangents/"))
	runtimeConfig.Registered = true
	runtimeConfig.RegisteredAt = time.Now()

	return saveRuntimeConfig()
}

// saveRuntimeConfig persists runtime configuration to storage.
// Saves the current runtime configuration to JSON file.
// Returns an error if file creation or encoding fails.
func saveRuntimeConfig() error {
	runtimeDir := GetRuntimeConfigDir()
	configPath := filepath.Join(runtimeDir, "runtime.json")
	f, err := os.Create(configPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to create runtime config file")
		return err
	}
	defer f.Close()
	if err := json.NewEncoder(f).Encode(runtimeConfig); err != nil {
		log.Error().Err(err).Msg("failed to encode runtime config")
		return err
	}
	return nil
}

// deleteRuntimeConfig removes the runtime configuration file.
// Used for cleanup during testing to ensure fresh configuration.
func deleteRuntimeConfig() {
	runtimeDir := GetRuntimeConfigDir()
	configPath := filepath.Join(runtimeDir, "runtime.json")
	os.Remove(configPath)
}

// createKeyPair generates a new Ed25519 cryptographic key pair.
// Returns the generated key pair and any error encountered during generation.
func createKeyPair() (KeyPair, error) {
	// create ED25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		return KeyPair{}, err
	}
	return KeyPair{
		PrivateKey: privKey,
		PublicKey:  pubKey,
	}, nil
}

var isTestMode bool

// SetTestMode enables or disables test mode for runtime configuration.
// In test mode, configuration uses test-specific directories for isolation.
func SetTestMode(testMode bool) {
	isTestMode = testMode
}

// getHTTPClient creates an HTTP client for catalog server communication.
// Returns a configured client for making requests to the Tansive server.
func getHTTPClient(config *clientConfig) httpclient.HTTPClientInterface {
	if isTestMode {
		c, err := httpclient.NewTestClient(config)
		if err != nil {
			return nil
		}
		return c
	}
	return httpclient.NewClient(config)
}

// clientConfig defines configuration for HTTP client creation.
// Contains authentication and connection settings for server communication.
type clientConfig struct {
	token       string    // authentication token
	tokenExpiry time.Time // token expiration time
	serverURL   string    // server URL for connection
}

// GetToken returns the authentication token for the client.
func (c *clientConfig) GetToken() string {
	return c.token
}

// GetAPIKey returns the API key for the client.
// Currently returns empty string as API keys are not used.
func (c *clientConfig) GetAPIKey() string {
	return ""
}

// GetTokenExpiry returns the expiration time for the authentication token.
func (c *clientConfig) GetTokenExpiry() time.Time {
	return c.tokenExpiry
}

// GetServerURL returns the server URL for the client.
func (c *clientConfig) GetServerURL() string {
	return c.serverURL
}

// GetSigningKey returns the signing key ID and private key for request signing.
// Returns empty values if signing is not configured.
func (c *clientConfig) GetSigningKey() (string, []byte) {
	if runtimeConfig != nil && runtimeConfig.Registered {
		return runtimeConfig.TangentID.String(), runtimeConfig.AccessKey.PrivateKey
	}
	return "", nil
}
