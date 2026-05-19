<script setup>
import { computed, onErrorCaptured, onMounted, ref } from 'vue'
import DRenderError from './DRenderError.vue'
import DUnknownComponent from './DUnknownComponent.vue'

const props = defineProps({
  card: {
    type: Object,
    default: () => ({}),
  },
  runtime: {
    type: Object,
    default: null,
  },
  componentRegistry: {
    type: Object,
    default: null,
  },
  parentContext: {
    type: Object,
    default: () => ({}),
  },
})

const renderError = ref(null)
const actionError = ref(null)

onErrorCaptured((error) => {
  renderError.value = error?.message || 'Card render failed.'
  return false
})

onMounted(() => {
  executeLifecycle('onMounted')
  if (isVisible.value) {
    executeLifecycle('onVisible')
  }
})

const cardData = computed(() => resolveCardData())
const isVisible = computed(() => toVisible(resolveValue(props.card.visible, true)))
const isDisabled = computed(() => Boolean(resolveValue(props.card.disabled, false)))
const slots = computed(() => (isPlainObject(props.card.slots) ? props.card.slots : {}))
const headerCards = computed(() => normalizeCardArray(slots.value.header))
const toolbarCards = computed(() => normalizeCardArray(slots.value.toolbar))
const contentCards = computed(() => [
  ...normalizeCardArray(slots.value.body),
  ...normalizeCardArray(props.card.children),
])
const footerCards = computed(() => normalizeCardArray(slots.value.footer))
const actionCards = computed(() => normalizeCardArray(slots.value.actions))
const hasLayoutOnlyContent = computed(() => !registeredComponent.value && !props.card.component?.type && contentCards.value.length > 0)

const registeredComponentEntry = computed(() => {
  const type = props.card.component?.type
  if (!type || !props.componentRegistry?.has?.(type)) return null
  return props.componentRegistry.get(type)
})

const registeredComponent = computed(() => {
  const entry = registeredComponentEntry.value
  return entry?.component || entry || null
})

const componentProps = computed(() => {
  const entry = registeredComponentEntry.value
  const defaultProps = isPlainObject(entry?.defaultProps) ? entry.defaultProps : {}
  const configuredProps = isPlainObject(props.card.component?.props) ? props.card.component.props : {}
  const resolvedProps = resolveValue({ ...defaultProps, ...configuredProps }, {})

  if (isDisabled.value && resolvedProps.disabled == null) {
    resolvedProps.disabled = true
  }

  return applyDataDefaults(resolvedProps)
})

const eventListeners = computed(() => {
  const events = {
    ...(isPlainObject(props.card.events) ? props.card.events : {}),
    ...(isPlainObject(props.card.component?.events) ? props.card.component.events : {}),
  }

  return Object.fromEntries(
    Object.entries(events).map(([eventName, actionRef]) => [
      eventName,
      (payload) => executeConfiguredAction(actionRef, eventName, payload),
    ]),
  )
})

const wrapperClasses = computed(() => {
  if (hasLayoutOnlyContent.value) {
    return [
      'd-page-card-node',
      `d-page-card-node--${props.card.kind || 'custom'}`,
      'd-page-card-node--layout-host',
    ]
  }

  const layout = isPlainObject(props.card.layout) ? props.card.layout : {}
  const classes = [
    'd-page-card-node',
    `d-page-card-node--${props.card.kind || 'custom'}`,
    `d-page-layout--${layout.mode || 'block'}`,
    `d-page-gap--${layout.gap || 'md'}`,
  ]

  if (props.card.component?.type) {
    classes.push(`d-page-component--${props.card.component.type}`)
  }

  if (layout.class || layout.className) {
    classes.push(layout.class || layout.className)
  }

  return classes
})

const wrapperStyle = computed(() => {
  const layout = isPlainObject(props.card.layout) ? props.card.layout : {}
  const style = isPlainObject(layout.style) ? { ...layout.style } : {}

  if (layout.mode === 'grid' && layout.columns) {
    style['--d-page-grid-columns'] = String(layout.columns)
  }

  if (layout.span) {
    style.gridColumn = `span ${layout.span}`
  }

  if (layout.align) {
    style.alignItems = layout.align
  }

  return style
})

const hasChrome = computed(() => Boolean(props.card.title || props.card.description || actionError.value))
const contentClasses = computed(() => {
  if (!hasLayoutOnlyContent.value) return ['d-page-card-node__children']

  const layout = isPlainObject(props.card.layout) ? props.card.layout : {}
  return [
    'd-page-card-node__children',
    'd-page-card-node__children--layout',
    `d-page-layout--${layout.mode || 'block'}`,
    `d-page-gap--${layout.gap || 'md'}`,
  ]
})
const contentStyle = computed(() => hasLayoutOnlyContent.value ? layoutStyleWithoutSpan() : {})

function layoutStyleWithoutSpan() {
  const layout = isPlainObject(props.card.layout) ? props.card.layout : {}
  const style = isPlainObject(layout.style) ? { ...layout.style } : {}

  if (layout.mode === 'grid' && layout.columns) {
    style['--d-page-grid-columns'] = String(layout.columns)
  }

  if (layout.align) {
    style.alignItems = layout.align
  }

  return style
}

