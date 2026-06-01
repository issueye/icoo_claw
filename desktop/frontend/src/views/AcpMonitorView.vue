<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AcpMonitorWindow from '@/components/acp/AcpMonitorWindow.vue'
import { AcpEventBusMonitorClient, eventBusEventToMonitorInput } from '@/services/acp/event-bus-monitor'
import { useAcpMonitorStore } from '@/stores/acpMonitor'
import { useSettingsStore } from '@/stores/settings'

const settingsStore = useSettingsStore()
const monitorStore = useAcpMonitorStore()
const connectionState = ref('idle')
const errorMessage = ref('')
let client = null

const gatewayBaseUrl = computed(() => settingsStore.settings.gateway?.baseUrl || '')
const connectionLabel = computed(() => {
  if (!gatewayBaseUrl.value) {
    return '事件总线未配置'
  }
  if (errorMessage.value) {
    return `事件总线 ${connectionState.value}，${errorMessage.value}`
  }
  return `事件总线 ${connectionState.value}`
})

onMounted(async () => {
  monitorStore.setOpen(true)
  if (!settingsStore.loaded) {
    await settingsStore.load()
  }
  connect()
})

onBeforeUnmount(() => {
  disconnect()
})

watch(
  gatewayBaseUrl,
  () => {
    connect()
  },
)

function connect() {
  disconnect()
  errorMessage.value = ''
  if (!gatewayBaseUrl.value) {
    connectionState.value = 'unconfigured'
    return
  }
  client = new AcpEventBusMonitorClient(gatewayBaseUrl.value, {
    onStateChange: (state) => {
      connectionState.value = state
    },
    onError: (error) => {
      errorMessage.value = error?.message || String(error)
    },
    onEvent: (event) => {
      monitorStore.record(eventBusEventToMonitorInput(event))
    },
  })
  client.connect()
}

function disconnect() {
  client?.close()
  client = null
}
</script>

<template>
  <main class="qq-theme qq-mesh h-screen overflow-hidden text-[color:var(--qq-text-primary)]">
    <div class="relative z-10 h-full">
      <AcpMonitorWindow standalone :connection-label="connectionLabel" />
    </div>
  </main>
</template>
