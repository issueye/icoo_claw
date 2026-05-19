<script setup>
import { computed, reactive } from 'vue'
import DCardRenderer from './DCardRenderer.vue'
import DRenderError from './DRenderError.vue'
import { createComponentRegistry } from '../registry/createComponentRegistry.js'
import { defaultComponents } from '../registry/defaultComponents.js'
import { createDPageRuntime } from '../runtime/createDPageRuntime.js'

const props = defineProps({
  schema: {
    type: Object,
    default: () => ({}),
  },
  runtime: {
    type: Object,
    default: null,
  },
  context: {
    type: Object,
    default: () => ({}),
  },
  state: {
    type: Object,
    default: () => ({}),
  },
  data: {
    type: Object,
    default: () => ({}),
  },
  actions: {
    type: Object,
    default: () => ({}),
  },
  adapters: {
    type: Object,
    default: () => ({}),
  },
  componentRegistry: {
    type: Object,
    default: null,
  },
})

const emit = defineEmits(['runtimeEmit'])

const fallbackRegistry = createComponentRegistry(defaultComponents)

const activeRegistry = computed(() => props.componentRegistry || fallbackRegistry)

const activeRuntime = computed(() => {
  const runtime = props.runtime || createDPageRuntime({
    schema: props.schema,
    context: props.context,
    state: props.state,
    data: props.data,
    actions: props.actions,
    adapters: props.adapters,
    onEmit: (event) => emit('runtimeEmit', event),
  })

  return makeRuntimeReactive(runtime)
})

const fullValidation = computed(() => activeRuntime.value.validateSchema({
  componentRegistry: activeRegistry.value,
}))

const blockingValidationErrors = computed(() => fullValidation.value.errors.filter(
  (error) => error.code !== 'component.registration',
))

const validation = computed(() => ({
  ...fullValidation.value,
  valid: blockingValidationErrors.value.length === 0,
  errors: blockingValidationErrors.value,
}))

const validationMessage = computed(() => {
  if (validation.value.valid) return ''
  return validation.value.errors
    .map((error) => `${error.path || 'schema'}: ${error.message}`)
    .join('\n')
})

function makeRuntimeReactive(runtime) {
  if (!runtime || typeof runtime !== 'object') return runtime

  runtime.state = reactive(runtime.state || {})
  runtime.data = reactive(runtime.data || {})
  runtime.context = reactive({
    ...(runtime.context || {}),
    ...props.context,
  })

  return runtime
}
</script>

<template>
  <section class="d-page-root" :data-d-page-id="activeRuntime.page?.id || undefined">
    <DRenderError v-if="!validation.valid" :message="validationMessage" />
    <DCardRenderer
      v-else-if="activeRuntime.schema.root"
      :card="activeRuntime.schema.root"
      :runtime="activeRuntime"
      :component-registry="activeRegistry"
    />
    <slot v-else name="placeholder">d_page renderer is initializing.</slot>
  </section>
</template>
