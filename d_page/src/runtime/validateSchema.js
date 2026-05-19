import { normalizeSchema } from './normalizeSchema.js'

const ALLOWED_CARD_KINDS = new Set(['page', 'layout', 'form', 'data', 'display', 'action', 'custom'])

export function validateSchema(schema, options = {}) {
  const normalized = normalizeSchema(schema)
  const errors = []
  const seenIds = new Set()

  if (!schema || typeof schema !== 'object' || Array.isArray(schema)) {
    errors.push(createError('schema.type', 'Schema must be an object.'))
    return buildResult(normalized, errors, options)
  }

  if (!normalized.root) {
    errors.push(createError('root.required', 'Schema root is required.', 'root'))
    return buildResult(normalized, errors, options)
  }

  validateCard(schema.root, 'root', normalized, options, errors, seenIds)

  return buildResult(normalized, errors, options)
}

function validateCard(card, path, schema, options, errors, seenIds) {
  if (!card || typeof card !== 'object' || Array.isArray(card)) {
    errors.push(createError('card.type', 'Card must be an object.', path))
    return
  }

  if (card.type !== 'card') {
    errors.push(createError('card.type', 'Card type must be "card".', `${path}.type`))
  }

  if (typeof card.id !== 'string' || !card.id.trim()) {
    errors.push(createError('card.id.required', 'Card id is required.', `${path}.id`))
  } else if (seenIds.has(card.id)) {
    errors.push(createError('card.id.unique', `Card id "${card.id}" must be unique.`, `${path}.id`))
  } else {
    seenIds.add(card.id)
  }

  if (card.kind != null && !ALLOWED_CARD_KINDS.has(card.kind)) {
    errors.push(createError('card.kind', `Card kind "${card.kind}" is not supported.`, `${path}.kind`))
  }

  if (card.children != null && !Array.isArray(card.children)) {
    errors.push(createError('card.children', 'Card children must be an array.', `${path}.children`))
  }

  if (card.slots != null && (!isPlainObject(card.slots) || !slotValuesAreArrays(card.slots))) {
    errors.push(createError('card.slots', 'Card slots must be an object whose values are card arrays.', `${path}.slots`))
  }

  validateComponent(card.component, `${path}.component`, options, errors)
  validateActionRefs(card.events, `${path}.events`, schema, errors)
  validateActionRefs(card.component?.events, `${path}.component.events`, schema, errors)

  if (Array.isArray(card.children)) {
    card.children.forEach((child, index) => {
      validateCard(child, `${path}.children.${index}`, schema, options, errors, seenIds)
    })
  }

  if (isPlainObject(card.slots)) {
    Object.entries(card.slots).forEach(([slotName, cards]) => {
      if (!Array.isArray(cards)) {
        return
      }

      cards.forEach((child, index) => {
        validateCard(child, `${path}.slots.${slotName}.${index}`, schema, options, errors, seenIds)
      })
    })
  }
}

function validateComponent(component, path, options, errors) {
  if (component == null) {
    return
  }

  if (!isPlainObject(component)) {
    errors.push(createError('component.type', 'Component must be an object.', path))
    return
  }

  if (typeof component.type !== 'string' || !component.type.trim()) {
    errors.push(createError('component.type.required', 'Component type is required.', `${path}.type`))
    return
  }

  if (options.componentRegistry && !hasRegisteredComponent(options.componentRegistry, component.type)) {
    errors.push(
      createError(
        'component.registration',
        `Component "${component.type}" is not registered.`,
        `${path}.type`,
      ),
    )
  }
}

function validateActionRefs(events, path, schema, errors) {
  if (events == null) {
    return
  }

  if (!isPlainObject(events)) {
    errors.push(createError('events.type', 'Events must be an object.', path))
    return
  }

  Object.entries(events).forEach(([eventName, actionRef]) => {
    collectActionRefs(actionRef).forEach((id) => {
      if (!schema.actions || !schema.actions[id]) {
        errors.push(
          createError(
            'events.action.missing',
            `Event "${eventName}" references missing action "${id}".`,
            `${path}.${eventName}`,
          ),
        )
      }
    })
  })
}

function collectActionRefs(actionRef) {
  if (typeof actionRef === 'string') {
    return [actionRef]
  }

  if (Array.isArray(actionRef)) {
    return actionRef.flatMap(collectActionRefs)
  }

  return []
}

function slotValuesAreArrays(slots) {
  return Object.values(slots).every(Array.isArray)
}

function hasRegisteredComponent(registry, type) {
  if (typeof registry.has === 'function') {
    return registry.has(type)
  }

  if (Array.isArray(registry)) {
    return registry.includes(type)
  }

  return Boolean(registry[type])
}

function buildResult(schema, errors, options) {
  const result = {
    valid: errors.length === 0,
    errors,
    schema,
  }

  if (!result.valid && options.throwOnError) {
    const error = new Error(errors.map((item) => item.message).join('\n'))
    error.name = 'SchemaValidationError'
    error.errors = errors
    throw error
  }

  return result
}

function createError(code, message, path = '') {
  return { code, message, path }
}

function isPlainObject(value) {
  return Object.prototype.toString.call(value) === '[object Object]'
}
