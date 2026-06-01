import { toWebSocketURL } from '@/services/gateway/http'

export class AcpEventBusMonitorClient {
  constructor(baseUrl, handlers = {}) {
    this.baseUrl = baseUrl
    this.handlers = handlers
    this.socket = null
  }

  connect() {
    if (!this.baseUrl || this.socket) {
      return
    }

    this.handlers.onStateChange?.('connecting')
    const socket = new WebSocket(toWebSocketURL(this.baseUrl, '/v1/ws/events?protocol=acp'))
    this.socket = socket

    socket.addEventListener('open', () => {
      this.handlers.onStateChange?.('open')
    })

    socket.addEventListener('message', (message) => {
      const event = parseEventBusEvent(message.data)
      if (!event || event.protocol !== 'acp') {
        return
      }
      this.handlers.onEvent?.(event)
    })

    socket.addEventListener('close', () => {
      if (this.socket === socket) {
        this.socket = null
      }
      this.handlers.onStateChange?.('closed')
    })

    socket.addEventListener('error', () => {
      this.handlers.onError?.(new Error('event bus websocket error'))
    })
  }

  close() {
    const socket = this.socket
    this.socket = null
    socket?.close()
  }
}

export function eventBusEventToMonitorInput(event = {}) {
  return {
    id: event.id,
    time: event.time,
    direction: event.direction,
    type: event.type,
    conversationId: event.conversation_id || event.conversationId,
    sessionId: event.session_id || event.sessionId,
    requestId: event.request_id || event.requestId,
    source: event.source || 'event-bus',
    payload: event.payload || {},
  }
}

function parseEventBusEvent(payload) {
  try {
    return JSON.parse(String(payload || ''))
  } catch {
    return null
  }
}
