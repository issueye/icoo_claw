package controller

import (
	"net/http"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type SkillController struct {
	skills *service.SkillService
}

func NewSkillController(skills *service.SkillService) *SkillController {
	return &SkillController{skills: skills}
}

func (s *SkillController) Create(c *gin.Context) {
	var req dto.CreateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	skill, err := s.skills.Create(c.Request.Context(), req)
	if err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	c.JSON(http.StatusCreated, skill)
}

func (s *SkillController) Get(c *gin.Context) {
	skill, err := s.skills.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, skill)
}

func (s *SkillController) List(c *gin.Context) {
	skills, err := s.skills.List(c.Request.Context())
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"skills": skills})
}

func (s *SkillController) Update(c *gin.Context) {
	var req dto.UpdateSkillRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	skill, err := s.skills.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, skill)
}

func (s *SkillController) Delete(c *gin.Context) {
	if err := s.skills.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *SkillController) Download(c *gin.Context) {
	data, filename, err := s.skills.Download(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(http.StatusOK, "text/markdown; charset=utf-8", data)
}

func (s *SkillController) Sync(c *gin.Context) {
	summary, err := s.skills.SyncSummary(c.Request.Context())
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusOK, summary)
}
