// Package skillservice provides a local HTTP service for skill execution.
// It runs on Unix domain sockets and provides endpoints for skill invocation, skill listing, and context management.
// The package requires a valid skill manager and supports graceful shutdown with signal handling.
package skillservice

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/httpx"
	"github.com/tansive/tansive/internal/tangent/config"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/pkg/api"
)

// SkillService provides a local HTTP server for skill execution.
// Manages skill invocation, skill listing, and context operations via Unix domain socket.
type SkillService struct {
	skillManager tangentcommon.SkillManager
	Router       *chi.Mux
	server       *http.Server
	socketPath   string
	mu           sync.Mutex
}

// NewSkillService creates a new skill service with the given skill manager.
// Returns the service instance and any error encountered during creation.
// SkillManager must not be nil for proper operation.
func NewSkillService(skillManager tangentcommon.SkillManager) *SkillService {
	if skillManager == nil {
		log.Error().Msg("SkillManager is nil")
		return nil
	}
	return &SkillService{
		skillManager: skillManager,
		Router:       chi.NewRouter(),
	}
}

// handleInvokeSkill processes skill invocation requests.
// Returns the skill execution result and any error encountered during processing.
func (s *SkillService) handleInvokeSkill(r *http.Request) (*httpx.Response, error) {
	var req api.SkillInvocation
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, ErrInvalidRequest.Msg(err.Error())
	}

	resp, err := s.skillManager.Run(r.Context(), &tangentcommon.RunParams{
		SessionID:    req.SessionID,
		InvocationID: req.InvocationID,
		SkillName:    req.SkillName,
		InputArgs:    req.Args,
	})

	if err != nil {
		return nil, ErrSkillServiceError.Msg(err.Error())
	}

	result := api.SkillResult{
		InvocationID: req.InvocationID,
		Output:       resp,
	}

	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   result,
	}, nil
}

// handleGetSkills retrieves available skills for a session.
// Returns the list of skills as LLM tools and any error encountered during retrieval.
func (s *SkillService) handleGetSkills(r *http.Request) (*httpx.Response, error) {
	tools, err := s.skillManager.GetSkills(r.Context(), r.URL.Query().Get("session_id"))
	if err != nil {
		return nil, ErrSkillServiceError.Msg(err.Error())
	}
	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   tools,
	}, nil
}

// handleGetContext retrieves context values for a session and invocation.
// Returns the context value and any error encountered during retrieval.
func (s *SkillService) handleGetContext(r *http.Request) (*httpx.Response, error) {
	sessionID := r.URL.Query().Get("session_id")
	invocationID := r.URL.Query().Get("invocation_id")
	name := r.URL.Query().Get("name")
	context, err := s.skillManager.GetContext(r.Context(), sessionID, invocationID, name)
	if err != nil {
		return nil, ErrSkillServiceError.Msg(err.Error())
	}
	return &httpx.Response{
		StatusCode: http.StatusOK,
		Response:   context,
	}, nil
}

// MountHandlers registers HTTP handlers for skill service endpoints.
// Sets up routes for skill invocation, skill listing, and context operations.
func (s *SkillService) MountHandlers() {
	s.Router.Post("/skill-invocations", httpx.WrapHttpRsp(s.handleInvokeSkill))
	s.Router.Get("/skills", httpx.WrapHttpRsp(s.handleGetSkills))
	s.Router.Get("/context", httpx.WrapHttpRsp(s.handleGetContext))
}

// StartServer starts the skill service on a Unix domain socket.
// Handles graceful shutdown with signal handling and returns any error encountered during startup.
func (s *SkillService) StartServer() error {
	socketPath, err := config.GetSocketPath()
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.socketPath = socketPath
	s.mu.Unlock()

	// Remove existing socket if it exists
	if _, err := os.Stat(socketPath); err == nil {
		if err := os.Remove(socketPath); err != nil {
			return fmt.Errorf("failed to remove existing socket: %w", err)
		}
	}

	socketDir := filepath.Dir(socketPath)
	if err := os.MkdirAll(socketDir, 0700); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket: %w", err)
	}
	if err := os.Chmod(socketPath, 0600); err != nil {
		log.Warn().Err(err).Msg("failed to chmod socket")
	}

	s.MountHandlers()
	srv := &http.Server{
		Handler:           s.Router,
		ReadHeaderTimeout: 5 * time.Second, // Keep this for initial connection setup
	}
	s.mu.Lock()
	s.server = srv
	s.mu.Unlock()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Info().Str("signal", sig.String()).Msg("received shutdown signal")
		s.StopServer()
	}()

	log.Info().Str("socket", socketPath).Msg("local service started")

	// Start server and handle shutdown
	err = srv.Serve(listener)
	if err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}

// StopServer gracefully shuts down the skill service.
// Closes the HTTP server and removes the Unix domain socket file.
func (s *SkillService) StopServer() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.server.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("error shutting down server")
		}
		s.server = nil
	}

	if s.socketPath != "" {
		if _, err := os.Stat(s.socketPath); err == nil {
			if err := os.Remove(s.socketPath); err != nil {
				log.Error().Err(err).Msg("error removing socket file")
			}
		}
		s.socketPath = ""
	}
}
