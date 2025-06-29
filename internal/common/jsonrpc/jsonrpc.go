// Package jsonrpc provides utilities for handling JSON-RPC 2.0 protocol messages.
// It supports request/response handling, notifications, and error management.
// The package requires valid JSON-serializable types for parameters and results.
package jsonrpc

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tansive/tansive/pkg/types"
)

// Version specifies the JSON-RPC protocol version
const Version = "2.0"

// MethodType represents a JSON-RPC method name
type MethodType string

// Request represents a JSON-RPC 2.0 request or notification.
// ID is optional for notifications.
type Request struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      string            `json:"id,omitempty"`
	Method  MethodType        `json:"method"`
	Params  types.NullableAny `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
// Either Result or Error must be set, but not both.
type Response struct {
	JSONRPC string       `json:"jsonrpc"`
	ID      string       `json:"id"`
	Result  any          `json:"result,omitempty"`
	Error   *ErrorObject `json:"error,omitempty"`
}

// ErrorObject represents a JSON-RPC 2.0 error object.
// Code must be a valid JSON-RPC error code.
// Message should be a short description of the error.
// Data is optional and can contain additional error information.
type ErrorObject struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ConstructRequest creates a JSON-RPC request message.
// Returns an error if params cannot be converted to NullableAny.
func ConstructRequest(id string, method MethodType, params any) ([]byte, error) {
	p, err := types.NullableAnyFrom(params)
	if err != nil {
		return nil, err
	}
	req := Request{
		JSONRPC: Version,
		ID:      id,
		Method:  method,
		Params:  p,
	}
	return json.Marshal(req)
}

// ConstructNotification creates a JSON-RPC notification (no response expected).
// Returns an error if params cannot be converted to NullableAny.
func ConstructNotification(method MethodType, params any) ([]byte, error) {
	p, err := types.NullableAnyFrom(params)
	if err != nil {
		return nil, err
	}
	req := Request{
		JSONRPC: Version,
		Method:  method,
		Params:  p,
	}
	return json.Marshal(req)
}

// ConstructSuccessResponse creates a JSON-RPC response with a result.
// The result must be JSON-serializable.
func ConstructSuccessResponse(id string, result any) ([]byte, error) {
	resp := Response{
		JSONRPC: Version,
		ID:      id,
		Result:  result,
	}
	return json.Marshal(resp)
}

// ConstructErrorResponse creates a JSON-RPC error response.
// The data parameter is optional and must be JSON-serializable if provided.
func ConstructErrorResponse(id string, code int, message string, data any) ([]byte, error) {
	resp := Response{
		JSONRPC: Version,
		ID:      id,
		Error: &ErrorObject{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	return json.Marshal(resp)
}

// ParseRequest unmarshals a JSON-RPC request or notification.
// Returns an error if the request is invalid or missing required fields.
func ParseRequest(data []byte) (*Request, error) {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, err
	}
	if req.JSONRPC != Version || req.Method == "" {
		return nil, errors.New("invalid JSON-RPC request")
	}
	return &req, nil
}

// ParseResponse unmarshals a JSON-RPC response.
// Returns an error if the response is invalid or has both result and error.
func ParseResponse(data []byte) (*Response, error) {
	var resp Response
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	if resp.JSONRPC != Version {
		return nil, errors.New("invalid JSON-RPC response")
	}
	if resp.Result != nil && resp.Error == nil {
		return nil, errors.New("response must have either result or error")
	}
	return &resp, nil
}

// Standard JSON-RPC 2.0 error codes
const (
	ErrCodeParseError        = -32700 // Invalid JSON was received
	ErrCodeInvalidRequest    = -32600 // The JSON sent is not a valid Request object
	ErrCodeMethodNotFound    = -32601 // The method does not exist
	ErrCodeInvalidParams     = -32602 // Invalid method parameter(s)
	ErrCodeInternalError     = -32603 // Internal JSON-RPC error
	ErrCodeConcurrentCommand = -32001 // Command is already running
	ErrCodeBadCommand        = -32002 // Command is invalid
)

// FormatErrorMessage returns a user-friendly error message from an error.
// Returns an empty string if err is nil.
func FormatErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("JSON-RPC error: %s", err.Error())
}
