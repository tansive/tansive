package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/runners"
	"github.com/tansive/tansive/internal/tangent/session/mcpservice"
	"github.com/tansive/tansive/internal/tangent/session/toolgraph"
	"github.com/tansive/tansive/pkg/api"
)

const (
	FilterNoFilter = "no-filter"
	FilterOnly     = "allow-only"
	FilterOverlay  = "overlay"
)

// mcpSession holds state and configuration for an active MCP proxy session, including runner, source, and session-specific identifiers.
type mcpSession struct {
	source       string         // Skill source identifier for filtering and validation
	runner       runners.Runner // Runner instance used to execute MCP tools
	url          string         // URL for the active MCP session
	random       string         // Random session token or identifier
	filter       string         // MCP tool filter annotation for access control
	invocationID string         // Current invocation ID for tracking tool calls
}

// RunMCPProxy executes a skill via the MCP proxy, handling policy checks, input transformation, auditing, and session setup. Returns the session URL or an error.
func (s *session) RunMCPProxy(ctx context.Context, invokerID string, skillName string, inputArgs map[string]any) (string, apperrors.Error) {
	log.Ctx(ctx).Info().Msgf("requested skill: %s", skillName)
	invocationID := uuid.New().String()
	s.mcpSession.invocationID = invocationID
	toolErr := s.callGraph.RegisterCall(toolgraph.CallID(invokerID), toolgraph.ToolName(skillName), toolgraph.CallID(invocationID))
	if toolErr != nil {
		return "", ErrToolGraphError.Msg(toolErr.Error())
	}
	s.invocationIDs[invocationID] = s.viewDef
	s.auditLogInfo.auditLogger.Info().
		Str("event", "skill_start").
		Str("invoker_id", invokerID).
		Str("invocation_id", invocationID).
		Str("skill", skillName).
		Any("input_args", inputArgs).
		Msg("requested skill")
	if invokerID != "" {
		if _, ok := s.invocationIDs[invokerID]; !ok {
			log.Ctx(ctx).Error().Str("invocation_id", invokerID).Msg("invocationID not found")
			return "", ErrInvalidInvocationID.Msg("invocationID not found")
		}
	}

	if err := s.fetchObjects(ctx); err != nil {
		s.logger.Error().Err(err).Msg("unable to fetch objects")
		return "", err
	}

	isAllowed, basis, actions, err := s.ValidateRunPolicy(ctx, invokerID, skillName)
	if err != nil {
		s.logger.Error().Err(err).Msg("unable to validate run policy")
		return "", err
	}
	if !isAllowed {
		msg := fmt.Sprintf("blocked by Tansive policy: view '%s' does not authorize any of required actions - %v - to use this skill", s.context.View, actions)
		s.logger.Error().Str("policy_decision", "true").Msg(msg)
		log.Ctx(ctx).Error().Str("policy_decision", "true").Msg(msg)
		s.auditLogInfo.auditLogger.Error().
			Str("event", "policy_decision").
			Str("decision", "blocked").
			Str("invocation_id", invocationID).
			Str("view", s.context.View).
			Any("basis", basis).
			Str("skill", skillName).
			Any("actions", actions).
			Msg("blocked by policy")
		return "", ErrBlockedByPolicy.Msg(msg)
	}
	msg := fmt.Sprintf("allowed by Tansive policy: view '%s' authorizes actions - %v - to use this skill", s.context.View, actions)
	s.logger.Info().Str("policy_decision", "true").Msg(msg)
	log.Ctx(ctx).Info().Str("policy_decision", "true").Msg(msg)
	s.auditLogInfo.auditLogger.Info().
		Str("event", "policy_decision").
		Str("decision", "allowed").
		Str("invocation_id", invocationID).
		Str("view", s.context.View).
		Any("basis", basis).
		Str("skill", skillName).
		Any("actions", actions).
		Msg("allowed by policy")

	transformApplied, inputArgs, err := s.TransformInputForSkill(ctx, skillName, inputArgs, invocationID)
	if err != nil {
		s.logger.Error().Err(err).Msg("unable to transform input")
		log.Ctx(ctx).Error().Err(err).Msg("unable to transform input")
		s.auditLogInfo.auditLogger.Error().
			Str("event", "skill_input_transformed").
			Str("status", "failed").
			Str("invocation_id", invocationID).
			Err(err).
			Str("skill", skillName).
			Msg("input transformed")
		return "", err
	}
	if transformApplied {
		s.auditLogInfo.auditLogger.Info().
			Str("event", "skill_input_transformed").
			Str("status", "success").
			Str("invocation_id", invocationID).
			Str("skill", skillName).
			Any("input_args", inputArgs).
			Msg("input transformed")
	}

	url, err := s.startMCPProxySession(ctx, invokerID, skillName, inputArgs)

	if err != nil {
		s.logger.Error().Err(err).Msg("unable to run interactive skill")
		s.auditLogInfo.auditLogger.Error().
			Str("event", "skill_end").
			Str("status", "failed").
			Str("invocation_id", invocationID).
			Err(err).
			Str("skill", skillName).
			Msg("skill completed")
	} else {
		s.logger.Info().Str("status", "success").Str("skill", skillName).Msg("skill completed")
		s.auditLogInfo.auditLogger.Info().
			Str("event", "skill_end").
			Str("status", "success").
			Str("invocation_id", invocationID).
			Str("skill", skillName).
			Msg("skill completed")
	}
	return url, err
}

