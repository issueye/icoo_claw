package controller

import (
	"errors"
	"net/http"

	"icoo_claw/server/session_store/internal/dto"
	"icoo_claw/server/session_store/internal/model"
	"icoo_claw/server/session_store/internal/repository"
	"icoo_claw/server/session_store/internal/service"

	"github.com/gin-gonic/gin"
)

type SessionController struct {
	sessions *service.SessionService
}

func NewSessionController(sessions *service.SessionService) *SessionController {
	return &SessionController{sessions: sessions}
}

func (s *SessionController) Create(c *gin.Context) {
	var req dto.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err)
		return
	}

	session, err := s.sessions.Create(c.Request.Context(), model.Session{
		SessionID: req.SessionID,
		UserID:    req.UserID,
		AgentID:   req.AgentID,
		Title:     req.Title,
		Metadata:  req.Metadata,
	})
	if err != nil {
		writeError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusCreated, toSessionDTO(*session))
}

func (s *SessionController) Get(c *gin.Context) {
	session, err := s.sessions.Get(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSessionDTO(*session))
}

func (s *SessionController) Update(c *gin.Context) {
	var req dto.UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err)
		return
	}

	session := model.Session{SessionID: c.Param("session_id"), Metadata: req.Metadata}
	if req.Title != nil {
		session.Title = *req.Title
	}
	if req.Status != nil {
		session.Status = *req.Status
	}

	updated, err := s.sessions.Update(c.Request.Context(), session)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSessionDTO(*updated))
}

func (s *SessionController) Delete(c *gin.Context) {
	if err := s.sessions.Delete(c.Request.Context(), c.Param("session_id")); err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *SessionController) ListMessages(c *gin.Context) {
	page := model.MessagePage{
		Offset: repository.ParseIntQuery(c.Query("offset"), 0),
		Limit:  repository.ParseIntQuery(c.Query("limit"), 50),
	}
	messages, err := s.sessions.ListMessages(c.Request.Context(), c.Param("session_id"), page)
	if err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessagesResponse{Messages: toMessageDTOs(messages)})
}

func (s *SessionController) AppendMessages(c *gin.Context) {
	var req dto.MessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	if err := s.sessions.AppendMessages(c.Request.Context(), c.Param("session_id"), toMessageModels(req.Messages)); err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *SessionController) ReplaceMessages(c *gin.Context) {
	var req dto.MessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	if err := s.sessions.ReplaceMessages(c.Request.Context(), c.Param("session_id"), toMessageModels(req.Messages)); err != nil {
		writeRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func toSessionDTO(session model.Session) dto.Session {
	return dto.Session{
		SessionID: session.SessionID,
		UserID:    session.UserID,
		AgentID:   session.AgentID,
		Title:     session.Title,
		Status:    session.Status,
		Metadata:  session.Metadata,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
}

func toMessageDTOs(messages []model.Message) []dto.Message {
	out := make([]dto.Message, len(messages))
	for i, msg := range messages {
		out[i] = dto.Message{
			ID:            msg.ID,
			Role:          msg.Role,
			Content:       msg.Content,
			ContentBlocks: msg.ContentBlocks,
			ToolCalls:     msg.ToolCalls,
			Metadata:      msg.Metadata,
			CreatedAt:     msg.CreatedAt,
		}
	}
	return out
}

func toMessageModels(messages []dto.Message) []model.Message {
	out := make([]model.Message, len(messages))
	for i, msg := range messages {
		out[i] = model.Message{
			ID:            msg.ID,
			Role:          msg.Role,
			Content:       msg.Content,
			ContentBlocks: msg.ContentBlocks,
			ToolCalls:     msg.ToolCalls,
			Metadata:      msg.Metadata,
			CreatedAt:     msg.CreatedAt,
		}
	}
	return out
}

func writeRepositoryError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		writeError(c, http.StatusNotFound, "not_found", err)
		return
	}
	writeError(c, http.StatusBadGateway, "store_error", err)
}

func writeError(c *gin.Context, status int, code string, err error) {
	c.JSON(status, gin.H{"code": code, "error": err.Error()})
}
