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
        type: 'session/update',
        conversation_id: 'conv_1',
        request_id: 'req_1',
        update: {
          sessionUpdate: 'agent_message_chunk',
          content: { type: 'text', text: 'hello' },
        },
      }),
    ).toEqual({
      type: 'session/update',
      conversationId: 'conv_1',
      sessionId: '',
      requestId: 'req_1',
      update: {
        sessionUpdate: 'agent_message_chunk',
        content: { type: 'text', text: 'hello', uri: '', mimeType: '', data: null },
        messageId: '',
        toolCallId: '',
        title: '',
        kind: '',
        status: '',
        locations: [],
        rawInput: null,
        rawOutput: null,
        usage: null,
      },
      permission: null,
      stopReason: '',
      code: '',
      error: '',
      metadata: {},
    })
  })

  it('normalizes ACP permission request frames', () => {
    expect(
      normalizeWSMessage({
        type: 'session/request_permission',
        conversation_id: 'conv_1',
        request_id: 'req_1',
        permission: {
          id: 'perm_1',
          sessionId: 'sess_1',
          toolCall: {
            toolCallId: 'tool_1',
            title: 'Edit file',
            rawInput: { path: 'README.md' },
          },
          options: [
            { optionId: 'allow_once', name: 'Allow once', kind: 'allow_once' },
            { optionId: 'reject_once', name: 'Reject', kind: 'reject_once' },
          ],
        },
      }).permission,
    ).toEqual({
      id: 'perm_1',
      sessionId: 'sess_1',
      toolCall: {
        toolCallId: 'tool_1',
        title: 'Edit file',
        kind: '',
        status: '',
        locations: [],
        rawInput: { path: 'README.md' },
        rawOutput: null,
      },
      options: [
        { optionId: 'allow_once', name: 'Allow once', kind: 'allow_once', metadata: {} },
        { optionId: 'reject_once', name: 'Reject', kind: 'reject_once', metadata: {} },
      ],
      metadata: {},
    })
  })
})
