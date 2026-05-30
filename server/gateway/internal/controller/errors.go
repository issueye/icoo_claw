package controller

import (
	"errors"
	"net/http"

	"icoo_claw/server/gateway/internal/client"
	"icoo_claw/server/gateway/internal/repository"
	"icoo_claw/server/gateway/internal/service"
	sessionrepo "icoo_claw/server/gateway/internal/sessionstore/repository"
	sessionservice "icoo_claw/server/gateway/internal/sessionstore/service"

	"github.com/gin-gonic/gin"
)

func writeGatewayRepositoryError(c *gin.Context, err error) {
	writeGatewayError(c, gatewayStatusForError(err), gatewayCodeForError(err), err)
}

func writeGatewayError(c *gin.Context, status int, code string, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}
	c.JSON(status, gin.H{"code": code, "error": message})
}

func gatewayStatusForError(err error) int {
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, sessionrepo.ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, sessionrepo.ErrConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, service.ErrAgentDisabled) {
		return http.StatusForbidden
	}
	if errors.Is(err, sessionservice.ErrInvalidInput) {
		return http.StatusBadRequest
	}
	var downstream *client.HTTPError
	if errors.As(err, &downstream) {
		return gatewayStatusForDownstream(downstream)
	}
	return http.StatusBadGateway
}

func gatewayCodeForError(err error) string {
	if errors.Is(err, repository.ErrNotFound) || errors.Is(err, sessionrepo.ErrNotFound) {
		return "not_found"
	}
	if errors.Is(err, sessionrepo.ErrConflict) {
		return "revision_conflict"
	}
	if errors.Is(err, service.ErrAgentDisabled) {
		return "agent_disabled"
	}
	if errors.Is(err, sessionservice.ErrInvalidInput) {
		return "bad_request"
	}
	var downstream *client.HTTPError
	if errors.As(err, &downstream) {
		return gatewayCodeForDownstream(downstream)
	}
	return "store_error"
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
