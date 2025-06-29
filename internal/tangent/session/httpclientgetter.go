package session

import (
	"time"

	"github.com/tansive/tansive/internal/common/httpclient"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/runners"
)

// clientConfig defines the configuration for HTTP client creation.
// Contains authentication tokens, signing keys, and server connection details.
type clientConfig struct {
	signingKey   []byte
	signingKeyID string
	token        string
	tokenExpiry  time.Time
	serverURL    string
}

// GetToken returns the authentication token for the client.
func (c *clientConfig) GetToken() string {
	return c.token
}

// GetTokenExpiry returns the expiration time for the authentication token.
func (c *clientConfig) GetTokenExpiry() time.Time {
	return c.tokenExpiry
}

// GetAPIKey returns the API key for the client.
// Currently returns empty string as API keys are not used.
func (c *clientConfig) GetAPIKey() string {
	return ""
}

// GetServerURL returns the server URL for the client.
func (c *clientConfig) GetServerURL() string {
	return c.serverURL
}

// GetSigningKey returns the signing key ID and private key for request signing.
// Returns empty values if signing is not configured.
func (c *clientConfig) GetSigningKey() (string, []byte) {
	if len(c.signingKey) == 0 || c.signingKeyID == "" {
		return "", nil
	}
	return c.signingKeyID, c.signingKey
}

// getHTTPClient creates an HTTP client with the given configuration.
// Returns a test client in test mode or a production client otherwise.
func getHTTPClient(clientConfig *clientConfig) httpclient.HTTPClientInterface {
	runtimeConfig := config.GetRuntimeConfig()
	if runtimeConfig != nil && runtimeConfig.Registered {
		clientConfig.signingKey = runtimeConfig.AccessKey.PrivateKey
		clientConfig.signingKeyID = runtimeConfig.TangentID.String()
	}
	if isTestMode {
		c, err := httpclient.NewTestClient(clientConfig)
		if err != nil {
			return nil
		}
		return c
	}
	return httpclient.NewClient(clientConfig)
}

var isTestMode bool

// SetTestMode enables or disables test mode for HTTP client creation.
// In test mode, test clients are returned instead of production clients.
func SetTestMode(testMode bool) {
	isTestMode = testMode
}

// Init initializes the session package dependencies.
// Must be called before using session functionality.
func Init() {
	runners.Init()
}