// startMCPProxySession initializes an MCP proxy session for the given skill, validating input and runner, and returns the session URL.
func (s *session) startMCPProxySession(ctx context.Context, invokerID string, skillName string, inputArgs map[string]any) (string, apperrors.Error) {
	_ = invokerID

	skill, err := s.resolveSkill(skillName)
	if err != nil {
		return "", err
	}

	if err := skill.ValidateInput(inputArgs); err != nil {
		return "", err
	}

	mcpfilter, ok := skill.Annotations["mcp:tools"]
	if !ok {
		return "", ErrSkillNotMCP.Msg("skill is not an MCP server")
	}

	runnerCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	runner, err := s.getRunner(runnerCtx, skillName)
	if err != nil {
		return "", err
	}
	s.mcpSession.runner = runner
	s.mcpSession.source = skill.Source

	url, random, err := mcpservice.NewMCPSession(ctx, s)
	if err != nil {
		return "", err
	}
	s.mcpSession.url = url
	s.mcpSession.random = random
	s.mcpSession.filter = mcpfilter
	return url, nil
}

// MCPFilterTools filters the available MCP tools based on the current skill's MCP filter annotation and skill set.
func (s *session) MCPFilterTools(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	skill, err := s.resolveSkill(s.context.Skill)
	if err != nil {
		return tools
	}

	mcpfilter, ok := skill.Annotations["mcp:tools"]
	if !ok {
		return tools
	}
	if mcpfilter == FilterNoFilter || mcpfilter == FilterOverlay {
		return tools
	}

	skills := s.skillSet.GetAllSkills()
	filteredTools := []mcp.Tool{}
	for _, tool := range tools {
		for _, skill := range skills {
			if skill.Source == s.mcpSession.source && skill.Name == tool.Name {
				filteredTools = append(filteredTools, tool)
			}
		}
	}

	return filteredTools
}

// MCPListTools retrieves and returns the list of available MCP tools from the current session's runner, parsing their annotations.
func (s *session) MCPListTools(ctx context.Context) ([]mcp.Tool, error) {
	tools, err := s.mcpSession.runner.FetchTools(ctx)
	if err != nil {
		return nil, err
	}
	retTools := []mcp.Tool{}

	for _, tool := range tools {
		tAnnotations := mcp.ToolAnnotation{}
		if err := json.Unmarshal(tool.Annotations, &tAnnotations); err != nil {
			return nil, err
		}
		retTools = append(retTools, mcp.Tool{
			Name:           tool.Name,
			Description:    tool.Description,
			RawInputSchema: tool.InputSchema,
			Annotations:    tAnnotations,
		})
	}
	return retTools, nil
}

