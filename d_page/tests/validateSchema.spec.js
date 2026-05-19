import { describe, expect, it } from 'vitest'
import { validateSchema } from '../src/runtime/validateSchema.js'

function createRegistry(names) {
  return {
    has(type) {
      return names.includes(type)
    },
  }
}

function createSchema(overrides = {}) {
  return {
    schemaVersion: '0.1.0',
    state: { title: 'Demo' },
    actions: {
      refresh: { type: 'emit', payload: { event: 'refresh' } },
    },
    root: {
      id: 'root',
      type: 'card',
      kind: 'page',
      component: {
        type: 'text',
        props: { text: '{{ state.title }}' },
        events: { click: 'refresh' },
      },
      children: [
        {
          id: 'child',
          type: 'card',
          kind: 'display',
          component: { type: 'heading', props: { text: 'Child' } },
          children: [],
        },
      ],
      slots: {
        footer: [
          {
            id: 'footer',
            type: 'card',
            kind: 'action',
            component: { type: 'button', props: { label: 'OK' } },
            children: [],
          },
        ],
      },
    },
    ...overrides,
  }
}

describe('validateSchema', () => {
  it('accepts a valid card tree schema', () => {
    const result = validateSchema(createSchema(), {
      componentRegistry: createRegistry(['text', 'heading', 'button']),
    })

    expect(result.valid).toBe(true)
    expect(result.errors).toEqual([])
    expect(result.schema.root.children).toHaveLength(1)
  })

  it('requires root and card type', () => {
    const missingRoot = validateSchema({ schemaVersion: '0.1.0' })
    expect(missingRoot.valid).toBe(false)
    expect(missingRoot.errors.map((error) => error.code)).toContain('root.required')

    const invalidType = validateSchema(createSchema({ root: { id: 'root', type: 'container', children: [] } }))
    expect(invalidType.valid).toBe(false)
    expect(invalidType.errors.map((error) => error.code)).toContain('card.type')
  })

  it('reports duplicate card ids', () => {
    const schema = createSchema()
    schema.root.children[0].id = 'root'

    const result = validateSchema(schema)

    expect(result.valid).toBe(false)
    expect(result.errors.map((error) => error.code)).toContain('card.id.unique')
  })

  it('reports invalid children or slots', () => {
    const schema = createSchema()
    schema.root.slots.footer = { bad: true }
    schema.root.children = 'not-an-array'

    const result = validateSchema(schema)

    expect(result.valid).toBe(false)
    expect(result.errors.map((error) => error.code)).toEqual(
      expect.arrayContaining(['card.children', 'card.slots']),
    )
  })

  it('checks component registration and event action references', () => {
    const schema = createSchema({ actions: {} })
    const result = validateSchema(schema, {
      componentRegistry: createRegistry(['heading', 'button']),
    })

    expect(result.valid).toBe(false)
    expect(result.errors.map((error) => error.code)).toEqual(
      expect.arrayContaining(['component.registration', 'events.action.missing']),
    )
  })
})
