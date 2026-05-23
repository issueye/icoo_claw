import { defineStore } from 'pinia'
import router from '@/router'
import { GatewayChatSocket } from '@/services/gateway/ws'
import { buildConversationTitle } from '@/services/utils/title'
import { useAgentsStore } from './agents'
import { useConversationsStore } from './conversations'
import { useProjectsStore } from './projects'
import { useSettingsStore } from './settings'

export const useChatStore = defineStore('chat', {
  state: () => ({
    streaming: false,
    socketState: 'idle',
    error: '',
    composerDraft: '',
    activeConversationId: '',
    activeRequestId: '',
    sessionId: '',
    socket: null,
    socketBaseUrl: '',
  }),

  actions: {
    async sendPrompt(prompt, conversationId = '') {
      const content = String(prompt || '').trim()
      if (!content || this.streaming) {
        return
      }

      const settingsStore = useSettingsStore()
      const agentsStore = useAgentsStore()
      const conversationsStore = useConversationsStore()
      const projectsStore = useProjectsStore()
      const baseUrl = settingsStore.settings.gateway.baseUrl
      const metadata = projectsStore.currentProjectMetadata

      if (!agentsStore.items.some((item) => item.id === settingsStore.settings.gateway.defaultAgentId)) {
        await agentsStore.fetchAgents(baseUrl)
      }

      const agentId = settingsStore.settings.gateway.defaultAgentId
      if (!agentId) {
        throw new Error('当前网关没有可用 Agent，请先在网关中创建 Agent 后再发起对话')
      }

      this.error = ''
      let targetConversationId = conversationId

      if (!targetConversationId) {
        const conversation = await conversationsStore.createConversation(baseUrl, {
          agentId,
          title: buildConversationTitle(content),
        })
        targetConversationId = conversation.id
        await router.push({ name: 'chat-conversation', params: { id: targetConversationId } })
      }

      conversationsStore.appendLocalUserMessage(targetConversationId, content)
      conversationsStore.startAssistantDraft(targetConversationId)

      this.streaming = true
      this.activeConversationId = targetConversationId
      this.activeRequestId = buildRequestID()
      try {
        const socket = await this.ensureSocket(baseUrl)
        await socket.startChat({
          conversationId: targetConversationId,
          prompt: content,
          requestId: this.activeRequestId,
          metadata,
        })
      } catch (error) {
        this.streaming = false
        this.activeRequestId = ''
        this.error = error?.message || String(error)
        conversationsStore.markAssistantDraftError(targetConversationId, this.error)
        throw error
      }
    },

    async cancelStream() {
      if (!this.streaming || !this.socket) {
        return
      }
      this.socket.cancelChat({
        conversationId: this.activeConversationId,
        requestId: this.activeRequestId,
      })
    },

    async ensureSocket(baseUrl) {
      if (!this.socket || this.socketBaseUrl !== baseUrl) {
        this.socket?.close()
        this.socketBaseUrl = baseUrl
        this.socket = new GatewayChatSocket(baseUrl, {
          onStateChange: (state) => {
            this.socketState = state
            if (state === 'closed' && this.streaming) {
              this.streaming = false
            }
          },
          onMessage: (message) => {
            void this.handleSocketMessage(message)
          },
          onError: (error) => {
            this.error = error?.message || String(error)
          },
        })
      }

      await this.socket.connect()
      return this.socket
    },

    async handleSocketMessage(message) {
      const conversationsStore = useConversationsStore()
      const settingsStore = useSettingsStore()

      switch (message.type) {
        case 'session.accepted':
          if (message.sessionId) {
            this.sessionId = message.sessionId
          }
          break
        case 'message.delta':
          conversationsStore.appendAssistantDelta(message.conversationId || this.activeConversationId, message.output)
          break
        case 'message.completed':
          conversationsStore.markAssistantDraftComplete(message.conversationId || this.activeConversationId)
          this.streaming = false
          this.activeRequestId = ''
          this.error = ''
          if (message.sessionId) {
            this.sessionId = message.sessionId
          }
          if (message.conversationId || this.activeConversationId) {
            await conversationsStore.loadMessages(
              settingsStore.settings.gateway.baseUrl,
              message.conversationId || this.activeConversationId,
              { force: true },
            )
          }
          break
        case 'message.error':
          this.streaming = false
          this.error = message.error || 'chat request failed'
          conversationsStore.markAssistantDraftError(message.conversationId || this.activeConversationId, this.error)
          this.activeRequestId = ''
          break
        default:
          break
      }
    },
  },
})

function buildRequestID() {
  return `req_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`
}
