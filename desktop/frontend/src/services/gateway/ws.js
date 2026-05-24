import { toWebSocketURL } from './http'

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
          this.handlers.onMessage?.(normalizeWSMessage(raw))
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
    this.socket.send(JSON.stringify(payload))
  }
}

export function normalizeWSMessage(raw = {}) {
  return {
    type: raw.type || '',
    conversationId: raw.conversation_id || raw.conversationId || '',
    sessionId: raw.session_id || raw.sessionId || '',
    requestId: raw.request_id || raw.requestId || '',
    update: raw.update || null,
    stopReason: raw.stop_reason || raw.stopReason || '',
    code: raw.code || '',
    error: raw.error || '',
    metadata: raw.metadata || {},
  }
}
