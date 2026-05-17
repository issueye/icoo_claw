package controller

import (
	"net/http"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type AgentInstanceController struct {
	instances *service.AgentInstanceService
}

func NewAgentInstanceController(instances *service.AgentInstanceService) *AgentInstanceController {
	return &AgentInstanceController{instances: instances}
}

func (a *AgentInstanceController) Start(c *gin.Context) {
	var req dto.StartAgentInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	instance, err := a.instances.Start(c.Request.Context(), req)
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, instance)
}

func (a *AgentInstanceController) List(c *gin.Context) {
	instances, err := a.instances.List(c.Request.Context())
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"instances": instances})
}

func (a *AgentInstanceController) Stop(c *gin.Context) {
	if err := a.instances.Stop(c.Request.Context(), c.Param("id")); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *AgentInstanceController) Restart(c *gin.Context) {
	instance, err := a.instances.Restart(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusCreated, instance)
}

func (a *AgentInstanceController) Drain(c *gin.Context) {
	instance, err := a.instances.Drain(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, instance)
}
