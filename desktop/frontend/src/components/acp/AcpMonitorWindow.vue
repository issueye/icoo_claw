<script setup>
import { computed, ref, watch } from 'vue'
import { ArrowDownLeft, ArrowUpRight, Ban, Copy, Pause, Play, ShieldAlert, Trash2, X } from 'lucide-vue-next'
import QqButton from '@/components/ued/QqButton.vue'
import { useAcpMonitorStore } from '@/stores/acpMonitor'

const monitorStore = useAcpMonitorStore()
const selectedId = ref('')
const directionFilter = ref('all')
const typeFilter = ref('all')

const eventTypes = computed(() => {
  const types = new Set(monitorStore.events.map((event) => event.type).filter(Boolean))
  return ['all', ...Array.from(types).sort()]
})

const filteredEvents = computed(() => monitorStore.events.filter((event) => {
  if (directionFilter.value !== 'all' && event.direction !== directionFilter.value) {
    return false
  }
  if (typeFilter.value !== 'all' && event.type !== typeFilter.value) {
    return false
  }
  return true
}))

const selectedEvent = computed(() => {
  if (!filteredEvents.value.length) {
    return null
  }
  return filteredEvents.value.find((event) => event.id === selectedId.value) || filteredEvents.value[0]
})

const selectedPayload = computed(() => {
  if (!selectedEvent.value) {
    return ''
  }
  try {
    return JSON.stringify(selectedEvent.value.payload, null, 2)
  } catch {
    return String(selectedEvent.value.payload)
  }
})

watch(
  filteredEvents,
  (events) => {
    if (!events.length) {
      selectedId.value = ''
      return
    }
    if (!events.some((event) => event.id === selectedId.value)) {
      selectedId.value = events[0].id
    }
  },
  { immediate: true },
)

function close() {
  monitorStore.setOpen(false)
}

function selectEvent(id) {
  selectedId.value = id
}

function timeLabel(time) {
  return new Date(time).toLocaleTimeString()
}

function directionLabel(direction) {
  return direction === 'outbound' ? '发送' : '接收'
}

function directionIcon(direction) {
  return direction === 'outbound' ? ArrowUpRight : ArrowDownLeft
}

function typeToneClass(type) {
  if (type === 'session/request_permission' || type === 'chat.permission_decision') {
    return 'text-amber-200'
  }
  if (type === 'session/error') {
    return 'text-rose-200'
  }
  if (type === 'session/completed') {
    return 'text-emerald-200'
  }
  return 'text-[color:var(--qq-text-primary)]'
}

async function copyPayload() {
  if (!selectedPayload.value) {
    return
  }
  await navigator.clipboard?.writeText(selectedPayload.value)
}
</script>

