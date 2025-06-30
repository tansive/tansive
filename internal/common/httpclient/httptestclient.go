package httpclient

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/tansive/tansive/internal/catalogsrv/server"
	"github.com/tidwall/gjson"
)

// TestHTTPClient represents a test client for making HTTP requests directly to the catalog server.
// It uses httptest.NewRecorder to capture responses without making actual network calls.
type TestHTTPClient struct {
	config     Configurator
	httpServer *server.CatalogServer
}

// NewTestClient creates a new test HTTP client using the provided configuration.
// It initializes a test server instance and mounts the necessary handlers.
// Returns an error if server creation fails.
func NewTestClient(config Configurator) (*TestHTTPClient, error) {
	s, err := server.CreateNewServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create test server: %v", err)
	}
	s.MountHandlers()

	return &TestHTTPClient{
		config:     config,
		httpServer: s,
	}, nil
}

// DoRequest makes an HTTP request with the given options directly to the test server.
// Uses httptest.NewRecorder to capture the response without making network calls.
// Returns the response body, Location header (if present), and any error that occurred.
func (c *TestHTTPClient) DoRequest(opts RequestOptions) ([]byte, string, error) {
	u, err := url.Parse(c.config.GetServerURL())
	if err != nil {
		return nil, "", fmt.Errorf("invalid server URL: %v", err)
	}
	if u.Path == "" {
		u.Path = "/"
	}
	u.Path = path.Join(u.Path, opts.Path)

	q := u.Query()
	for k, v := range opts.QueryParams {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(opts.Method, u.String(), bytes.NewBuffer(opts.Body))
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %v", err)
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

		// Canonical string to sign
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

	rr := httptest.NewRecorder()
	c.httpServer.Router.ServeHTTP(rr, req)
	body := rr.Body.Bytes()

	if rr.Code >= 400 {
		var serverErr ServerError
		if err := json.Unmarshal(body, &serverErr); err == nil && serverErr.Error != "" {
			return nil, "", &HTTPError{
				StatusCode: rr.Code,
				Message:    serverErr.Error,
			}
		}
		return nil, "", &HTTPError{
			StatusCode: rr.Code,
			Message:    string(body),
		}
	}

	return body, rr.Header().Get("Location"), nil
}

// CreateResource creates a new resource using the given JSON data.
// resourceType specifies the API endpoint, data contains the resource JSON,
// and queryParams are optional query parameters.
// Returns the response body, Location header, and any error that occurred.
func (c *TestHTTPClient) CreateResource(resourceType string, data []byte, queryParams map[string]string) ([]byte, string, error) {
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
func (c *TestHTTPClient) GetResource(resourceType string, resourceName string, queryParams map[string]string, objectType string) ([]byte, error) {
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
func (c *TestHTTPClient) DeleteResource(resourceType string, resourceName string, queryParams map[string]string, objectType string) error {
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
func (c *TestHTTPClient) UpdateResource(resourceType string, data []byte, queryParams map[string]string, objectType string) ([]byte, error) {
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
func (c *TestHTTPClient) UpdateResourceValue(resourcePath string, data []byte, queryParams map[string]string) ([]byte, error) {
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
func (c *TestHTTPClient) ListResources(resourceType string, queryParams map[string]string) ([]byte, error) {
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
func (c *TestHTTPClient) StreamRequest(opts RequestOptions) (io.ReadCloser, error) {
	u, err := url.Parse(c.config.GetServerURL())
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}
	if u.Path == "" {
		u.Path = "/"
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

	rr := httptest.NewRecorder()
	c.httpServer.Router.ServeHTTP(rr, req)

	if rr.Code >= 400 {
		body, _ := io.ReadAll(rr.Body)
		var serverErr ServerError
		if err := json.Unmarshal(body, &serverErr); err == nil && serverErr.Error != "" {
			return nil, &HTTPError{
				StatusCode: rr.Code,
				Message:    serverErr.Error,
			}
		}
		return nil, &HTTPError{
			StatusCode: rr.Code,
			Message:    string(body),
		}
	}

	return io.NopCloser(rr.Body), nil
}
