export function createActionRegistry(initialActions = {}) {
  const actions = new Map()

  const api = {
    register,
    unregister,
    get,
    has,
    list,
  }

  Object.entries(initialActions).forEach(([type, handler]) => {
    register(type, handler)
  })

  function register(type, handler) {
    if (typeof type !== 'string' || !type.trim()) {
      throw new TypeError('Action type must be a non-empty string.')
    }

    if (typeof handler !== 'function') {
      throw new TypeError(`Action handler for "${type}" must be a function.`)
    }

    actions.set(type, handler)
    return api
  }

  function unregister(type) {
    return actions.delete(type)
  }

  function get(type) {
    return actions.get(type)
  }

  function has(type) {
    return actions.has(type)
  }

  function list() {
    return Array.from(actions.keys())
  }

  return api
}
