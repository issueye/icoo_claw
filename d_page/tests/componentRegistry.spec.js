import { describe, expect, it } from 'vitest'
import { createComponentRegistry } from '../src/registry/createComponentRegistry.js'
import { defaultComponents } from '../src/registry/defaultComponents.js'

describe('createComponentRegistry', () => {
  it('registers and reads components', () => {
    const TextComponent = { name: 'TextComponent' }
    const registry = createComponentRegistry()

    registry.register('text', TextComponent)

    expect(registry.has('text')).toBe(true)
    expect(registry.get('text')).toBe(TextComponent)
    expect(registry.list()).toEqual([{ type: 'text', component: TextComponent }])
  })

  it('allows an existing component registration to be overwritten', () => {
    const OriginalButton = { name: 'OriginalButton' }
    const ReplacementButton = { name: 'ReplacementButton' }
    const registry = createComponentRegistry({ button: OriginalButton })

    registry.register('button', ReplacementButton)

    expect(registry.get('button')).toBe(ReplacementButton)
    expect(registry.list()).toHaveLength(1)
  })

  it('extends a registry without mutating the original registry', () => {
    const TextComponent = { name: 'TextComponent' }
    const ButtonComponent = { name: 'ButtonComponent' }
    const registry = createComponentRegistry({ text: TextComponent })

    const extendedRegistry = registry.extend({ button: ButtonComponent })

    expect(registry.has('button')).toBe(false)
    expect(extendedRegistry.get('text')).toBe(TextComponent)
    expect(extendedRegistry.get('button')).toBe(ButtonComponent)
  })

  it('includes the default MVP component list', () => {
    expect(Object.keys(defaultComponents).sort()).toEqual([
      'alert',
      'button',
      'cardSurface',
      'checkbox',
      'divider',
      'heading',
      'image',
      'input',
      'list',
      'select',
      'stat',
      'switch',
      'table',
      'tag',
      'text',
      'textarea',
    ])

    const registry = createComponentRegistry(defaultComponents)

    expect(registry.list().map((entry) => entry.type).sort()).toEqual([
      'alert',
      'button',
      'cardSurface',
      'checkbox',
      'divider',
      'heading',
      'image',
      'input',
      'list',
      'select',
      'stat',
      'switch',
      'table',
      'tag',
      'text',
      'textarea',
    ])
  })
})
