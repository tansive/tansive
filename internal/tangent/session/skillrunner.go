package session

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/session/skillservice"
	"github.com/tansive/tansive/internal/tangent/tangentcommon"
	"github.com/tansive/tansive/pkg/api"
)

// CreateSkillService creates and starts a skill service for local skill execution.
// Returns the skill service instance and any error encountered during creation.
// The service runs on a Unix domain socket for local communication.
func CreateSkillService() (*skillservice.SkillService, apperrors.Error) {
	// Create and register the skill service
	skillService := skillservice.NewSkillService(&skillRunner{})

	go func() {
		err := skillService.StartServer()
		if err != nil {
			log.Fatal().Err(err).Msg("failed to start skill service server")
		}
	}()

	return skillService, nil
}

// skillRunner implements the SkillManager interface for session-based skill execution.
// Provides skill listing, context management, and skill execution capabilities.
type skillRunner struct{}

// GetSkills retrieves available skills for a session as LLM tools.
// Returns the skills list and any error encountered during retrieval.
func (s *skillRunner) GetSkills(ctx context.Context, sessionID string) ([]api.LLMTool, apperrors.Error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionError.Msg("invalid sessionID")
	}
	session, err := ActiveSessionManager().GetSession(sessionUUID)
	if err != nil {
		return nil, ErrSessionError.Msg(err.Error())
	}
	return session.getSkillsAsLLMTools()
}

// GetContext retrieves a context value for a session and invocation.
// Returns the context value and any error encountered during retrieval.
func (s *skillRunner) GetContext(ctx context.Context, sessionID, invocationID, name string) (any, apperrors.Error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, ErrSessionError.Msg("invalid sessionID")
	}
	session, err := ActiveSessionManager().GetSession(sessionUUID)
	if err != nil {
		return nil, ErrSessionError.Msg(err.Error())
	}
	return session.getContext(invocationID, name)
}

// Run executes a skill with the given parameters.
// Validates parameters, retrieves the session, and executes the skill.
// Returns the skill output and any error encountered during execution.
func (s *skillRunner) Run(ctx context.Context, params *tangentcommon.RunParams) (map[string]any, apperrors.Error) {
	if params == nil {
		return nil, ErrSessionError.Msg("params is nil")
	}

	if params.SessionID == "" {
		return nil, ErrSessionError.Msg("sessionID is empty")
	}

	if params.SkillName == "" {
		return nil, ErrSessionError.Msg("skillName is empty")
	}

	if params.InvocationID == "" {
		return nil, ErrSessionError.Msg("invocationID is empty")
	}

	// Get the session
	sessionID, err := uuid.Parse(params.SessionID)
	if err != nil {
		return nil, ErrSessionError.Msg(err.Error())
	}

	session, err := ActiveSessionManager().GetSession(sessionID)
	if err != nil {
		return nil, ErrSessionError.Msg(err.Error())
	}

	// Create writers to capture command outputs
	outWriter := tangentcommon.NewBufferedWriter()
	errWriter := tangentcommon.NewBufferedWriter()

	// Run the skill
	runCtx := session.getLogger(TopicSessionLog).With().Str("actor", params.SkillName).Str("session_id", session.id.String()).Str("skill", session.context.Skill).Logger().WithContext(ctx)
	apperr := session.Run(runCtx, params.InvocationID, params.SkillName, params.InputArgs, &tangentcommon.IOWriters{
		Out: outWriter,
		Err: errWriter,
	})

	return processOutput(outWriter, errWriter, apperr)
}

// processOutput processes the output from skill execution.
// Formats output based on content type and error conditions.
// Returns the processed response and any error encountered during processing.
func processOutput(outWriter *tangentcommon.BufferedWriter, errWriter *tangentcommon.BufferedWriter, err apperrors.Error) (map[string]any, apperrors.Error) {
	response := make(map[string]any)

	if err != nil {
		if errors.Is(err, ErrBlockedByPolicy) {
			response["error"] = err.Error() + " Please contact the administrator of your Tansive system to request access."
		} else {
			response["error"] = err.Error()
		}
		b := strings.Builder{}
		b.WriteString(outWriter.String())
		b.WriteString("\n")
		b.WriteString(errWriter.String())
		if b.Len() > 2 {
			response["content"] = map[string]any{
				"type":  "text",
				"value": b.String(),
			}
		}
		return response, nil
	}

	output := outWriter.Bytes()
	var parsed any
	if json.Unmarshal(output, &parsed) == nil {
		response["content"] = map[string]any{
			"type":  detectJSONType(parsed),
			"value": parsed,
		}
	} else {
		// Not JSON, treat as plaintext
		response["content"] = map[string]any{
			"type":  "text",
			"value": outWriter.String(),
		}
	}

	return response, nil
}

// detectJSONType determines the JSON type of a parsed value.
// Returns a string representation of the JSON type for content type detection.
func detectJSONType(v any) string {
	switch v := v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case nil:
		return "null"
	default:
		_ = v
		return "unknown"
	}
}
