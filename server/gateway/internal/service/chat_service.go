package service

import (
	"context"
	"time"

	"icoo_claw/common/agentproto"
	"icoo_claw/common/id"
	"icoo_claw/common/stringutil"
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
	sessions      SessionBackend
	executor      *GatewayAgentExecutor
}

func NewChatService(conversations repository.ConversationRepository, agents repository.AgentRepository, providers repository.ProviderRepository, router RouterPolicy, sessions SessionBackend, claw AgentRunner, skills ...*SkillService) *ChatService {
	_ = skills
	return &ChatService{
		conversations: conversations,
		agents:        agents,
		sessions:      sessions,
		executor: NewGatewayAgentExecutor(GatewayAgentExecutorConfig{
			Agents:    agents,
			Providers: providers,
			Router:    router,
			Runner:    claw,
		}),
	}
}

func (s *ChatService) CreateConversation(ctx context.Context, req dto.CreateConversationRequest) (*dto.Conversation, error) {
	agent, err := s.agents.Get(ctx, req.AgentID)
	if err != nil {
		return nil, err
	}
	if err := EnsureAgentRunnable(agent); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	conversation := model.Conversation{
		ID:        "conv_" + id.Random(),
		SessionID: "sess_" + id.Random(),
		AgentID:   req.AgentID,
		UserID:    req.UserID,
		Title:     stringutil.Default(req.Title, "New conversation"),
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
	conversation, err := s.conversations.Get(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.markConversationStatus(ctx, conversation, "running"); err != nil {
		return nil, err
	}
	defer func() {
		_ = s.markConversationStatus(context.Background(), conversation, "active")
	}()

	execCtx, events, cleanup, err := s.executor.Stream(ctx, AgentExecutionRequest{
		Conversation: conversation,
		Prompt:       req.Prompt,
		RequestID:    req.RequestID,
		ForceSkills:  req.ForceSkills,
		Metadata:     req.Metadata,
	})
	if err != nil {
		return nil, err
	}
	defer cleanup()

	output, stopReason, sessionID, requestID, err := collectClawStream(events, conversation.SessionID, req.RequestID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	conversation.LastMessageAt = &now
	conversation.UpdatedAt = now
	if err := s.conversations.Update(ctx, *conversation); err != nil {
		return nil, err
	}

	if execCtx != nil && execCtx.Request.RequestID != "" {
		requestID = execCtx.Request.RequestID
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
	conversation, err := s.conversations.Get(ctx, conversationID)
	if err != nil {
		return nil, err
	}
	if err := s.markConversationStatus(ctx, conversation, "running"); err != nil {
		return nil, err
	}

	_, events, cleanup, err := s.executor.Stream(ctx, AgentExecutionRequest{
		Conversation: conversation,
		Prompt:       req.Prompt,
		RequestID:    req.RequestID,
		ForceSkills:  req.ForceSkills,
		Metadata:     req.Metadata,
	})
	if err != nil {
		_ = s.markConversationStatus(context.Background(), conversation, "active")
		return nil, err
	}

	out := make(chan client.StreamEvent, 128)
	go func() {
		defer close(out)
		defer cleanup()
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
	collected, err := agentproto.CollectTextStream(events, fallbackSessionID, fallbackRequestID)
	if collected == nil {
		return "", "", fallbackSessionID, fallbackRequestID, err
	}
	if err != nil {
		return "", "", collected.SessionID, collected.RequestID, err
	}
	return collected.Output, collected.StopReason, collected.SessionID, collected.RequestID, nil
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
