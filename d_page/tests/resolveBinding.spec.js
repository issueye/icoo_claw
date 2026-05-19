import { describe, expect, it } from 'vitest'
import { resolveBinding } from '../src/runtime/resolveBinding.js'

describe('resolveBinding', () => {
  const sources = {
    state: {
      user: { name: 'Ada' },
      count: 3,
    },
    data: {
      users: [{ name: 'Lin' }],
      summary: 'ready',
    },
    context: {
      messageId: 'msg-1',
    },
  }

  it('returns raw values for whole safe bindings', () => {
    expect(resolveBinding('{{ state.user.name }}', sources)).toBe('Ada')
    expect(resolveBinding('{{ state.count }}', sources)).toBe(3)
    expect(resolveBinding('{{ data.users.0.name }}', sources)).toBe('Lin')
    expect(resolveBinding('{{ context.messageId }}', sources)).toBe('msg-1')
  })

  it('interpolates safe bindings inside strings', () => {
    expect(resolveBinding('Hello {{ state.user.name }}: {{ data.summary }}', sources)).toBe('Hello Ada: ready')
  })

  it('resolves arrays and objects recursively', () => {
    expect(
      resolveBinding(
        {
          title: '{{ state.user.name }}',
          rows: ['{{ data.users.0.name }}'],
        },
        sources,
      ),
    ).toEqual({ title: 'Ada', rows: ['Lin'] })
  })

  it('rejects arbitrary JavaScript expressions', () => {
    expect(() => resolveBinding('{{ state.count + 1 }}', sources)).toThrow('Only state.xxx, data.xxx and context.xxx')
    expect(() => resolveBinding('{{ window.location }}', sources)).toThrow('Only state.xxx, data.xxx and context.xxx')
    expect(() => resolveBinding('{{ state.user.constructor }}', sources)).toThrow('Unsafe binding path segment')
  })
})
