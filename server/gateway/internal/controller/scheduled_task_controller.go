package controller

import (
	"errors"
	"net/http"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/repository"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type ScheduledTaskController struct {
	tasks *service.ScheduledTaskService
}

func NewScheduledTaskController(tasks *service.ScheduledTaskService) *ScheduledTaskController {
	return &ScheduledTaskController{tasks: tasks}
}

func (t *ScheduledTaskController) Create(c *gin.Context) {
	var req dto.CreateScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	task, err := t.tasks.Create(c.Request.Context(), req)
	if err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	c.JSON(http.StatusCreated, task)
}

func (t *ScheduledTaskController) Get(c *gin.Context) {
	task, err := t.tasks.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (t *ScheduledTaskController) List(c *gin.Context) {
	tasks, err := t.tasks.List(c.Request.Context())
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

func (t *ScheduledTaskController) Update(c *gin.Context) {
	var req dto.UpdateScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	task, err := t.tasks.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			writeGatewayRepositoryError(c, err)
			return
		}
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	c.JSON(http.StatusOK, task)
}

func (t *ScheduledTaskController) Delete(c *gin.Context) {
	if err := t.tasks.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (t *ScheduledTaskController) ListRuns(c *gin.Context) {
	limit := 20
	runs, err := t.tasks.ListRuns(c.Request.Context(), c.Param("id"), limit)
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"runs": runs})
}
