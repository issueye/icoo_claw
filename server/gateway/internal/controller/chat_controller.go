package controller

import (
	"net/http"

	"icoo_claw/server/gateway/internal/dto"
	"icoo_claw/server/gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type ChatController struct {
	chat *service.ChatService
}

func NewChatController(chat *service.ChatService) *ChatController {
	return &ChatController{chat: chat}
}

func (c *ChatController) CreateConversation(ctx *gin.Context) {
	var req dto.CreateConversationRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeGatewayError(ctx, http.StatusBadRequest, "bad_request", err)
		return
	}
	conversation, err := c.chat.CreateConversation(ctx.Request.Context(), req)
	if err != nil {
		writeGatewayRepositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, conversation)
}

func (c *ChatController) ListConversations(ctx *gin.Context) {
	conversations, err := c.chat.ListConversations(ctx.Request.Context())
	if err != nil {
		writeGatewayError(ctx, http.StatusBadGateway, "store_error", err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"conversations": conversations})
}

func (c *ChatController) ListMessages(ctx *gin.Context) {
	messages, err := c.chat.ListMessages(ctx.Request.Context(), ctx.Param("id"))
	if err != nil {
		writeGatewayRepositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, dto.ConversationMessagesResponse{Messages: messages})
}

func (c *ChatController) SendMessage(ctx *gin.Context) {
	var req dto.SendMessageRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		writeGatewayError(ctx, http.StatusBadRequest, "bad_request", err)
		return
	}
	resp, err := c.chat.SendMessage(ctx.Request.Context(), ctx.Param("id"), req)
	if err != nil {
		writeGatewayRepositoryError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

func (c *ChatController) DeleteConversation(ctx *gin.Context) {
	if err := c.chat.DeleteConversation(ctx.Request.Context(), ctx.Param("id")); err != nil {
		writeGatewayRepositoryError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
