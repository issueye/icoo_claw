const BINDING_RE = /{{\s*([^{}]+?)\s*}}/g
const WHOLE_BINDING_RE = /^{{\s*([^{}]+?)\s*}}$/
const SAFE_PATH_RE = /^(state|data|context)(?:\.(?:[A-Za-z_][A-Za-z0-9_]*|\d+))*$/
const BLOCKED_SEGMENTS = new Set(['__proto__', 'prototype', 'constructor'])

export class BindingError extends Error {
  constructor(message, expression) {
    super(message)
    this.name = 'BindingError'
    this.expression = expression
  }
}

export function isBinding(value) {
  return typeof value === 'string' && WHOLE_BINDING_RE.test(value.trim())
}

export function resolveBinding(value, sources = {}, options = {}) {
  if (Array.isArray(value)) {
    return value.map((item) => resolveBinding(item, sources, options))
  }

  if (isPlainObject(value)) {
    return Object.fromEntries(
      Object.entries(value).map(([key, item]) => [key, resolveBinding(item, sources, options)]),
    )
  }

  if (typeof value !== 'string') {
    return value
  }

  const trimmed = value.trim()
  const wholeMatch = trimmed.match(WHOLE_BINDING_RE)

  if (wholeMatch) {
    return readSafePath(wholeMatch[1].trim(), sources, options)
  }

  return value.replace(BINDING_RE, (_match, expression) => {
    const resolved = readSafePath(expression.trim(), sources, options)
    if (resolved == null) {
      return options.missingValue ?? ''
    }
    if (typeof resolved === 'object') {
      return JSON.stringify(resolved)
    }
    return String(resolved)
  })
}

export function readSafePath(expression, sources = {}, options = {}) {
  if (!SAFE_PATH_RE.test(expression)) {
    throw new BindingError(
      `Unsupported binding expression "${expression}". Only state.xxx, data.xxx and context.xxx paths are allowed.`,
      expression,
    )
  }

  const segments = expression.split('.')
  const root = segments.shift()
  let cursor = sources[root]

  for (const segment of segments) {
    if (BLOCKED_SEGMENTS.has(segment)) {
      throw new BindingError(`Unsafe binding path segment "${segment}" is not allowed.`, expression)
    }

    if (cursor == null) {
      return options.missingValue
    }

    cursor = cursor[segment]
  }

  return cursor === undefined ? options.missingValue : cursor
}

function isPlainObject(value) {
  return Object.prototype.toString.call(value) === '[object Object]'
}
