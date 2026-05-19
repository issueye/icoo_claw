function assertComponentType(type) {
  if (typeof type !== 'string' || type.trim().length === 0) {
    throw new TypeError('Component type must be a non-empty string.')
  }
}

function assertComponent(component, type) {
  if (!component) {
    throw new TypeError(`Component "${type}" must be provided.`)
  }
}

export function createComponentRegistry(initialComponents = {}) {
  const components = new Map()

  const registry = {
    register(type, component) {
      assertComponentType(type)
      assertComponent(component, type)
      components.set(type, component)
      return registry
    },

    get(type) {
      assertComponentType(type)
      return components.get(type)
    },

    has(type) {
      assertComponentType(type)
      return components.has(type)
    },

    list() {
      return Array.from(components.entries()).map(([type, component]) => ({ type, component }))
    },

    extend(extraComponents = {}) {
      return createComponentRegistry({
        ...Object.fromEntries(components),
        ...extraComponents,
      })
    },
  }

  Object.entries(initialComponents).forEach(([type, component]) => {
    registry.register(type, component)
  })

  return registry
}
