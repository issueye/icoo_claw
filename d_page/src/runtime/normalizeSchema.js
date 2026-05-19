const DEFAULT_SCHEMA_VERSION = '0.1.0'

export function normalizeSchema(schema = {}) {
  if (!isPlainObject(schema)) {
    return {
      schemaVersion: DEFAULT_SCHEMA_VERSION,
      page: {},
      state: {},
      data: {},
      actions: {},
      root: null,
    }
  }

  return {
    ...schema,
    schemaVersion: schema.schemaVersion || DEFAULT_SCHEMA_VERSION,
    page: isPlainObject(schema.page) ? clone(schema.page) : {},
    state: isPlainObject(schema.state) ? clone(schema.state) : {},
    data: isPlainObject(schema.data) ? clone(schema.data) : {},
    actions: isPlainObject(schema.actions) ? clone(schema.actions) : {},
    root: schema.root ? normalizeCard(schema.root) : null,
  }
}

export function normalizeCard(card) {
  if (!isPlainObject(card)) {
    return card
  }

  const normalized = {
    ...card,
    children: Array.isArray(card.children) ? card.children.map(normalizeCard) : [],
    slots: normalizeSlots(card.slots),
    events: isPlainObject(card.events) ? clone(card.events) : {},
  }

  if (isPlainObject(card.component)) {
    normalized.component = {
      ...card.component,
      props: isPlainObject(card.component.props) ? clone(card.component.props) : {},
      events: isPlainObject(card.component.events) ? clone(card.component.events) : {},
    }
  }

  return normalized
}

function normalizeSlots(slots) {
  if (!isPlainObject(slots)) {
    return {}
  }

  return Object.fromEntries(
    Object.entries(slots).map(([name, cards]) => [
      name,
      Array.isArray(cards) ? cards.map(normalizeCard) : cards,
    ]),
  )
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function isPlainObject(value) {
  return Object.prototype.toString.call(value) === '[object Object]'
}
