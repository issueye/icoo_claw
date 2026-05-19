import { describe, expect, it, vi } from 'vitest'
import { createActionRegistry } from '../src/registry/createActionRegistry.js'
import { defaultActions } from '../src/registry/defaultActions.js'
import { createDPageRuntime } from '../src/runtime/createDPageRuntime.js'

function createRuntime(options = {}) {
  return createDPageRuntime({
    schema: {
      state: { count: 1, nested: { value: 'old' } },
      data: { url: 'https://example.com', text: 'copy me' },
      actions: {
        rename: { type: 'setState', payload: { name: 'Ada' } },
        emitSaved: { type: 'emit', payload: { event: 'saved', payload: { id: '{{ context.id }}' } } },
        copy: { type: 'copyText', payload: { text: '{{ data.text }}' } },
        open: { type: 'openUrl', payload: { url: '{{ data.url }}', target: '_blank' } },
        combo: {
          type: 'chain',
          payload: {
            actions: [
              { type: 'setState', payload: { count: 2 } },
              { type: 'emit', payload: { event: 'done' } },
            ],
          },
        },
      },
    },
    context: { id: 'ctx-1' },
    ...options,
  })
}

describe('runtime actions', () => {
  it('creates an action registry with default safe actions only', () => {
    const registry = createActionRegistry(defaultActions)

    expect(registry.list().sort()).toEqual(['chain', 'copyText', 'emit', 'openUrl', 'setState'])
    expect(registry.has('request')).toBe(false)
  })

  it('executes setState by id and inline object', async () => {
    const runtime = createRuntime()

    await expect(runtime.executeAction('rename')).resolves.toMatchObject({ ok: true, type: 'setState' })
    expect(runtime.state.name).toBe('Ada')

    await runtime.executeAction({ type: 'setState', payload: { path: 'nested.value', value: 'new' } })
    expect(runtime.state.nested.value).toBe('new')
  })

  it('emits events with resolved payload', async () => {
    const runtime = createRuntime()
    const listener = vi.fn()
    runtime.on('saved', listener)

    const result = await runtime.executeAction('emitSaved')

    expect(result).toMatchObject({ ok: true, type: 'emit', event: 'saved' })
    expect(runtime.emitted[0].payload).toEqual({ id: 'ctx-1' })
    expect(listener).toHaveBeenCalledTimes(1)
  })

  it('uses adapters for copyText and openUrl', async () => {
    const copyText = vi.fn()
    const openUrl = vi.fn()
    const runtime = createRuntime({ adapters: { copyText, openUrl } })

    await expect(runtime.executeAction('copy')).resolves.toMatchObject({ ok: true, type: 'copyText', text: 'copy me' })
    await expect(runtime.executeAction('open')).resolves.toMatchObject({ ok: true, type: 'openUrl', url: 'https://example.com' })

    expect(copyText).toHaveBeenCalledWith('copy me')
    expect(openUrl).toHaveBeenCalledWith('https://example.com', { target: '_blank' })
  })

  it('returns understandable adapter errors instead of using browser globals', async () => {
    const runtime = createRuntime()

    await expect(runtime.executeAction('copy')).resolves.toMatchObject({
      ok: false,
      error: { code: 'ADAPTER_MISSING' },
    })
    await expect(runtime.executeAction('open')).resolves.toMatchObject({
      ok: false,
      error: { code: 'ADAPTER_MISSING' },
    })
  })

  it('executes chain actions in order', async () => {
    const runtime = createRuntime()

    const result = await runtime.executeAction('combo')

    expect(result).toMatchObject({ ok: true, type: 'chain' })
    expect(runtime.state.count).toBe(2)
    expect(runtime.emitted.at(-1).event).toBe('done')
  })

  it('supports injected host actions', async () => {
    const runtime = createRuntime({
      actions: {
        hostSave(action) {
          return { ok: true, type: action.type, saved: true }
        },
      },
    })

    await expect(runtime.executeAction({ type: 'hostSave' })).resolves.toEqual({
      ok: true,
      type: 'hostSave',
      saved: true,
    })
  })
})