<template>
  <aside
    v-if="monitorStore.open"
    class="acp-monitor qq-panel-strong fixed right-4 top-16 z-[190] flex h-[min(720px,calc(100vh-6rem))] w-[min(1040px,calc(100vw-2rem))] flex-col rounded-[8px] border border-[color:var(--qq-border-strong)] shadow-[0_24px_90px_rgba(0,0,0,0.45)]"
  >
    <header class="flex h-12 shrink-0 items-center justify-between border-b border-[color:var(--qq-border)] px-4">
      <div class="flex min-w-0 items-center gap-3">
        <span class="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-[6px] border border-[color:var(--qq-border)] bg-[var(--qq-fill-medium)] text-[color:var(--qq-accent)]">
          <ShieldAlert class="h-4 w-4" />
        </span>
        <div class="min-w-0">
          <h2 class="truncate text-sm font-semibold text-[color:var(--qq-text-primary)]">ACP 事件监控</h2>
          <p class="truncate text-xs text-[color:var(--qq-text-tertiary)]">
            {{ monitorStore.total }} 条事件 · 接收 {{ monitorStore.inboundCount }} · 发送 {{ monitorStore.outboundCount }} · 权限 {{ monitorStore.permissionCount }}
          </p>
        </div>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <QqButton variant="ghost" size="sm" @click="monitorStore.togglePaused">
          <component :is="monitorStore.paused ? Play : Pause" class="h-4 w-4" />
          {{ monitorStore.paused ? '继续' : '暂停' }}
        </QqButton>
        <QqButton variant="ghost" size="sm" @click="monitorStore.clear">
          <Trash2 class="h-4 w-4" />
          清空
        </QqButton>
        <button
          class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] text-[color:var(--qq-text-tertiary)] hover:bg-[var(--qq-fill-soft)] hover:text-[color:var(--qq-text-primary)]"
          type="button"
          title="关闭监控"
          @click="close"
        >
          <X class="h-4 w-4" />
        </button>
      </div>
    </header>

    <div class="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[minmax(24rem,0.95fr)_minmax(0,1.15fr)]">
      <section class="flex min-h-0 flex-col border-b border-[color:var(--qq-border)] lg:border-b-0 lg:border-r">
        <div class="grid shrink-0 grid-cols-2 gap-2 border-b border-[color:var(--qq-border)] p-3">
          <select v-model="directionFilter" class="acp-select">
            <option value="all">全部方向</option>
            <option value="inbound">接收</option>
            <option value="outbound">发送</option>
          </select>
          <select v-model="typeFilter" class="acp-select">
            <option v-for="type in eventTypes" :key="type" :value="type">{{ type === 'all' ? '全部类型' : type }}</option>
          </select>
        </div>

        <div v-if="filteredEvents.length" class="min-h-0 flex-1 overflow-auto p-2">
          <button
            v-for="event in filteredEvents"
            :key="event.id"
            class="event-row flex w-full min-w-0 items-start gap-3 rounded-[6px] border px-3 py-2 text-left transition"
            :class="selectedEvent?.id === event.id ? 'is-active' : ''"
            type="button"
            @click="selectEvent(event.id)"
          >
            <span class="mt-0.5 inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-[4px] border border-[color:var(--qq-border)] bg-[var(--qq-fill-soft)] text-[color:var(--qq-accent)]">
              <component :is="directionIcon(event.direction)" class="h-4 w-4" />
            </span>
            <span class="min-w-0 flex-1">
              <span class="flex min-w-0 items-center gap-2">
                <span class="truncate text-xs font-semibold" :class="typeToneClass(event.type)">{{ event.type }}</span>
                <span class="shrink-0 rounded-[4px] border border-[color:var(--qq-border)] px-1.5 py-0.5 text-[10px] text-[color:var(--qq-text-tertiary)]">{{ directionLabel(event.direction) }}</span>
              </span>
              <span class="mt-1 block truncate text-xs text-[color:var(--qq-text-secondary)]">{{ event.summary }}</span>
              <span class="mt-1 block truncate text-[10px] text-[color:var(--qq-text-tertiary)]">{{ timeLabel(event.time) }} · {{ event.conversationId || '-' }} · {{ event.requestId || '-' }}</span>
            </span>
          </button>
        </div>

        <div v-else class="flex min-h-0 flex-1 items-center justify-center p-6 text-center text-sm text-[color:var(--qq-text-tertiary)]">
          <div>
            <Ban class="mx-auto mb-3 h-7 w-7 opacity-70" />
            暂无匹配的 ACP 事件
          </div>
        </div>
      </section>

      <section class="flex min-h-0 flex-col">
        <div class="flex h-12 shrink-0 items-center justify-between border-b border-[color:var(--qq-border)] px-4">
          <div class="min-w-0">
            <p class="truncate text-sm font-semibold text-[color:var(--qq-text-primary)]">{{ selectedEvent?.type || '未选择事件' }}</p>
            <p class="truncate text-xs text-[color:var(--qq-text-tertiary)]">
              {{ selectedEvent ? `${directionLabel(selectedEvent.direction)} · ${timeLabel(selectedEvent.time)} · ${selectedEvent.source}` : '选择左侧事件查看完整 payload' }}
            </p>
          </div>
          <QqButton v-if="selectedEvent" variant="ghost" size="sm" @click="copyPayload">
            <Copy class="h-4 w-4" />
            复制
          </QqButton>
        </div>

        <div v-if="selectedEvent" class="grid shrink-0 grid-cols-3 gap-2 border-b border-[color:var(--qq-border)] p-3 text-xs">
          <div class="rounded-[6px] border border-[color:var(--qq-border)] bg-[var(--qq-fill-subtle)] p-2">
            <p class="text-[color:var(--qq-text-tertiary)]">Conversation</p>
            <p class="mt-1 truncate text-[color:var(--qq-text-primary)]">{{ selectedEvent.conversationId || '-' }}</p>
          </div>
          <div class="rounded-[6px] border border-[color:var(--qq-border)] bg-[var(--qq-fill-subtle)] p-2">
            <p class="text-[color:var(--qq-text-tertiary)]">Session</p>
            <p class="mt-1 truncate text-[color:var(--qq-text-primary)]">{{ selectedEvent.sessionId || '-' }}</p>
          </div>
          <div class="rounded-[6px] border border-[color:var(--qq-border)] bg-[var(--qq-fill-subtle)] p-2">
            <p class="text-[color:var(--qq-text-tertiary)]">Request</p>
            <p class="mt-1 truncate text-[color:var(--qq-text-primary)]">{{ selectedEvent.requestId || '-' }}</p>
          </div>
        </div>

        <pre class="min-h-0 flex-1 overflow-auto p-4 text-xs leading-5 text-[color:var(--qq-text-secondary)]">{{ selectedPayload || '暂无 payload' }}</pre>
      </section>
    </div>
  </aside>
</template>

<style scoped>
.acp-monitor {
  backdrop-filter: blur(28px);
}

.acp-select {
  height: 2rem;
  min-width: 0;
  border-radius: 4px;
  border: 1px solid var(--qq-border);
  background: var(--qq-fill-soft);
  color: var(--qq-text-primary);
  padding: 0 0.6rem;
  font-size: 0.75rem;
  outline: none;
}

.acp-select:focus {
  border-color: color-mix(in srgb, var(--qq-accent) 52%, var(--qq-border));
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--qq-accent) 18%, transparent);
}

.event-row {
  border-color: transparent;
  background: transparent;
}

.event-row:hover {
  border-color: var(--qq-border);
  background: var(--qq-fill-subtle);
}

.event-row.is-active {
  border-color: color-mix(in srgb, var(--qq-accent) 42%, var(--qq-border));
  background: color-mix(in srgb, var(--qq-accent) 10%, transparent);
}
</style>
