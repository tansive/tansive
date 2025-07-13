package session

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/apperrors"
	"github.com/tansive/tansive/internal/common/uuid"
	"github.com/tansive/tansive/internal/tangent/runners"
	"github.com/tansive/tansive/internal/tangent/session/mcpservice"
	"github.com/tansive/tansive/pkg/api"
)

type mcpSession struct {
	source string
	runner runners.Runner
	url    string
	random string
}

func (s *session) RunMCPProxy(ctx context.Context, invokerID string, skillName string, inputArgs map[string]any) (string, apperrors.Error) {
	log.Ctx(ctx).Info().Msgf("requested skill: %s", skillName)
	invocationID := uuid.New().String()
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

	transformApplied, inputArgs, err := s.TransformInputForSkill(ctx, skillName, inputArgs)
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

	// We only support interactive skills for now
	url, err := s.startMCPSession(ctx, invokerID, skillName, inputArgs)

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

func (s *session) startMCPSession(ctx context.Context, invokerID string, skillName string, inputArgs map[string]any) (string, apperrors.Error) {
	_ = invokerID

	skill, err := s.resolveSkill(skillName)
	if err != nil {
		return "", err
	}

	if err := skill.ValidateInput(inputArgs); err != nil {
		return "", err
	}

	_, ok := skill.Annotations["mcp:tools"]
	if !ok {
		return "", ErrSkillNotMCP.Msg("skill is not an MCP server")
	}

	runner, err := s.getRunner(ctx, skillName)
	if err != nil {
		return "", err
	}
	s.mcpSession = mcpSession{
		source: skill.Source,
		runner: runner,
	}

	url, random, err := mcpservice.NewMCPSession(ctx, s)
	if err != nil {
		return "", err
	}
	s.mcpSession.url = url
	s.mcpSession.random = random

	return url, nil
}

func (s *session) MCPFilterTools(ctx context.Context, tools []mcp.Tool) []mcp.Tool {
	skill, err := s.resolveSkill(s.context.Skill)
	if err != nil {
		return tools
	}

	mcpfilter, ok := skill.Annotations["mcp:tools"]
	if !ok {
		return tools
	}
	if mcpfilter == "no-filter" {
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

func (s *session) MCPCallTool(ctx context.Context, tool mcp.Tool, params mcp.CallToolParams) (*mcp.CallToolResult, error) {
	skill, err := s.resolveSkill(tool.Name)
	if err != nil {
		return nil, err
	}

	if skill.Source != s.mcpSession.source {
		return nil, ErrSkillNotMCP.Msg("skill is not from the same MCP server")
	}

	inputArgs, ok := params.Arguments.(map[string]any)
	if !ok {
		return nil, ErrInvalidInput.Msg("invalid input arguments")
	}

	transformApplied, inputArgs, err := s.TransformInputForSkill(ctx, skill.Name, inputArgs)
	if err != nil {
		return nil, err
	}

	_ = transformApplied

	result, err := s.mcpSession.runner.RunMCP(ctx, &api.SkillInputArgs{
		InvocationID: uuid.New().String(),
		SkillName:    tool.Name,
		InputArgs:    inputArgs,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("unable to call tool")
		return nil, err
	}

	return result, nil
}
