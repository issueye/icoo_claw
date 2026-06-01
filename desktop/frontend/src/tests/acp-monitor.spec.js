import { beforeEach, describe, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAcpMonitorStore } from '@/stores/acpMonitor'

describe('ACP monitor store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('records inbound and outbound ACP events', () => {
    const store = useAcpMonitorStore()

    store.record({
      direction: 'outbound',
      payload: {
        type: 'chat.start',
        conversation_id: 'conv_1',
        request_id: 'req_1',
        prompt: 'hello',
      },
    })
    store.record({
      direction: 'inbound',
      payload: {
        type: 'session/request_permission',
        conversationId: 'conv_1',
        requestId: 'req_1',
        permission: { toolCall: { title: 'Edit file' } },
      },
    })

    expect(store.total).toBe(2)
    expect(store.outboundCount).toBe(1)
    expect(store.inboundCount).toBe(1)
    expect(store.permissionCount).toBe(1)
    expect(store.events[0].summary).toBe('Edit file')
    expect(store.events[1].conversationId).toBe('conv_1')
  })

  it('does not record while paused and can clear events', () => {
    const store = useAcpMonitorStore()

    store.record({ payload: { type: 'session/update' } })
    store.togglePaused()
    store.record({ payload: { type: 'session/error', error: 'boom' } })

    expect(store.total).toBe(1)
    store.clear()
    expect(store.total).toBe(0)
  })
})
