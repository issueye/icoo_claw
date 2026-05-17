package service

import (
	"context"
	"fmt"
	"time"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type SessionStore interface {
	CreateSession(ctx context.Context, req client.CreateSessionRequest) error
	ListMessages(ctx context.Context, sessionID string) ([]dto.SessionMessage, error)
}

type AgentRunner interface {
	Run(ctx context.Context, baseURL string, req client.RunRequest) (*client.RunResponse, error)
	Stream(ctx context.Context, baseURL string, req client.RunRequest) (<-chan client.StreamEvent, error)
}

type ChatService struct {
	conversations repository.ConversationRepository
	agents        repository.AgentRepository
	instances     repository.AgentInstanceRepository
	sessionStore  SessionStore
	claw          AgentRunner
}

func NewChatService(conversations repository.ConversationRepository, agents repository.AgentRepository, instances repository.AgentInstanceRepository, sessionStore SessionStore, claw AgentRunner) *ChatService {
	return &ChatService{
		conversations: conversations,
		agents:        agents,
		instances:     instances,
		sessionStore:  sessionStore,
		claw:          claw,
	}
}

func (s *ChatService) CreateConversation(ctx context.Context, req dto.CreateConversationRequest) (*dto.Conversation, error) {
	if _, err := s.agents.Get(ctx, req.AgentID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	conversation := model.Conversation{
		ID:        "conv_" + randomID(),
		SessionID: "sess_" + randomID(),
		AgentID:   req.AgentID,
		UserID:    req.UserID,
		Title:     defaultString(req.Title, "New conversation"),
		Status:    "active",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.sessionStore.CreateSession(ctx, client.CreateSessionRequest{
		SessionID: conversation.SessionID,
		UserID:    conversation.UserID,
		AgentID:   conversation.AgentID,
		Title:     conversation.Title,
		Metadata:  map[string]any{"conversation_id": conversation.ID},
	}); err != nil {
		return nil, err
	}
	if err := s.conversations.Create(ctx, conversation); err != nil {
		return nil, err
	}
	return toConversationDTO(conversation), nil
}

func (s *ChatService) ListConversations(ctx context.Context) ([]dto.Conversation, error) {
	conversations, err := s.conversations.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]dto.Conversation, len(conversations))
	for i, conversation := range conversations {
		out[i] = *toConversationDTO(conversation)
	}
	return out, nil
}

func (s *ChatService) ListMessages(ctx context.Context, conversationID string) ([]dto.SessionMessage, error) {
	conversation, err := s.conversations.Get(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	return s.sessionStore.ListMessages(ctx, conversation.SessionID)
}

func (s *ChatService) SendMessage(ctx context.Context, conversationID string, req dto.SendMessageRequest) (*dto.ChatResponse, error) {
	conversation, agent, instance, err := s.prepareRun(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if err := s.markInflight(ctx, instance, 1); err != nil {
		return nil, err
	}
	defer func() { _ = s.markInflight(context.Background(), instance, -1) }()

	resp, err := s.claw.Run(ctx, instance.BaseURL, client.RunRequest{
		SessionID:     conversation.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		Agent:         agentProfileMap(*agent),
		ToolWhitelist: parseStringSlice(agent.ToolWhitelistJSON),
		Metadata:      req.Metadata,
	})
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	conversation.LastMessageAt = &now
	conversation.UpdatedAt = now
	if err := s.conversations.Update(ctx, *conversation); err != nil {
		return nil, err
	}

	return &dto.ChatResponse{
		ConversationID: conversation.ID,
		SessionID:      resp.SessionID,
		RequestID:      resp.RequestID,
		Output:         resp.Output,
		StopReason:     resp.StopReason,
	}, nil
}

func (s *ChatService) StreamMessage(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
	conversation, agent, instance, err := s.prepareRun(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.markInflight(ctx, instance, 1); err != nil {
		return nil, err
	}

	events, err := s.claw.Stream(ctx, instance.BaseURL, client.RunRequest{
		SessionID:     conversation.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		Agent:         agentProfileMap(*agent),
		ToolWhitelist: parseStringSlice(agent.ToolWhitelistJSON),
		Metadata:      req.Metadata,
	})
	if err != nil {
		_ = s.markInflight(context.Background(), instance, -1)
		return nil, err
	}

	out := make(chan client.StreamEvent, 128)
	go func() {
		defer close(out)
		defer func() { _ = s.markInflight(context.Background(), instance, -1) }()
		defer func() {
			now := time.Now().UTC()
			conversation.LastMessageAt = &now
			conversation.UpdatedAt = now
			_ = s.conversations.Update(context.Background(), *conversation)
		}()
		for event := range events {
			out <- event
		}
	}()
	return out, nil
}

func (s *ChatService) DeleteConversation(ctx context.Context, id string) error {
	return s.conversations.Delete(ctx, id)
}

func (s *ChatService) prepareRun(ctx context.Context, conversationID string) (*model.Conversation, *model.AgentProfile, *model.AgentInstance, error) {
	conversation, err := s.conversations.Get(ctx, conversationID)
	if err != nil {
		return nil, nil, nil, err
	}
	agent, err := s.agents.Get(ctx, conversation.AgentID)
	if err != nil {
		return nil, nil, nil, err
	}
	instance, err := s.selectInstance(ctx, conversation.AgentID)
	if err != nil {
		return nil, nil, nil, err
	}
	return conversation, agent, instance, nil
}

func (s *ChatService) selectInstance(ctx context.Context, agentID string) (*model.AgentInstance, error) {
	instances, err := s.instances.List(ctx)
	if err != nil {
		return nil, err
	}
	var selected *model.AgentInstance
	for i := range instances {
		instance := instances[i]
		if instance.AgentID != agentID || instance.Status != "ready" {
			continue
		}
		if selected == nil || instance.Inflight < selected.Inflight {
			selected = &instance
		}
	}
	if selected == nil {
		return nil, fmt.Errorf("no ready agent instance for agent %s", agentID)
	}
	return selected, nil
}

func (s *ChatService) markInflight(ctx context.Context, instance *model.AgentInstance, delta int) error {
	current, err := s.instances.Get(ctx, instance.ID)
	if err != nil {
		return err
	}
	current.Inflight += delta
	if current.Inflight < 0 {
		current.Inflight = 0
	}
	current.UpdatedAt = time.Now().UTC()
	return s.instances.Update(ctx, *current)
}

func toConversationDTO(conversation model.Conversation) *dto.Conversation {
	return &dto.Conversation{
		ID:            conversation.ID,
		SessionID:     conversation.SessionID,
		AgentID:       conversation.AgentID,
		UserID:        conversation.UserID,
		Title:         conversation.Title,
		Status:        conversation.Status,
		LastMessageAt: conversation.LastMessageAt,
		CreatedAt:     conversation.CreatedAt,
		UpdatedAt:     conversation.UpdatedAt,
	}
}

func agentProfileMap(agent model.AgentProfile) map[string]any {
	return map[string]any{
		"model_provider":        agent.ModelProvider,
		"model_name":            agent.ModelName,
		"base_url":              agent.BaseURL,
		"system_prompt":         agent.SystemPrompt,
		"max_iterations":        agent.MaxIterations,
		"enabled_builtin_tools": parseStringSlice(agent.ToolWhitelistJSON),
		"mcp_servers":           parseStringSlice(agent.MCPServerIDsJSON),
	}
}
