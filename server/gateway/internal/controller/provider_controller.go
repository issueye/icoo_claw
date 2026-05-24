package controller

import (
	"net/http"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type ProviderController struct {
	providers *service.ProviderService
}

func NewProviderController(providers *service.ProviderService) *ProviderController {
	return &ProviderController{providers: providers}
}

func (p *ProviderController) Create(c *gin.Context) {
	var req dto.CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	provider, err := p.providers.Create(c.Request.Context(), req)
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusCreated, provider)
}

func (p *ProviderController) Get(c *gin.Context) {
	provider, err := p.providers.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, provider)
}

func (p *ProviderController) List(c *gin.Context) {
	providers, err := p.providers.List(c.Request.Context())
	if err != nil {
		writeGatewayError(c, http.StatusBadGateway, "store_error", err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (p *ProviderController) Update(c *gin.Context) {
	var req dto.UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeGatewayError(c, http.StatusBadRequest, "bad_request", err)
		return
	}
	provider, err := p.providers.Update(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.JSON(http.StatusOK, provider)
}

func (p *ProviderController) Delete(c *gin.Context) {
	if err := p.providers.Delete(c.Request.Context(), c.Param("id")); err != nil {
		writeGatewayRepositoryError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
