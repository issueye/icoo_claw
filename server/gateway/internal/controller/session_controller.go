package controller

import (
	"net/http"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/sessionstore/model"
	"icoo_claw/server/gateway/internal/sessionstore/repository"
	"icoo_claw/server/gateway/internal/sessionstore/service"

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
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
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
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toSessionDTO(*session))
}

func (s *SessionController) List(c *gin.Context) {
	filter := model.SessionListFilter{
		UserID:  c.Query("user_id"),
		AgentID: c.Query("agent_id"),
		Status:  c.Query("status"),
		Offset:  repository.ParseIntQuery(c.Query("offset"), 0),
		Limit:   repository.ParseIntQuery(c.Query("limit"), 50),
	}
	list, err := s.sessions.List(c.Request.Context(), filter)
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.SessionsResponse{Sessions: toSessionDTOs(list.Sessions)})
}

func (s *SessionController) Get(c *gin.Context) {
	session, err := s.sessions.Get(c.Request.Context(), c.Param("session_id"))
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSessionDTO(*session))
}

func (s *SessionController) Update(c *gin.Context) {
	var req dto.UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
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
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, toSessionDTO(*updated))
}

func (s *SessionController) Delete(c *gin.Context) {
	if err := s.sessions.Delete(c.Request.Context(), c.Param("session_id")); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *SessionController) ListMessages(c *gin.Context) {
	page := model.MessagePage{
		Offset:   repository.ParseIntQuery(c.Query("offset"), 0),
		Limit:    repository.ParseIntQuery(c.Query("limit"), 50),
		Tail:     repository.ParseIntQuery(c.Query("tail"), 0),
		BeforeID: c.Query("before"),
		AfterID:  c.Query("after"),
	}
	list, err := s.sessions.ListMessages(c.Request.Context(), c.Param("session_id"), page)
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.MessagesResponse{Messages: toMessageDTOs(list.Messages), Revision: list.Revision})
}

func (s *SessionController) AppendMessages(c *gin.Context) {
	var req dto.MessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	if err := s.sessions.AppendMessages(c.Request.Context(), c.Param("session_id"), toMessageModels(req.Messages)); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *SessionController) ReplaceMessages(c *gin.Context) {
	var req dto.MessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	if err := s.sessions.ReplaceMessages(c.Request.Context(), c.Param("session_id"), toMessageModels(req.Messages), req.ExpectedRevision); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *SessionController) ListRuns(c *gin.Context) {
	page := model.RunPage{
		Offset: repository.ParseIntQuery(c.Query("offset"), 0),
		Limit:  repository.ParseIntQuery(c.Query("limit"), 50),
	}
	runs, err := s.sessions.ListRuns(c.Request.Context(), c.Param("session_id"), page)
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.RunsResponse{Runs: toRunDTOs(runs)})
}

func (s *SessionController) AppendRuns(c *gin.Context) {
	var req dto.RunsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	if err := s.sessions.AppendRuns(c.Request.Context(), c.Param("session_id"), toRunModels(req.Runs)); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *SessionController) ListRunEvents(c *gin.Context) {
	page := model.RunEventPage{
		Offset: repository.ParseIntQuery(c.Query("offset"), 0),
		Limit:  repository.ParseIntQuery(c.Query("limit"), 50),
	}
	events, err := s.sessions.ListRunEvents(c.Request.Context(), c.Param("session_id"), c.Param("run_id"), page)
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, dto.RunEventsResponse{Events: toRunEventDTOs(events)})
}

func (s *SessionController) AppendRunEvents(c *gin.Context) {
	var req dto.RunEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	if err := s.sessions.AppendRunEvents(c.Request.Context(), c.Param("session_id"), c.Param("run_id"), toRunEventModels(req.Events)); err != nil {
		writeGatewayRepositoryError(c, err)
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
		Revision:  session.Revision,
		CreatedAt: session.CreatedAt,
		UpdatedAt: session.UpdatedAt,
	}
}

func toSessionDTOs(sessions []model.Session) []dto.Session {
	out := make([]dto.Session, len(sessions))
	for i, session := range sessions {
		out[i] = toSessionDTO(session)
	}
	return out
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

func toRunDTOs(runs []model.Run) []dto.Run {
	out := make([]dto.Run, len(runs))
	for i, run := range runs {
		out[i] = dto.Run{
			ID:          run.ID,
			RequestID:   run.RequestID,
			Status:      run.Status,
			StopReason:  run.StopReason,
			Error:       run.Error,
			Usage:       run.Usage,
			Metadata:    run.Metadata,
			StartedAt:   run.StartedAt,
			CompletedAt: run.CompletedAt,
		}
	}
	return out
}

func toRunModels(runs []dto.Run) []model.Run {
	out := make([]model.Run, len(runs))
	for i, run := range runs {
		out[i] = model.Run{
			ID:          run.ID,
			RequestID:   run.RequestID,
			Status:      run.Status,
			StopReason:  run.StopReason,
			Error:       run.Error,
			Usage:       run.Usage,
			Metadata:    run.Metadata,
			StartedAt:   run.StartedAt,
			CompletedAt: run.CompletedAt,
		}
	}
	return out
}

func toRunEventDTOs(events []model.RunEvent) []dto.RunEvent {
	out := make([]dto.RunEvent, len(events))
	for i, event := range events {
		out[i] = dto.RunEvent{
			ID:        event.ID,
			RunID:     event.RunID,
			Type:      event.Type,
			Sequence:  event.Sequence,
			Payload:   event.Payload,
			Metadata:  event.Metadata,
			CreatedAt: event.CreatedAt,
		}
	}
	return out
}

func toRunEventModels(events []dto.RunEvent) []model.RunEvent {
	out := make([]model.RunEvent, len(events))
	for i, event := range events {
		out[i] = model.RunEvent{
			ID:        event.ID,
			RunID:     event.RunID,
			Type:      event.Type,
			Sequence:  event.Sequence,
			Payload:   event.Payload,
			Metadata:  event.Metadata,
			CreatedAt: event.CreatedAt,
		}
	}
	return out
}
