import { toWebSocketURL } from './http'
import { normalizeSessionFrame } from './session-events'
import { useAcpMonitorStore } from '@/stores/acpMonitor'

export class GatewayChatSocket {
  constructor(baseUrl, handlers = {}) {
    this.baseUrl = baseUrl
    this.handlers = handlers
    this.socket = null
    this.connectPromise = null
  }

  setHandlers(handlers) {
    this.handlers = handlers
  }

  async connect() {
    if (this.socket?.readyState === WebSocket.OPEN) {
      return
    }
    if (this.connectPromise) {
      return this.connectPromise
    }

    this.handlers.onStateChange?.('connecting')

    this.connectPromise = new Promise((resolve, reject) => {
      const socket = new WebSocket(toWebSocketURL(this.baseUrl))
      let settled = false

      socket.addEventListener('open', () => {
        settled = true
        this.socket = socket
        this.handlers.onStateChange?.('open')
        resolve()
      })

      socket.addEventListener('message', (event) => {
        try {
          const raw = JSON.parse(event.data)
          const message = normalizeWSMessage(raw)
          recordACPEvent('inbound', message)
          this.handlers.onMessage?.(message)
        } catch {
          this.handlers.onError?.(new Error('invalid websocket message'))
        }
      })

      socket.addEventListener('close', () => {
        this.socket = null
        this.connectPromise = null
        this.handlers.onStateChange?.('closed')
        if (!settled) {
          reject(new Error('chat socket closed before it became ready'))
        }
      })

      socket.addEventListener('error', () => {
        this.handlers.onError?.(new Error('gateway websocket error'))
        if (!settled) {
          reject(new Error('gateway websocket error'))
        }
      })
    })

    try {
      await this.connectPromise
    } finally {
      if (this.socket?.readyState !== WebSocket.OPEN) {
        this.connectPromise = null
      }
    }
  }

  async startChat(input) {
    await this.connect()
    this.send({
      type: 'chat.start',
      conversation_id: input.conversationId,
      prompt: input.prompt,
      request_id: input.requestId,
      metadata: input.metadata || {},
    })
  }

  cancelChat(input) {
    this.send({
      type: 'chat.cancel',
      conversation_id: input.conversationId,
      request_id: input.requestId,
    })
  }

  sendPermissionDecision(input) {
    this.send({
      type: 'chat.permission_decision',
      conversation_id: input.conversationId,
      request_id: input.requestId,
      permission_id: input.permissionId,
      outcome: input.outcome,
      option_id: input.optionId || '',
    })
  }

  ping() {
    this.send({ type: 'ping' })
  }

  close() {
    this.socket?.close()
    this.socket = null
    this.connectPromise = null
  }

  send(payload) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      throw new Error('chat socket is not connected')
    }
    recordACPEvent('outbound', payload)
    this.socket.send(JSON.stringify(payload))
  }
}

export function normalizeWSMessage(raw = {}) {
  return normalizeSessionFrame(raw)
}

function recordACPEvent(direction, payload) {
  try {
    useAcpMonitorStore().record({
      direction,
      payload,
      conversationId: payload.conversationId || payload.conversation_id,
      sessionId: payload.sessionId || payload.session_id,
      requestId: payload.requestId || payload.request_id,
    })
  } catch {
    // Store may be unavailable in isolated helper tests before Pinia is active.
  }
}
