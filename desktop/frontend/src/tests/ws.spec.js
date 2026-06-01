import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { toWebSocketURL } from '@/services/gateway/http'
import { GatewayChatSocket, normalizeWSMessage } from '@/services/gateway/ws'
import { useAcpMonitorStore } from '@/stores/acpMonitor'

describe('gateway websocket helpers', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('converts http urls to websocket urls', () => {
    expect(toWebSocketURL('http://127.0.0.1:8080')).toBe('ws://127.0.0.1:8080/v1/ws/chat')
    expect(toWebSocketURL('https://gateway.example.com')).toBe('wss://gateway.example.com/v1/ws/chat')
    expect(toWebSocketURL('http://127.0.0.1:8080', '/v1/ws/events?protocol=acp')).toBe(
      'ws://127.0.0.1:8080/v1/ws/events?protocol=acp',
    )
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
        entries: [],
        configOptions: [],
        currentModeId: '',
        availableModes: [],
      },
      permission: null,
      stopReason: '',
      code: '',
      error: '',
      metadata: {},
    })
  })

  it('normalizes ACP plan and config option updates', () => {
    const plan = normalizeWSMessage({
      type: 'session/update',
      update: {
        sessionUpdate: 'plan',
        entries: [{ content: 'Inspect files', priority: 'high', status: 'in_progress' }],
      },
    })
    expect(plan.update.entries).toEqual([
      { content: 'Inspect files', priority: 'high', status: 'in_progress', metadata: {} },
    ])

    const config = normalizeWSMessage({
      type: 'session/update',
      update: {
        sessionUpdate: 'config_option_update',
        configOptions: [
          {
            id: 'mode',
            name: 'Session Mode',
            category: 'mode',
            type: 'select',
            currentValue: 'code',
            options: [{ value: 'ask', name: 'Ask' }],
          },
        ],
      },
    })
    expect(config.update.configOptions[0]).toMatchObject({
      id: 'mode',
      category: 'mode',
      currentValue: 'code',
      options: [{ value: 'ask', name: 'Ask', description: '', metadata: {} }],
    })

    const mode = normalizeWSMessage({
      type: 'session/update',
      update: {
        sessionUpdate: 'current_mode_update',
        currentModeId: 'architect',
      },
    })
    expect(mode.update.currentModeId).toBe('architect')
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

  it('records outbound websocket payloads in the ACP monitor', async () => {
    const monitorStore = useAcpMonitorStore()
    const socket = new GatewayChatSocket('http://127.0.0.1:8080')
    socket.socket = {
      readyState: WebSocket.OPEN,
      send: () => {},
    }

    socket.sendPermissionDecision({
      conversationId: 'conv_1',
      requestId: 'req_1',
      permissionId: 'perm_1',
      outcome: 'selected',
      optionId: 'allow_once',
    })

    expect(monitorStore.total).toBe(1)
    expect(monitorStore.events[0].direction).toBe('outbound')
    expect(monitorStore.events[0].type).toBe('chat.permission_decision')
    expect(monitorStore.events[0].conversationId).toBe('conv_1')
  })
})
