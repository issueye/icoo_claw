import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useChatStore } from '@/stores/chat'
import { useConversationsStore } from '@/stores/conversations'

vi.mock('@/router', () => ({
  default: {
    push: vi.fn(),
  },
}))

vi.mock('@/services/wails/config', () => ({
  loadDesktopSettings: vi.fn(),
  saveDesktopSettings: vi.fn(),
}))

describe('chat store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('tracks running streams independently by conversation', () => {
    const chatStore = useChatStore()

    chatStore.setStream('conv_a', { requestId: 'req_a', socketState: 'open' })
    chatStore.setStream('conv_b', { requestId: 'req_b', socketState: 'connecting' })

    expect(chatStore.streaming).toBe(true)
    expect(chatStore.anyStreaming).toBe(true)
    expect(chatStore.runningConversationIds).toEqual(['conv_a', 'conv_b'])
    expect(chatStore.isStreaming('conv_a')).toBe(true)
    expect(chatStore.isStreaming('conv_b')).toBe(true)
    expect(chatStore.isStreaming('conv_c')).toBe(false)
    expect(chatStore.socketState).toBe('connecting')
    expect(chatStore.socketStateFor('conv_a')).toBe('open')

    chatStore.cleanupStream('conv_a')

    expect(chatStore.runningConversationIds).toEqual(['conv_b'])
    expect(chatStore.isStreaming('conv_a')).toBe(false)
    expect(chatStore.socketState).toBe('connecting')
  })

  it('keeps composer drafts per conversation', () => {
    const chatStore = useChatStore()

    chatStore.setComposerDraft('conv_a', 'hello a')
    chatStore.setComposerDraft('conv_b', 'hello b')

    expect(chatStore.composerDraftFor('conv_a')).toBe('hello a')
    expect(chatStore.composerDraftFor('conv_b')).toBe('hello b')

    chatStore.clearComposerDraft('conv_a')

    expect(chatStore.composerDraftFor('conv_a')).toBe('')
    expect(chatStore.composerDraftFor('conv_b')).toBe('hello b')
  })

  it('routes websocket deltas to the matching conversation draft', async () => {
    const chatStore = useChatStore()
    const conversationsStore = useConversationsStore()

    conversationsStore.items = [
      { id: 'conv_a', title: 'A' },
      { id: 'conv_b', title: 'B' },
    ]
    conversationsStore.startAssistantDraft('conv_a')
    conversationsStore.startAssistantDraft('conv_b')

    await chatStore.handleSocketMessage({ type: 'session/update', update: textUpdate('alpha') }, 'conv_a')
    await chatStore.handleSocketMessage({ type: 'session/update', update: textUpdate('beta') }, 'conv_b')

    expect(conversationsStore.messagesFor('conv_a').at(-1).content).toBe('alpha')
    expect(conversationsStore.messagesFor('conv_b').at(-1).content).toBe('beta')
  })
})

function textUpdate(text) {
  return {
    sessionUpdate: 'agent_message_chunk',
    content: { type: 'text', text },
  }
}
