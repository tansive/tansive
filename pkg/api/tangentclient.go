// Package api provides client functionality for interacting with the Tansive Tangent service.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// Client represents a client for communicating with the Tansive Tangent service via Unix socket.
// It provides methods for invoking skills, retrieving available skills, and accessing session context.
type Client struct {
	httpClient *http.Client
	socketPath string
	config     clientConfig
}

// SkillInvocation represents a request to invoke a skill with specific arguments.
// It contains the session and invocation identifiers along with the skill name and input arguments.
type SkillInvocation struct {
	SessionID    string         `json:"session_id"`
	InvocationID string         `json:"invocation_id"`
	SkillName    string         `json:"skill_name"`
	Args         map[string]any `json:"args"`
}

// SkillResult represents the output of a skill invocation.
// It contains the invocation identifier and the output data from the skill execution.
type SkillResult struct {
	InvocationID string         `json:"invocation_id"`
	Output       map[string]any `json:"output"`
}

// ClientOption is a function type for configuring client behavior.
// It allows setting various client options like timeouts and retry behavior.
type ClientOption func(*clientConfig)

type clientConfig struct {
	dialTimeout time.Duration
	maxRetries  int
	retryDelay  time.Duration
}

// WithDialTimeout sets the dial timeout for the client's HTTP transport.
// This controls how long the client waits when establishing a connection to the Unix socket.
func WithDialTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.dialTimeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retry attempts for failed requests.
// The client will retry failed requests up to this many times before giving up.
func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *clientConfig) {
		c.maxRetries = maxRetries
	}
}

// WithRetryDelay sets the delay between retry attempts for failed requests.
// This controls how long the client waits before retrying a failed request.
func WithRetryDelay(delay time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.retryDelay = delay
	}
}

// NewClient creates a new Client instance for communicating with the Tansive Tangent service.
// It configures an HTTP client that communicates via Unix socket and applies the provided options.
// Returns an error if the socket path is empty or if client creation fails.
func NewClient(socketPath string, opts ...ClientOption) (*Client, error) {
	config := clientConfig{
		dialTimeout: 5 * time.Second,
		maxRetries:  3,
		retryDelay:  100 * time.Millisecond,
	}
	for _, opt := range opts {
		opt(&config)
	}

	if socketPath == "" {
		return nil, fmt.Errorf("socket path is required")
	}

	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.dialTimeout,
	}

	return &Client{
		httpClient: httpClient,
		socketPath: socketPath,
		config:     config,
	}, nil
}

// Close closes the client and releases any associated resources.
// Since this client uses standard net/http, no persistent connections need to be closed.
func (c *Client) Close() error {
	// No persistent connection to close in net/http
	return nil
}

// InvokeSkill executes a skill with the given arguments and returns the result.
// It sends a POST request to the skill-invocations endpoint with retry logic for reliability.
// Returns the skill execution result or an error if the invocation fails.
func (c *Client) InvokeSkill(ctx context.Context, sessionID, invocationID, skillName string, args map[string]any) (*SkillResult, error) {
	invocation := SkillInvocation{
		SessionID:    sessionID,
		InvocationID: invocationID,
		SkillName:    skillName,
		Args:         args,
	}

	body, err := json.Marshal(invocation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal invocation: %w", err)
	}

	var lastErr error
	for i := 0; i < c.config.maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, "POST", "http://unix/skill-invocations", bytes.NewReader(body))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(c.config.retryDelay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("skill invocation failed: %s", string(respBody))
		}

		var result SkillResult
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode result: %w", err)
		}
		return &result, nil
	}

	return nil, fmt.Errorf("failed to invoke skill after %d retries: %w", c.config.maxRetries, lastErr)
}

// GetSkills retrieves the list of available skills for a given session.
// It sends a GET request to the skills endpoint and returns the available LLM tools.
// Returns an error if the request fails after all retry attempts.
func (c *Client) GetSkills(ctx context.Context, sessionID string) ([]LLMTool, error) {
	var lastErr error
	for i := 0; i < c.config.maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://unix/skills?session_id=%s", sessionID), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(c.config.retryDelay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("get skills failed: %s", string(respBody))
		}

		var skills []LLMTool
		if err := json.NewDecoder(resp.Body).Decode(&skills); err != nil {
			return nil, fmt.Errorf("failed to decode skills: %w", err)
		}
		return skills, nil
	}

	return nil, fmt.Errorf("failed to get tools after %d retries: %w", c.config.maxRetries, lastErr)
}

// GetContext retrieves a specific context value for a session and invocation.
// It sends a GET request to the context endpoint to retrieve session-specific data.
// Returns the context value as interface{} or an error if the request fails.
func (c *Client) GetContext(ctx context.Context, sessionID, invocationID, name string) (any, error) {
	var lastErr error
	for i := 0; i < c.config.maxRetries; i++ {
		req, err := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("http://unix/context?session_id=%s&invocation_id=%s&name=%s", sessionID, invocationID, name), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(c.config.retryDelay)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			respBody, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("get context failed: %s", string(respBody))
		}

		var context any
		if err := json.NewDecoder(resp.Body).Decode(&context); err != nil {
			return nil, fmt.Errorf("failed to decode context: %w", err)
		}
		return context, nil
	}

	return nil, fmt.Errorf("failed to get context after %d retries: %w", c.config.maxRetries, lastErr)
}
