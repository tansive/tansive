// Package httpclient provides a configurable HTTP client for making requests to REST APIs.
// It supports authentication via API keys and tokens, handles common HTTP operations,
// and provides error handling for server responses. The package requires a Configurator
// implementation for server configuration and authentication details.
package httpclient

import (
	"io"
)

// HTTPClientInterface defines the interface for HTTP client implementations.
// It provides a common set of methods for making HTTP requests and managing resources.
// Implementations must handle authentication, request building, and response processing.
type HTTPClientInterface interface {
	// DoRequest makes an HTTP request with the given options.
	// Returns the response body, Location header (if present), and any error that occurred.
	DoRequest(opts RequestOptions) ([]byte, string, error)

	// StreamRequest makes an HTTP request with the given options and streams the response.
	// Returns an io.ReadCloser for streaming large responses. The caller is responsible
	// for closing the returned reader.
	StreamRequest(opts RequestOptions) (io.ReadCloser, error)

	// CreateResource creates a new resource using the given JSON data.
	// resourceType specifies the API endpoint, data contains the resource JSON,
	// and queryParams are optional query parameters.
	// Returns the response body, Location header, and any error that occurred.
	CreateResource(resourceType string, data []byte, queryParams map[string]string) ([]byte, string, error)

	// GetResource retrieves a resource using the given resource name.
	// resourceType specifies the API endpoint, resourceName identifies the resource,
	// queryParams are optional query parameters, and objectType is an optional type qualifier.
	// Returns the response body and any error that occurred.
	GetResource(resourceType string, resourceName string, queryParams map[string]string, objectType string) ([]byte, error)

	// DeleteResource deletes a resource using the given resource name.
	// resourceType specifies the API endpoint, resourceName identifies the resource,
	// queryParams are optional query parameters, and objectType is an optional type qualifier.
	// Returns any error that occurred during the deletion.
	DeleteResource(resourceType string, resourceName string, queryParams map[string]string, objectType string) error

	// UpdateResource updates an existing resource using the given JSON data.
	// resourceType specifies the API endpoint, data contains the updated resource JSON,
	// queryParams are optional query parameters, and objectType is an optional type qualifier.
	// The data must contain a metadata.name field.
	// Returns the response body and any error that occurred.
	UpdateResource(resourceType string, data []byte, queryParams map[string]string, objectType string) ([]byte, error)

	// UpdateResourceValue updates a specific resource value at the given path.
	// resourcePath specifies the full API endpoint path, data contains the update JSON,
	// and queryParams are optional query parameters.
	// Returns the response body and any error that occurred.
	UpdateResourceValue(resourcePath string, data []byte, queryParams map[string]string) ([]byte, error)

	// ListResources lists resources of a specific type.
	// resourceType specifies the API endpoint, and queryParams are optional query parameters.
	// Returns the response body and any error that occurred.
	ListResources(resourceType string, queryParams map[string]string) ([]byte, error)
}

// Verify that the HTTPClient and TestHTTPClient implement the HTTPClientInterface.
// This is a compile-time check to ensure both implementations satisfy the interface.
var _ HTTPClientInterface = &HTTPClient{}
var _ HTTPClientInterface = &TestHTTPClient{}
