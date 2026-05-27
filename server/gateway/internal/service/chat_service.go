package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/model"
	"icoo_claw/server/gateway/internal/repository"
)

type SessionBackend interface {
	CreateSession(ctx context.Context, req SessionCreateRequest) error
	ListMessages(ctx context.Context, sessionID string) ([]dto.SessionMessage, error)
}

type SessionCreateRequest struct {
	SessionID string
	UserID    string
	AgentID   string
	Title     string
	Metadata  map[string]any
}

type AgentRunner interface {
	Run(ctx context.Context, baseURL string, req client.RunRequest) (*client.RunResponse, error)
	Stream(ctx context.Context, baseURL string, req client.RunRequest) (<-chan client.StreamEvent, error)
}

type ChatService struct {
	conversations repository.ConversationRepository
	agents        repository.AgentRepository
	providers     repository.ProviderRepository
	skills        *SkillService
	router        RouterPolicy
	sessions      SessionBackend
	claw          AgentRunner
}

func NewChatService(conversations repository.ConversationRepository, agents repository.AgentRepository, providers repository.ProviderRepository, router RouterPolicy, sessions SessionBackend, claw AgentRunner, skills ...*SkillService) *ChatService {
	var skillService *SkillService
	if len(skills) > 0 {
		skillService = skills[0]
	}
	return &ChatService{
		conversations: conversations,
		agents:        agents,
		providers:     providers,
		skills:        skillService,
		router:        router,
		sessions:      sessions,
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
	if err := s.sessions.CreateSession(ctx, SessionCreateRequest{
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
	return s.sessions.ListMessages(ctx, conversation.SessionID)
}

func (s *ChatService) SendMessage(ctx context.Context, conversationID string, req dto.SendMessageRequest) (*dto.ChatResponse, error) {
	conversation, agent, provider, instance, err := s.prepareRun(ctx, conversationID)
	if err != nil {
		return nil, err
	}

	if err := s.router.MarkInflight(ctx, instance.ID, 1); err != nil {
		return nil, err
	}
	defer func() { _ = s.router.MarkInflight(context.Background(), instance.ID, -1) }()
	if err := s.markConversationStatus(ctx, conversation, "running"); err != nil {
		return nil, err
	}
	defer func() {
		_ = s.markConversationStatus(context.Background(), conversation, "active")
	}()

	agentPayload, err := s.agentProfilePayload(ctx, *agent, provider, req.Metadata)
	if err != nil {
		return nil, err
	}
	stream, err := s.claw.Stream(ctx, instance.BaseURL, client.RunRequest{
		SessionID:     conversation.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		Agent:         agentPayload,
		ToolWhitelist: parseStringSlice(agent.ToolWhitelistJSON),
		ForceSkills:   cleanStringSlice(req.ForceSkills),
		Metadata:      req.Metadata,
	})
	if err != nil {
		return nil, err
	}
	output, stopReason, sessionID, requestID, err := collectClawStream(stream, conversation.SessionID, req.RequestID)
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
		SessionID:      sessionID,
		RequestID:      requestID,
		Output:         output,
		StopReason:     stopReason,
	}, nil
}

func (s *ChatService) StreamMessage(ctx context.Context, conversationID string, req dto.SendMessageRequest) (<-chan client.StreamEvent, error) {
	conversation, agent, provider, instance, err := s.prepareRun(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.router.MarkInflight(ctx, instance.ID, 1); err != nil {
		return nil, err
	}
	if err := s.markConversationStatus(ctx, conversation, "running"); err != nil {
		_ = s.router.MarkInflight(context.Background(), instance.ID, -1)
		return nil, err
	}

	agentPayload, err := s.agentProfilePayload(ctx, *agent, provider, req.Metadata)
	if err != nil {
		_ = s.router.MarkInflight(context.Background(), instance.ID, -1)
		return nil, err
	}
	events, err := s.claw.Stream(ctx, instance.BaseURL, client.RunRequest{
		SessionID:     conversation.SessionID,
		RequestID:     req.RequestID,
		Prompt:        req.Prompt,
		Agent:         agentPayload,
		ToolWhitelist: parseStringSlice(agent.ToolWhitelistJSON),
		ForceSkills:   cleanStringSlice(req.ForceSkills),
		Metadata:      req.Metadata,
	})
	if err != nil {
		_ = s.router.MarkInflight(context.Background(), instance.ID, -1)
		return nil, err
	}

	out := make(chan client.StreamEvent, 128)
	go func() {
		defer close(out)
		defer func() { _ = s.router.MarkInflight(context.Background(), instance.ID, -1) }()
		defer func() {
			_ = s.markConversationStatus(context.Background(), conversation, "active")
		}()
		defer func() {
			now := time.Now().UTC()
			conversation.LastMessageAt = &now
			conversation.UpdatedAt = now
			_ = s.conversations.Update(context.Background(), *conversation)
		}()
		_ = client.DispatchStreamEvents(events, client.StreamEventHandlerFunc{
			OnUpdate: func(event client.StreamEvent) error {
				out <- event
				return nil
			},
			OnCompleted: func(event client.StreamEvent) error {
				out <- event
				return nil
			},
			OnError: func(event client.StreamEvent) error {
				out <- event
				return nil
			},
			OnUnhandled: func(event client.StreamEvent) error {
				out <- event
				return nil
			},
		})
	}()
	return out, nil
}

func (s *ChatService) markConversationStatus(ctx context.Context, conversation *model.Conversation, status string) error {
	if conversation == nil {
		return nil
	}
	conversation.Status = status
	now := time.Now().UTC()
	conversation.UpdatedAt = now
	return s.conversations.Update(ctx, *conversation)
}

func (s *ChatService) DeleteConversation(ctx context.Context, id string) error {
	return s.conversations.Delete(ctx, id)
}

func collectClawStream(events <-chan client.StreamEvent, fallbackSessionID, fallbackRequestID string) (string, string, string, string, error) {
	var output strings.Builder
	stopReason := "stream_closed"
	sessionID := fallbackSessionID
	requestID := fallbackRequestID
	completed := false

	handler := client.StreamEventHandlerFunc{
		OnUpdate: func(event client.StreamEvent) error {
			if event.Update != nil && event.Update.SessionUpdate == "agent_message_chunk" && event.Update.Content != nil {
				output.WriteString(event.Update.Content.Text)
			}
			return nil
		},
		OnCompleted: func(event client.StreamEvent) error {
			stopReason = defaultString(event.StopReason, "end_turn")
			completed = true
			return nil
		},
		OnError: func(event client.StreamEvent) error {
			message := ""
			if event.Error != nil {
				message = event.Error.Message
			}
			return errors.New(defaultString(message, "stream error"))
		},
	}

	for event := range events {
		if event.SessionID != "" {
			sessionID = event.SessionID
		}
		if event.RequestID != "" {
			requestID = event.RequestID
		}
		if err := client.DispatchStreamEvent(event, handler); err != nil {
			return "", "", sessionID, requestID, err
		}
	}

	if !completed {
		return "", "", sessionID, requestID, errors.New("agent stream closed before completion")
	}

	return output.String(), stopReason, sessionID, requestID, nil
}

func (s *ChatService) prepareRun(ctx context.Context, conversationID string) (*model.Conversation, *model.AgentProfile, *model.ProviderProfile, *model.AgentInstance, error) {
	conversation, err := s.conversations.Get(ctx, conversationID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	agent, err := s.agents.Get(ctx, conversation.AgentID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	provider, err := s.resolveProvider(ctx, agent)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	instance, err := s.router.SelectInstance(ctx, conversation)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return conversation, agent, provider, instance, nil
}

func (s *ChatService) resolveProvider(ctx context.Context, agent *model.AgentProfile) (*model.ProviderProfile, error) {
	if s.providers == nil || agent == nil {
		return nil, nil
	}
	if agent.ProviderID != "" {
		return s.providers.Get(ctx, agent.ProviderID)
	}
	if agent.ModelProvider == "" {
		return nil, nil
	}
	provider, err := s.providers.GetEnabledByType(ctx, agent.ModelProvider)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return provider, nil
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

func (s *ChatService) agentProfilePayload(ctx context.Context, agent model.AgentProfile, provider *model.ProviderProfile, metadata map[string]any) (map[string]any, error) {
	profile := agentProfileMap(agent, provider, metadata)
	return profile, nil
}

func agentProfileMap(agent model.AgentProfile, provider *model.ProviderProfile, metadata map[string]any) map[string]any {
	modelProvider := agent.ModelProvider
	modelName := agent.ModelName
	baseURL := agent.BaseURL
	apiKey := ""
	if provider != nil {
		if provider.Type != "" {
			modelProvider = provider.Type
		}
		if modelName == "" {
			modelName = provider.DefaultModel
		}
		if baseURL == "" {
			baseURL = provider.BaseURL
		}
		apiKey = provider.APIKey
	}
	profile := map[string]any{
		"model_provider": modelProvider,
		"model_name":     modelName,
		"base_url":       baseURL,
		"api_key":        apiKey,
		"system_prompt":  agent.SystemPrompt,
		"max_iterations": agent.MaxIterations,
		"network_allow":  parseStringSlice(agent.NetworkAllowJSON),
		"mcp_servers":    parseStringSlice(agent.MCPServerIDsJSON),
	}
	if tools := parseStringSlice(agent.ToolWhitelistJSON); len(tools) > 0 {
		profile["enabled_builtin_tools"] = tools
	}
	if projectRoot := metadataString(metadata, "project_root"); projectRoot != "" {
		profile["project_root"] = projectRoot
	}
	return profile
}

func metadataString(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	value, ok := metadata[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}
