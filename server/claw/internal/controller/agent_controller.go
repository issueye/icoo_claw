package controller

import (
	"errors"
	"net/http"

	"icoo_claw/server/claw/internal/dto"
	"icoo_claw/server/claw/internal/service"
	"icoo_claw/server/claw/pkg/agent_sdk"

	"github.com/gin-gonic/gin"
)

type AgentController struct {
	agentService *service.AgentService
}

func NewAgentController(agentService *service.AgentService) *AgentController {
	return &AgentController{agentService: agentService}
}

func (a *AgentController) Run(c *gin.Context) {
	var req dto.RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "bad_request", "error": err.Error()})
		return
	}

	resp, err := a.agentService.Run(c.Request.Context(), req)
	if err != nil {
		writeAgentError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (a *AgentController) RunStream(c *gin.Context) {
	var req dto.RunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "bad_request", "error": err.Error()})
		return
	}

	events, err := a.agentService.RunStream(c.Request.Context(), req)
	if err != nil {
		writeAgentError(c, err)
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	for event := range events {
		c.SSEvent("message", event)
		c.Writer.Flush()
	}
}

func writeAgentError(c *gin.Context, err error) {
	if errors.Is(err, agent_sdk.ErrSessionBusy) {
		c.JSON(http.StatusConflict, gin.H{"code": "session_busy", "error": err.Error()})
		return
	}
	c.JSON(http.StatusBadGateway, gin.H{"code": "agent_error", "error": err.Error()})
}
