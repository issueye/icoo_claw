package controller

import (
	"errors"
	"net/http"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/repository"
	"icoo_claw/server/gateway/internal/service"
	sessionrepo "icoo_claw/server/gateway/internal/sessionstore/repository"

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

func writeGatewayRepositoryError(c *gin.Context, err error) {
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, sessionrepo.ErrNotFound) {
		writeGatewayError(c, http.StatusNotFound, "not_found", err)
		return
	}
	if errors.Is(err, sessionrepo.ErrConflict) {
		writeGatewayError(c, http.StatusConflict, "revision_conflict", err)
		return
	}
	var downstream *client.HTTPError
	if errors.As(err, &downstream) {
		writeGatewayError(c, gatewayStatusForDownstream(downstream), gatewayCodeForDownstream(downstream), downstream)
		return
	}
	writeGatewayError(c, http.StatusBadGateway, "store_error", err)
}

func writeGatewayError(c *gin.Context, status int, code string, err error) {
	c.JSON(status, gin.H{"code": code, "error": err.Error()})
}

func gatewayStatusForDownstream(err *client.HTTPError) int {
	if err == nil {
		return http.StatusBadGateway
	}
	switch err.StatusCode {
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusConflict, http.StatusRequestEntityTooLarge:
		return err.StatusCode
	default:
		return http.StatusBadGateway
	}
}

func gatewayCodeForDownstream(err *client.HTTPError) string {
	if err == nil || err.Code == "" {
		return "dependency_unavailable"
	}
	if err.Service == "claw" && err.Code == "agent_error" {
		return "agent_error"
	}
	return err.Code
}
