package controller

import (
	"net/http"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type AgentController struct {
	agents *service.AgentService
}

func NewAgentController(agents *service.AgentService) *AgentController {
	return &AgentController{agents: agents}
}

func (a *AgentController) Create(c *gin.Context) {
	var req dto.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	agent, err := a.agents.Create(c.Request.Context(), req)
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusCreated, agent)
}

func (a *AgentController) Get(c *gin.Context) {
	agent, err := a.agents.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, agent)
}

func (a *AgentController) List(c *gin.Context) {
	agents, err := a.agents.List(c.Request.Context())
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"agents": agents})
}

func (a *AgentController) Update(c *gin.Context) {
	var req dto.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	agent, err := a.agents.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, agent)
}

func (a *AgentController) Delete(c *gin.Context) {
	if err := a.agents.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
