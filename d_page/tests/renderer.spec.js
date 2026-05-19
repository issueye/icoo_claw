import { createSSRApp } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { describe, expect, it } from 'vitest'
import DPageRenderer from '../src/components/DPageRenderer.vue'

describe('DPageRenderer', () => {
  it('renders a card tree schema with default components', async () => {
    const html = await renderSchema({
      state: { title: 'Renderer ready' },
      root: {
        id: 'root',
        type: 'card',
        kind: 'page',
        children: [
          {
            id: 'title',
            type: 'card',
            kind: 'display',
            component: {
              type: 'heading',
              props: {
                level: 2,
                text: '{{ state.title }}',
              },
            },
          },
        ],
      },
    })

    expect(html).toContain('Renderer ready')
    expect(html).toContain('data-d-page-card-id="root"')
  })

  it('shows an unknown component fallback without blocking the page', async () => {
    const html = await renderSchema({
      root: {
        id: 'root',
        type: 'card',
        kind: 'page',
        children: [
          {
            id: 'unknown',
            type: 'card',
            kind: 'custom',
            component: {
              type: 'missingComponent',
              props: {},
            },
          },
        ],
      },
    })

    expect(html).toContain('Unknown component')
    expect(html).toContain('missingComponent')
  })

  it('keeps layout-only card children inside the configured grid container', async () => {
    const html = await renderSchema({
      root: {
        id: 'root',
        type: 'card',
        kind: 'page',
        children: [
          {
            id: 'columns',
            type: 'card',
            kind: 'layout',
            layout: { mode: 'grid', columns: 12, gap: 'lg' },
            children: [
              {
                id: 'left',
                type: 'card',
                kind: 'display',
                layout: { span: 6 },
                component: { type: 'text', props: { text: 'Left column' } },
              },
              {
                id: 'right',
                type: 'card',
                kind: 'display',
                layout: { span: 6 },
                component: { type: 'text', props: { text: 'Right column' } },
              },
            ],
          },
        ],
      },
    })

    expect(html).toContain('d-page-layout--grid')
    expect(html).toContain('--d-page-grid-columns:12')
    expect(html).toContain('grid-column:span 6')
    expect(html).toContain('Left column')
    expect(html).toContain('Right column')
  })
})

function renderSchema(schema) {
  const app = createSSRApp(DPageRenderer, { schema })
  return renderToString(app)
}
