// Package httpclient provides a configurable HTTP client for making requests to REST APIs.
// It supports authentication via API keys and tokens, handles common HTTP operations,
// and provides error handling for server responses. The package requires a Configurator
// implementation for server configuration and authentication details.
package httpclient

import (
	"bytes"
	"crypto/ed25519"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// Configurator defines the interface for providing server configuration and authentication details.
// Implementations must provide server URL, API key, and token management capabilities.
type Configurator interface {
	GetServerURL() string
	GetAPIKey() string
	GetSigningKey() (string, []byte)
	GetToken() string
	GetTokenExpiry() time.Time
}

// ServerError represents an error response from the server with a result code and error message.
type ServerError struct {
	Result int    `json:"result"` // HTTP status code or result code from server
	Error  string `json:"error"`  // Error message from server
}

// HTTPError represents an error response from the server with HTTP status code and message.
type HTTPError struct {
	StatusCode int    // HTTP status code of the error
	Message    string // Error message or response body
}

// Error implements the error interface for HTTPError.
func (e *HTTPError) Error() string {
	return e.Message
}

// HTTPClient represents a client for making HTTP requests to a REST API server.
// It handles authentication, request building, and response processing.
type HTTPClient struct {
	config     Configurator
	httpClient *http.Client
}

// ClientOptions contains options for configuring the HTTP client.
type ClientOptions struct {
	DisableCertValidation bool // If true, skips SSL certificate validation
}

// NewClient creates a new HTTP client using the provided configuration.
// The config parameter must implement the Configurator interface.
func NewClient(config Configurator, opts ...ClientOptions) *HTTPClient {
	clientOpts := ClientOptions{}
	if strings.HasPrefix(config.GetServerURL(), "https://") {
		clientOpts.DisableCertValidation = true
	}
	if len(opts) > 0 {
		clientOpts = opts[0]
	}
	return NewClientWithOptions(config, clientOpts)
}

// NewClientWithOptions creates a new HTTP client using the provided configuration and options.
// The config parameter must implement the Configurator interface.
func NewClientWithOptions(config Configurator, opts ClientOptions) *HTTPClient {
	httpClient := &http.Client{}

	if opts.DisableCertValidation {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	return &HTTPClient{
		config:     config,
		httpClient: httpClient,
	}
}

// RequestOptions contains options for making HTTP requests.
// All fields are required except QueryParams and Body.
type RequestOptions struct {
	Method      string            // HTTP method (GET, POST, PUT, DELETE)
	Path        string            // API endpoint path
	QueryParams map[string]string // Optional query parameters
	Body        []byte            // Optional request body
}

// DoRequest makes an HTTP request with the given options.
// Returns the response body, Location header (if present), and any error that occurred.
// Handles authentication using either token or API key based on availability and validity.
func (c *HTTPClient) DoRequest(opts RequestOptions) ([]byte, string, error) {
	u, err := url.Parse(c.config.GetServerURL())
	if err != nil {
		return nil, "", fmt.Errorf("invalid server URL: %v", err)
	}
	u.Path = path.Join(u.Path, opts.Path)

	q := u.Query()
	for k, v := range opts.QueryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	bodyReader := bytes.NewBuffer(opts.Body)
	req, err := http.NewRequest(opts.Method, u.String(), bodyReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use token if valid
	if c.config.GetToken() != "" && !c.config.GetTokenExpiry().IsZero() {
		expiry := c.config.GetTokenExpiry()
		if time.Now().Before(expiry) {
			req.Header.Set("Authorization", "Bearer "+c.config.GetToken())
		} else {
			if c.config.GetAPIKey() != "" {
				req.Header.Set("Authorization", "Bearer "+c.config.GetAPIKey())
			}
		}
	} else if c.config.GetAPIKey() != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.GetAPIKey())
	}
	// Sign request if SigningKey is present
	keyID, privateKeyBytes := c.config.GetSigningKey()
	if len(privateKeyBytes) == ed25519.PrivateKeySize {
		privateKey := ed25519.PrivateKey(privateKeyBytes)

		timestamp := time.Now().UTC().Format(time.RFC3339)

		// Ensure the request path starts with a slash to match server expectation
		requestPath := opts.Path
		if !strings.HasPrefix(requestPath, "/") {
			requestPath = "/" + requestPath
		}

		// Canonical string to sign - use requestPath to match server expectation
		stringToSign := strings.Join([]string{
			opts.Method,
			requestPath,
			u.RawQuery,
			string(opts.Body),
			timestamp,
		}, "\n")

		signature := ed25519.Sign(privateKey, []byte(stringToSign))
		signatureB64 := base64.StdEncoding.EncodeToString(signature)

		req.Header.Set("X-Tangent-Signature", signatureB64)
		req.Header.Set("X-Tangent-Signature-Timestamp", timestamp)
		req.Header.Set("X-TangentID", keyID)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %v", err)
	}

	if resp.StatusCode >= 400 {
		var serverErr ServerError
		if err := json.Unmarshal(body, &serverErr); err == nil && serverErr.Error != "" {
			return nil, "", &HTTPError{
				StatusCode: resp.StatusCode,
				Message:    serverErr.Error,
			}
		}
		if resp.StatusCode == http.StatusNotFound {
			return nil, "", &HTTPError{
				StatusCode: resp.StatusCode,
				Message:    "server doesn't implement this endpoint",
			}
		}
		return nil, "", &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return body, resp.Header.Get("Location"), nil
}

// CreateResource creates a new resource using the given JSON data.
// resourceType specifies the API endpoint, data contains the resource JSON,
// and queryParams are optional query parameters.
// Returns the response body, Location header, and any error that occurred.
func (c *HTTPClient) CreateResource(resourceType string, data []byte, queryParams map[string]string) ([]byte, string, error) {
	opts := RequestOptions{
		Method:      http.MethodPost,
		Path:        resourceType,
		QueryParams: queryParams,
		Body:        data,
	}
	return c.DoRequest(opts)
}

// GetResource retrieves a resource using the given resource name.
// resourceType specifies the API endpoint, resourceName identifies the resource,
// queryParams are optional query parameters, and objectType is an optional type qualifier.
// Returns the response body and any error that occurred.
func (c *HTTPClient) GetResource(resourceType string, resourceName string, queryParams map[string]string, objectType string) ([]byte, error) {
	resourceType = strings.Trim(resourceType, "/")
	resourceName = strings.Trim(resourceName, "/")

	path := strings.TrimSuffix(resourceType, "/")

	if objectType != "" {
		path = path + "/" + objectType
	}

	path = path + "/" + resourceName

	opts := RequestOptions{
		Method:      http.MethodGet,
		Path:        path,
		QueryParams: queryParams,
	}
	body, _, err := c.DoRequest(opts)
	return body, err
}

// DeleteResource deletes a resource using the given resource name.
// resourceType specifies the API endpoint, resourceName identifies the resource,
// queryParams are optional query parameters, and objectType is an optional type qualifier.
// Returns any error that occurred during the deletion.
func (c *HTTPClient) DeleteResource(resourceType string, resourceName string, queryParams map[string]string, objectType string) error {
	resourceType = strings.Trim(resourceType, "/")
	resourceName = strings.Trim(resourceName, "/")

	path := strings.TrimSuffix(resourceType, "/")

	if objectType != "" {
		path = path + "/" + objectType
	}

	path = path + "/" + resourceName

	opts := RequestOptions{
		Method:      http.MethodDelete,
		Path:        path,
		QueryParams: queryParams,
	}
	_, _, err := c.DoRequest(opts)
	return err
}

// UpdateResource updates an existing resource using the given JSON data.
// resourceType specifies the API endpoint, data contains the updated resource JSON,
// queryParams are optional query parameters, and objectType is an optional type qualifier.
// The data must contain a metadata.name field.
// Returns the response body and any error that occurred.
func (c *HTTPClient) UpdateResource(resourceType string, data []byte, queryParams map[string]string, objectType string) ([]byte, error) {
	resourceName := gjson.GetBytes(data, "metadata.name").String()
	if resourceName == "" {
		return nil, fmt.Errorf("metadata.name is required for update")
	}

	resourceType = strings.Trim(resourceType, "/")
	resourceName = strings.Trim(resourceName, "/")

	path := strings.TrimSuffix(resourceType, "/")

	if objectType != "" {
		path = path + "/" + objectType
	}

	path = path + "/" + resourceName

	opts := RequestOptions{
		Method:      http.MethodPut,
		Path:        path,
		QueryParams: queryParams,
		Body:        data,
	}
	body, _, err := c.DoRequest(opts)
	return body, err
}

// UpdateResourceValue updates a specific resource value at the given path.
// resourcePath specifies the full API endpoint path, data contains the update JSON,
// and queryParams are optional query parameters.
// Returns the response body and any error that occurred.
func (c *HTTPClient) UpdateResourceValue(resourcePath string, data []byte, queryParams map[string]string) ([]byte, error) {
	opts := RequestOptions{
		Method:      http.MethodPut,
		Path:        resourcePath,
		QueryParams: queryParams,
		Body:        data,
	}
	body, _, err := c.DoRequest(opts)
	return body, err
}

// ListResources lists resources of a specific type.
// resourceType specifies the API endpoint, and queryParams are optional query parameters.
// Returns the response body and any error that occurred.
func (c *HTTPClient) ListResources(resourceType string, queryParams map[string]string) ([]byte, error) {
	opts := RequestOptions{
		Method:      http.MethodGet,
		Path:        resourceType,
		QueryParams: queryParams,
	}
	body, _, err := c.DoRequest(opts)
	return body, err
}

// StreamRequest makes an HTTP request with the given options and returns a reader for streaming the response.
// Similar to DoRequest but returns an io.ReadCloser for streaming large responses.
// The caller is responsible for closing the returned reader.
func (c *HTTPClient) StreamRequest(opts RequestOptions) (io.ReadCloser, error) {
	u, err := url.Parse(c.config.GetServerURL())
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}
	u.Path = path.Join(u.Path, opts.Path)

	q := u.Query()
	for k, v := range opts.QueryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(opts.Method, u.String(), bytes.NewBuffer(opts.Body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if c.config.GetToken() != "" && !c.config.GetTokenExpiry().IsZero() {
		expiry := c.config.GetTokenExpiry()
		if time.Now().Before(expiry) {
			req.Header.Set("Authorization", "Bearer "+c.config.GetToken())
		} else {
			if c.config.GetAPIKey() != "" {
				req.Header.Set("Authorization", "Bearer "+c.config.GetAPIKey())
			}
		}
	} else if c.config.GetAPIKey() != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.GetAPIKey())
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %v", err)
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		var serverErr ServerError
		if err := json.Unmarshal(body, &serverErr); err == nil && serverErr.Error != "" {
			return nil, &HTTPError{
				StatusCode: resp.StatusCode,
				Message:    serverErr.Error,
			}
		}
		return nil, &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    string(body),
		}
	}

	return resp.Body, nil
}
