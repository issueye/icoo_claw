import { createActionRegistry } from '../registry/createActionRegistry.js'
import { defaultActions } from '../registry/defaultActions.js'
import { executeAction } from './executeAction.js'
import { normalizeSchema } from './normalizeSchema.js'
import { resolveBinding, readSafePath } from './resolveBinding.js'
import { validateSchema } from './validateSchema.js'

export function createDPageRuntime(options = {}) {
  const schema = normalizeSchema(options.schema || {})
  const listeners = new Map()
  const emitted = []
  const actionRegistry = options.actionRegistry || createActionRegistry(defaultActions)

  if (options.actions) {
    Object.entries(options.actions).forEach(([type, handler]) => {
      actionRegistry.register(type, handler)
    })
  }

  const runtime = {
    schema,
    actionDefinitions: schema.actions,
    actionRegistry,
    adapters: options.adapters || {},
    page: schema.page,
    state: mergeObjects(schema.state, options.state),
    data: mergeObjects(schema.data, options.data),
    context: mergeObjects(options.context, {}),
    emitted,

    get(path, extraSources = {}) {
      return readSafePath(path, runtime.createBindingSources(extraSources))
    },

    setState(pathOrPatch, value) {
      if (typeof pathOrPatch === 'string') {
        setByPath(runtime.state, stripStatePrefix(pathOrPatch), value)
        return runtime.state
      }

      if (isPlainObject(pathOrPatch)) {
        Object.entries(pathOrPatch).forEach(([key, patchValue]) => {
          if (key.includes('.')) {
            setByPath(runtime.state, stripStatePrefix(key), patchValue)
          } else {
            runtime.state[key] = patchValue
          }
        })
      }

      return runtime.state
    },

    setData(pathOrPatch, value) {
      if (typeof pathOrPatch === 'string') {
        setByPath(runtime.data, stripDataPrefix(pathOrPatch), value)
        return runtime.data
      }

      if (isPlainObject(pathOrPatch)) {
        Object.assign(runtime.data, pathOrPatch)
      }

      return runtime.data
    },

    resolveBinding(valueToResolve, extraSources = {}) {
      return resolveBinding(valueToResolve, runtime.createBindingSources(extraSources))
    },

    createBindingSources(extraSources = {}) {
      return {
        state: runtime.state,
        data: runtime.data,
        context: {
          ...runtime.context,
          ...(extraSources.context || {}),
        },
      }
    },

    executeAction(actionRef, eventContext = {}) {
      return executeAction(actionRef, runtime, eventContext)
    },

    validateSchema(validateOptions = {}) {
      return validateSchema(schema, validateOptions)
    },

    emit(eventName, payload, eventContext = {}) {
      const event = { event: eventName, payload, context: eventContext }
      emitted.push(event)

      if (typeof options.onEmit === 'function') {
        options.onEmit(event)
      }

      const handlers = listeners.get(eventName) || []
      handlers.forEach((handler) => handler(event))
      return event
    },

    on(eventName, handler) {
      if (!listeners.has(eventName)) {
        listeners.set(eventName, new Set())
      }

      listeners.get(eventName).add(handler)
      return () => runtime.off(eventName, handler)
    },

    off(eventName, handler) {
      return listeners.get(eventName)?.delete(handler) || false
    },
  }

  return runtime
}

function setByPath(target, path, value) {
  const segments = path.split('.').filter(Boolean)
  let cursor = target

  segments.forEach((segment, index) => {
    if (['__proto__', 'prototype', 'constructor'].includes(segment)) {
      throw new Error(`Unsafe state path segment "${segment}" is not allowed.`)
    }

    if (index === segments.length - 1) {
      cursor[segment] = value
      return
    }

    if (!cursor[segment] || typeof cursor[segment] !== 'object') {
      cursor[segment] = {}
    }

    cursor = cursor[segment]
  })
}

function stripStatePrefix(path) {
  return path.startsWith('state.') ? path.slice('state.'.length) : path
}

function stripDataPrefix(path) {
  return path.startsWith('data.') ? path.slice('data.'.length) : path
}

function mergeObjects(base = {}, override = {}) {
  return {
    ...(isPlainObject(base) ? clone(base) : {}),
    ...(isPlainObject(override) ? clone(override) : {}),
  }
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function isPlainObject(value) {
  return Object.prototype.toString.call(value) === '[object Object]'
}
