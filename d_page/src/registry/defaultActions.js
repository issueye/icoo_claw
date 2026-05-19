export const defaultActions = {
  setState(action, runtime, eventContext = {}) {
    const payload = runtime.resolveBinding(action.payload ?? {}, eventContext)

    if (payload && typeof payload === 'object' && !Array.isArray(payload) && typeof payload.path === 'string') {
      runtime.setState(payload.path, payload.value)
      return { ok: true, type: 'setState', path: payload.path, value: payload.value }
    }

    if (!payload || typeof payload !== 'object' || Array.isArray(payload)) {
      return actionError('INVALID_PAYLOAD', 'setState payload must be an object or a { path, value } pair.')
    }

    runtime.setState(payload)
    return { ok: true, type: 'setState', state: runtime.state }
  },

  emit(action, runtime, eventContext = {}) {
    const payload = runtime.resolveBinding(action.payload ?? {}, eventContext)
    const eventName = payload.event || payload.name || action.event || action.name

    if (!eventName) {
      return actionError('INVALID_PAYLOAD', 'emit action requires payload.event or payload.name.')
    }

    const eventPayload = Object.prototype.hasOwnProperty.call(payload, 'payload') ? payload.payload : payload
    runtime.emit(eventName, eventPayload, eventContext)
    return { ok: true, type: 'emit', event: eventName, payload: eventPayload }
  },

  async copyText(action, runtime, eventContext = {}) {
    const payload = runtime.resolveBinding(action.payload ?? {}, eventContext)
    const text = typeof payload === 'string' ? payload : payload.text

    if (typeof text !== 'string') {
      return actionError('INVALID_PAYLOAD', 'copyText action requires text.')
    }

    if (typeof runtime.adapters.copyText !== 'function') {
      return actionError('ADAPTER_MISSING', 'copyText adapter is not available in this runtime.')
    }

    await runtime.adapters.copyText(text)
    return { ok: true, type: 'copyText', text }
  },

  async openUrl(action, runtime, eventContext = {}) {
    const payload = runtime.resolveBinding(action.payload ?? {}, eventContext)
    const url = typeof payload === 'string' ? payload : payload.url
    const target = typeof payload === 'object' && payload ? payload.target : undefined

    if (typeof url !== 'string' || !url.trim()) {
      return actionError('INVALID_PAYLOAD', 'openUrl action requires url.')
    }

    if (typeof runtime.adapters.openUrl !== 'function') {
      return actionError('ADAPTER_MISSING', 'openUrl adapter is not available in this runtime.')
    }

    await runtime.adapters.openUrl(url, { target })
    return { ok: true, type: 'openUrl', url, target }
  },

  async chain(action, runtime, eventContext = {}) {
    const actions = action.actions ?? action.payload?.actions ?? action.payload

    if (!Array.isArray(actions)) {
      return actionError('INVALID_PAYLOAD', 'chain action requires an actions array.')
    }

    const results = []

    for (const childAction of actions) {
      const result = await runtime.executeAction(childAction, eventContext)
      results.push(result)

      if (!result.ok) {
        return { ok: false, type: 'chain', results, error: result.error }
      }
    }

    return { ok: true, type: 'chain', results }
  },
}

function actionError(code, message) {
  return {
    ok: false,
    error: { code, message },
  }
}
