// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, nextTick, reactive } from 'vue'
import { DPageRenderer, createDPageRuntime } from '../src/index.js'
import chatToolResultSchema from '../src/schemas/examples/chat-tool-result.json'
import tableCardSchema from '../src/schemas/examples/table-card.json'
import liveInputPreviewSchema from '../src/schemas/examples/live-input-preview.json'

afterEach(() => {
  document.body.innerHTML = ''
})

describe('demo interactions', () => {
  it('syncs live preview title, notes and tone through input controls', async () => {
    const { container, app, runtime, flush } = mountSchema(liveInputPreviewSchema)

    try {
      const titleInput = container.querySelector('input[name="title"]')
      const notesTextarea = container.querySelector('textarea[name="notes"]')
      const toneSelect = container.querySelector('select[name="tone"]')

      titleInput.value = '新的预览标题'
      titleInput.dispatchEvent(new Event('input', { bubbles: true }))
      await flush()

      expect(runtime.state.title).toBe('新的预览标题')
      expect(container.textContent).toContain('新的预览标题')

      notesTextarea.value = '正文已经同步更新'
      notesTextarea.dispatchEvent(new Event('input', { bubbles: true }))
      await flush()

      expect(runtime.state.notes).toBe('正文已经同步更新')
      expect(container.textContent).toContain('正文已经同步更新')

      toneSelect.value = 'warning'
      toneSelect.dispatchEvent(new Event('change', { bubbles: true }))
      await flush()

      expect(runtime.state.tone).toBe('warning')
      expect(container.querySelector('.d-page-alert--warning')).toBeTruthy()
    } finally {
      app.unmount()
    }
  })

  it('syncs checkbox and switch boolean controls into runtime state', async () => {
    const { container, app, runtime, flush } = mountSchema(liveInputPreviewSchema)

    try {
      const confirmedCheckbox = container.querySelector('input[name="confirmed"]')
      const compactSwitch = container.querySelector('input[name="compactMode"]')

      confirmedCheckbox.checked = false
      confirmedCheckbox.dispatchEvent(new Event('change', { bubbles: true }))
      await flush()

      expect(runtime.state.confirmed).toBe(false)
      expect(container.textContent).toContain('false')

      compactSwitch.checked = true
      compactSwitch.dispatchEvent(new Event('change', { bubbles: true }))
      await flush()

      expect(runtime.state.compactMode).toBe(true)
      expect(container.textContent).toContain('紧凑模式：true')
    } finally {
      app.unmount()
    }
  })

  it('updates selectedName when a table row is clicked', async () => {
    const { container, app, runtime, flush } = mountSchema(tableCardSchema)

    try {
      const runtimeRow = Array.from(container.querySelectorAll('tbody tr')).find((row) => row.textContent.includes('运行时'))

      runtimeRow.dispatchEvent(new MouseEvent('click', { bubbles: true }))
      await flush()

      expect(runtime.state.selectedName).toBe('运行时')
      expect(container.textContent).toContain('已选择：运行时')
    } finally {
      app.unmount()
    }
  })

  it('records adapter and emit calls from chat result actions', async () => {
    const copyText = vi.fn()
    const emitted = []
    const { container, app, runtime, flush } = mountSchema(chatToolResultSchema, {
      adapters: { copyText },
      onEmit: (event) => emitted.push(event),
    })

    try {
      const copyButton = findButton(container, '复制摘要')
      const detailsButton = findButton(container, '打开详情')

      copyButton.dispatchEvent(new MouseEvent('click', { bubbles: true }))
      await flush()

      expect(copyText).toHaveBeenCalledWith(chatToolResultSchema.data.summary)
      expect(runtime.state.copied).toBe(true)
      expect(container.textContent).toContain('已复制')

      detailsButton.dispatchEvent(new MouseEvent('click', { bubbles: true }))
      await flush()

      expect(emitted).toHaveLength(1)
      expect(emitted[0]).toMatchObject({
        event: 'openDetails',
        payload: {
          source: 'chat-tool-result',
          status: chatToolResultSchema.data.status,
        },
      })
    } finally {
      app.unmount()
    }
  })
})

function mountSchema(schema, runtimeOptions = {}) {
  const host = document.createElement('div')
  document.body.appendChild(host)

  const runtime = createDPageRuntime({
    schema: clone(schema),
    ...runtimeOptions,
  })

  runtime.state = reactive(runtime.state)
  runtime.data = reactive(runtime.data)
  runtime.context = reactive(runtime.context)

  const app = createApp(DPageRenderer, {
    schema,
    runtime,
  })

  app.mount(host)

  return {
    app,
    runtime,
    container: host,
    flush,
  }
}

async function flush() {
  for (let index = 0; index < 4; index += 1) {
    await Promise.resolve()
    await nextTick()
  }
}

function findButton(container, label) {
  return Array.from(container.querySelectorAll('button')).find((button) => button.textContent.includes(label))
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}
