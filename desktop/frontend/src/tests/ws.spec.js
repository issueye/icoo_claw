import { describe, expect, it } from 'vitest'
import { toWebSocketURL } from '@/services/gateway/http'
import { normalizeWSMessage } from '@/services/gateway/ws'

describe('gateway websocket helpers', () => {
  it('converts http urls to websocket urls', () => {
    expect(toWebSocketURL('http://127.0.0.1:8080')).toBe('ws://127.0.0.1:8080/v1/ws/chat')
    expect(toWebSocketURL('https://gateway.example.com')).toBe('wss://gateway.example.com/v1/ws/chat')
  })

  it('normalizes gateway frames into camelCase', () => {
    expect(
      normalizeWSMessage({
        type: 'message.delta',
        conversation_id: 'conv_1',
        request_id: 'req_1',
        output: 'hello',
      }),
    ).toEqual({
      type: 'message.delta',
      conversationId: 'conv_1',
      sessionId: '',
      requestId: 'req_1',
      output: 'hello',
      stopReason: '',
      code: '',
      error: '',
      metadata: {},
    })
  })
})
