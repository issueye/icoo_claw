import { defineStore } from 'pinia'

const maxEvents = 500

export const useAcpMonitorStore = defineStore('acpMonitor', {
  state: () => ({
    open: false,
    paused: false,
    events: [],
  }),

  getters: {
    total: (state) => state.events.length,
    inboundCount: (state) => state.events.filter((event) => event.direction === 'inbound').length,
    outboundCount: (state) => state.events.filter((event) => event.direction === 'outbound').length,
    permissionCount: (state) => state.events.filter((event) => isPermissionEvent(event.type)).length,
    latest: (state) => state.events[0] || null,
  },

  actions: {
    toggleOpen() {
      this.open = !this.open
    },

    setOpen(value) {
      this.open = Boolean(value)
    },

    togglePaused() {
      this.paused = !this.paused
    },

    record(input = {}) {
      if (this.paused) {
        return ''
      }

      const type = String(input.type || input.payload?.type || '').trim()
      const event = {
        id: buildEventId(),
        time: Date.now(),
        direction: input.direction === 'outbound' ? 'outbound' : 'inbound',
        type: type || 'unknown',
        conversationId: String(input.conversationId || input.payload?.conversationId || input.payload?.conversation_id || ''),
        sessionId: String(input.sessionId || input.payload?.sessionId || input.payload?.session_id || ''),
        requestId: String(input.requestId || input.payload?.requestId || input.payload?.request_id || ''),
        source: input.source || 'gateway-ws',
        summary: input.summary || summarizePayload(type, input.payload),
        payload: clonePayload(input.payload || {}),
      }

      this.events = [event, ...this.events].slice(0, maxEvents)
      return event.id
    },

    clear() {
      this.events = []
    },
  },
})

function buildEventId() {
  return `acp_evt_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`
}

function clonePayload(payload) {
  if (!payload || typeof payload !== 'object') {
    return payload
  }
  try {
    return JSON.parse(JSON.stringify(payload))
  } catch {
    return payload
  }
}

function summarizePayload(type, payload = {}) {
  if (type === 'session/update') {
    return payload.update?.title || payload.update?.content?.text || payload.update?.sessionUpdate || '会话更新'
  }
  if (type === 'session/request_permission') {
    return payload.permission?.toolCall?.title || payload.permission?.tool_call?.title || '权限请求'
  }
  if (type === 'chat.permission_decision') {
    return payload.optionId || payload.option_id || payload.outcome || '权限决策'
  }
  if (type === 'chat.start') {
    return payload.prompt || '开始会话'
  }
  if (type === 'chat.cancel') {
    return '取消会话'
  }
  if (type === 'session/error') {
    return payload.error || '会话错误'
  }
  if (type === 'session/completed') {
    return payload.stopReason || payload.stop_reason || '会话完成'
  }
  return type || 'ACP 事件'
}

function isPermissionEvent(type) {
  return ['session/request_permission', 'chat.permission_decision', 'chat.permission_decision.accepted'].includes(type)
}