function resolveCardData() {
  const cardDataConfig = props.card.data

  if (!isPlainObject(cardDataConfig)) {
    return undefined
  }

  const defaultValue = Object.prototype.hasOwnProperty.call(cardDataConfig, 'defaultValue')
    ? cardDataConfig.defaultValue
    : undefined

  if (!cardDataConfig.bind) {
    return defaultValue
  }

  try {
    if (typeof cardDataConfig.bind === 'string' && cardDataConfig.bind.trim().startsWith('{{')) {
      return props.runtime.resolveBinding(cardDataConfig.bind, createBindingContext({}, false))
    }

    const value = props.runtime.get(cardDataConfig.bind, createBindingContext({}, false))
    return value === undefined ? defaultValue : value
  } catch (error) {
    renderError.value = error?.message || 'Card data binding failed.'
    return defaultValue
  }
}

function resolveValue(value, fallback) {
  if (value === undefined) return fallback

  try {
    return props.runtime?.resolveBinding(value, createBindingContext())
  } catch (error) {
    renderError.value = error?.message || 'Card binding failed.'
    return fallback
  }
}

function applyDataDefaults(resolvedProps) {
  const componentType = props.card.component?.type
  const value = cardData.value

  if (value !== undefined && resolvedProps.data == null) {
    resolvedProps.data = value
  }

  if (componentType === 'table' && resolvedProps.rows == null && Array.isArray(value)) {
    resolvedProps.rows = value
  }

  if ((componentType === 'text' || componentType === 'heading') && resolvedProps.text == null && isScalar(value)) {
    resolvedProps.text = value
  }

  return resolvedProps
}

async function executeConfiguredAction(actionRef, eventName, payload) {
  if (!props.runtime || isDisabled.value) return null

  actionError.value = null
  const result = await props.runtime.executeAction(actionRef, {
    context: {
      card: props.card,
      parent: props.parentContext,
      cardData: cardData.value,
      component: props.card.component || {},
      event: payload,
      eventName,
      value: payload?.value,
    },
  })

  if (!result?.ok) {
    actionError.value = result?.error?.message || 'Action failed.'
  }

  return result
}

function executeLifecycle(name) {
  const actionRef = props.card.lifecycle?.[name]
  if (!actionRef) return
  executeConfiguredAction(actionRef, name, { type: name, source: 'lifecycle' })
}

function createBindingContext(extraContext = {}, includeCardData = true) {
  return {
    context: {
      card: props.card,
      parent: props.parentContext,
      ...(includeCardData ? { cardData: cardData.value } : {}),
      ...extraContext,
    },
  }
}

function normalizeCardArray(value) {
  return Array.isArray(value) ? value.filter(Boolean) : []
}

function isPlainObject(value) {
  return Object.prototype.toString.call(value) === '[object Object]'
}

function isScalar(value) {
  return ['string', 'number', 'boolean'].includes(typeof value)
}

function toVisible(value) {
  if (value === undefined || value === null) return true
  if (typeof value === 'boolean') return value
  if (typeof value === 'string') return value.trim().length > 0
  return Boolean(value)
}
</script>

<template>
  <DRenderError v-if="renderError" :message="renderError" />
  <article
    v-else-if="isVisible"
    class="d-page-card"
    :class="wrapperClasses"
    :style="wrapperStyle"
    :data-d-page-card-id="card.id || undefined"
  >
    <header v-if="hasChrome" class="d-page-card-node__header">
      <div v-if="card.title || card.description" class="d-page-card-node__title-group">
        <h2 v-if="card.title" class="d-page-card-node__title">{{ card.title }}</h2>
        <p v-if="card.description" class="d-page-card-node__description">{{ card.description }}</p>
      </div>
      <p v-if="actionError" class="d-page-card-node__error" role="alert">{{ actionError }}</p>
    </header>

    <div v-if="headerCards.length" class="d-page-card-node__slot d-page-card-node__slot--header">
      <DCardRenderer
        v-for="child in headerCards"
        :key="child.id"
        :card="child"
        :runtime="runtime"
        :component-registry="componentRegistry"
        :parent-context="{ card, slot: 'header' }"
      />
    </div>

    <div v-if="toolbarCards.length" class="d-page-card-node__slot d-page-card-node__slot--toolbar">
      <DCardRenderer
        v-for="child in toolbarCards"
        :key="child.id"
        :card="child"
        :runtime="runtime"
        :component-registry="componentRegistry"
        :parent-context="{ card, slot: 'toolbar' }"
      />
    </div>

    <component
      :is="registeredComponent"
      v-if="registeredComponent"
      v-bind="componentProps"
      v-on="eventListeners"
    >
      <DCardRenderer
        v-for="child in contentCards"
        :key="child.id"
        :card="child"
        :runtime="runtime"
        :component-registry="componentRegistry"
        :parent-context="{ card, slot: 'default' }"
      />
    </component>

    <DUnknownComponent v-else-if="card.component?.type" :type="card.component.type" />

    <div
      v-else-if="contentCards.length"
      :class="contentClasses"
      :style="contentStyle"
    >
      <DCardRenderer
        v-for="child in contentCards"
        :key="child.id"
        :card="child"
        :runtime="runtime"
        :component-registry="componentRegistry"
        :parent-context="{ card, slot: 'default' }"
      />
    </div>

    <div v-if="footerCards.length" class="d-page-card-node__slot d-page-card-node__slot--footer">
      <DCardRenderer
        v-for="child in footerCards"
        :key="child.id"
        :card="child"
        :runtime="runtime"
        :component-registry="componentRegistry"
        :parent-context="{ card, slot: 'footer' }"
      />
    </div>

    <div v-if="actionCards.length" class="d-page-card-node__slot d-page-card-node__slot--actions">
      <DCardRenderer
        v-for="child in actionCards"
        :key="child.id"
        :card="child"
        :runtime="runtime"
        :component-registry="componentRegistry"
        :parent-context="{ card, slot: 'actions' }"
      />
    </div>
  </article>
</template>