// MCPCallTool invokes a specific MCP tool, performing policy checks, input transformation, auditing, and error handling. Returns the tool's result or an error.
func (s *session) MCPCallTool(ctx context.Context, tool mcp.Tool, params mcp.CallToolParams) (*mcp.CallToolResult, error) {
	inputArgs, ok := params.Arguments.(map[string]any)
	if !ok {
		return nil, ErrInvalidInput.Msg("invalid input arguments")
	}
	invokerID := s.mcpSession.invocationID
	invocationID := uuid.New().String()
	toolErr := s.callGraph.RegisterCall(toolgraph.CallID(invokerID), toolgraph.ToolName(tool.Name), toolgraph.CallID(invocationID))
	if toolErr != nil {
		return nil, ErrToolGraphError.Msg(toolErr.Error())
	}
	s.invocationIDs[invocationID] = s.viewDef

	s.auditLogInfo.auditLogger.Info().
		Str("event", "skill_start").
		Str("invoker_id", invokerID).
		Str("invocation_id", invocationID).
		Str("skill", tool.Name).
		Any("input_args", inputArgs).
		Msg("requested skill")

	if s.mcpSession.filter != FilterNoFilter {
		skill, err := s.resolveSkill(tool.Name)
		if err != nil && s.mcpSession.filter == FilterOnly {
			return nil, err
		}
		if err == nil {
			if skill.Source != s.mcpSession.source {
				return nil, ErrSkillNotMCP.Msg("skill is not from the same MCP server")
			}

			isAllowed, basis, actions, err := s.ValidateRunPolicy(ctx, "", skill.Name)
			if err != nil {
				s.logger.Error().Err(err).Msg("unable to validate run policy")
				return nil, err
			}

			if !isAllowed {
				msg := fmt.Sprintf("blocked by Tansive policy: view '%s' does not authorize any of required actions - %v - to use this skill", s.context.View, actions)
				s.logger.Error().Str("policy_decision", "true").Msg(msg)
				log.Ctx(ctx).Error().Str("policy_decision", "true").Msg(msg)
				s.auditLogInfo.auditLogger.Error().
					Str("event", "policy_decision").
					Str("decision", "blocked").
					Str("invoker_id", invokerID).
					Str("invocation_id", invocationID).
					Str("view", s.context.View).
					Any("basis", basis).
					Str("skill", skill.Name).
					Any("actions", actions).
					Msg("blocked by policy")

				result := &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: msg,
						},
					},
				}
				return result, nil
			}

			s.auditLogInfo.auditLogger.Info().
				Str("event", "policy_decision").
				Str("decision", "allowed").
				Str("invoker_id", invokerID).
				Str("invocation_id", invocationID).
				Str("view", s.context.View).
				Any("basis", basis).
				Str("skill", skill.Name).
				Any("actions", actions).
				Msg("allowed by policy")

			var transformApplied bool
			transformApplied, inputArgs, err = s.TransformInputForSkill(ctx, skill.Name, inputArgs, invocationID)
			if err != nil {
				s.logger.Error().Err(err).Msg("unable to transform input")
				log.Ctx(ctx).Error().Err(err).Msg("unable to transform input")
				s.auditLogInfo.auditLogger.Error().
					Str("event", "skill_input_transformed").
					Str("status", "failed").
					Str("invocation_id", invocationID).
					Err(err).
					Str("skill", skill.Name).
					Msg("input transformed")
				result := &mcp.CallToolResult{
					IsError: true,
					Content: []mcp.Content{
						mcp.TextContent{
							Type: "text",
							Text: "Blocked by Tansive, policy-driven, secure AI Agents: " + err.Error(),
						},
					},
				}
				return result, nil
			}

			if transformApplied {
				s.auditLogInfo.auditLogger.Info().
					Str("event", "skill_input_transformed").
					Str("status", "success").
					Str("invocation_id", invocationID).
					Str("skill", skill.Name).
					Any("input_args", inputArgs).
					Msg("input transformed")
			}
		} else {
			s.auditLogInfo.auditLogger.Info().
				Str("event", "policy_decision").
				Str("decision", "allowed").
				Str("invoker_id", invokerID).
				Str("invocation_id", invocationID).
				Str("view", s.context.View).
				Any("basis", "overlay, no policy filter").
				Str("skill", tool.Name).
				Any("actions", "none").
				Msg("allowed by policy")
		}
	} else {
		s.auditLogInfo.auditLogger.Info().
			Str("event", "policy_decision").
			Str("decision", "allowed").
			Str("invoker_id", invokerID).
			Str("invocation_id", invocationID).
			Str("view", s.context.View).
			Any("basis", "no-filter, no policy filter").
			Str("skill", tool.Name).
			Any("actions", "none").
			Msg("allowed by policy")
	}

	result, err := s.mcpSession.runner.RunMCP(ctx, &api.SkillInputArgs{
		InvocationID: s.mcpSession.invocationID,
		SkillName:    tool.Name,
		InputArgs:    inputArgs,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to call tool")
		s.auditLogInfo.auditLogger.Error().
			Str("event", "skill_end").
			Str("status", "failed").
			Str("invocation_id", s.mcpSession.invocationID).
			Err(err).
			Str("skill", tool.Name).
			Msg("skill completed")
		return nil, err
	}

	s.auditLogInfo.auditLogger.Info().
		Str("event", "skill_end").
		Str("status", "success").
		Str("invocation_id", s.mcpSession.invocationID).
		Str("skill", tool.Name).
		Msg("skill completed")

	return result, nil
}
