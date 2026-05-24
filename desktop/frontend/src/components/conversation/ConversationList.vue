<script setup>
import { LoaderCircle, MessageSquarePlus, PanelLeftClose, PanelLeftOpen, RefreshCw, Trash2 } from 'lucide-vue-next'

defineProps({
  conversations: {
    type: Array,
    default: () => [],
  },
  activeId: {
    type: String,
    default: '',
  },
  loading: {
    type: Boolean,
    default: false,
  },
  streaming: {
    type: Boolean,
    default: false,
  },
  runningConversationIds: {
    type: Array,
    default: () => [],
  },
  deletingId: {
    type: String,
    default: '',
  },
  collapsed: {
    type: Boolean,
    default: false,
  },
})

defineEmits(['delete', 'new-chat', 'refresh', 'toggle-collapse'])

function formatTime(value) {
  if (!value) return ''
  return new Date(value).toLocaleDateString()
}

function isRunning(conversation, runningConversationIds) {
  return Boolean(conversation?.id && runningConversationIds.includes(conversation.id))
}
</script>

<template>
  <aside
    class="qq-panel-strong min-h-0 shrink-0 flex-col overflow-hidden border-r border-white/10 transition-[width] duration-200 ease-out"
    :class="collapsed ? 'w-12' : 'w-80'"
  >
    <div v-if="collapsed" class="flex min-h-0 flex-1 flex-col items-center gap-2 px-2 py-3">
      <button
        class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.08)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[rgba(255,255,255,0.12)] hover:text-white"
        type="button"
        title="展开会话列表"
        @click="$emit('toggle-collapse')"
      >
        <PanelLeftOpen class="h-4 w-4" />
      </button>
      <button
        class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] bg-[linear-gradient(135deg,var(--qq-accent),var(--qq-accent-strong))] text-slate-950 transition hover:brightness-105"
        type="button"
        title="新建会话"
        @click="$emit('new-chat')"
      >
        <MessageSquarePlus class="h-4 w-4" />
      </button>
      <button
        class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.08)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[rgba(255,255,255,0.12)] hover:text-white"
        type="button"
        title="刷新会话"
        @click="$emit('refresh')"
      >
        <RefreshCw class="h-4 w-4" />
      </button>
      <div class="mt-2 flex flex-col items-center gap-2 text-[11px] text-[color:var(--qq-text-tertiary)]">
        <span class="rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.08)] px-2 py-1 text-[color:var(--qq-text-secondary)]">
          {{ conversations.length }}
        </span>
        <span
          v-if="streaming"
          class="inline-flex h-6 w-6 items-center justify-center rounded-full bg-[rgba(72,255,209,0.14)] text-[color:var(--qq-accent)]"
          title="有会话正在运行"
        >
          <LoaderCircle class="h-3.5 w-3.5 animate-spin" />
        </span>
      </div>
    </div>

    <template v-else>
    <div class="flex items-center justify-between border-b border-white/10 px-4 py-3">
      <div>
        <p class="text-xs uppercase tracking-[0.18em] text-[color:var(--qq-text-tertiary)]">Conversations</p>
        <h2 class="mt-1 text-sm font-semibold text-slate-50">会话列表</h2>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.08)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[rgba(255,255,255,0.12)] hover:text-white"
          type="button"
          title="收起会话列表"
          @click="$emit('toggle-collapse')"
        >
          <PanelLeftClose class="h-4 w-4" />
        </button>
        <button
          class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.08)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[rgba(255,255,255,0.12)] hover:text-white"
          type="button"
          title="刷新"
          @click="$emit('refresh')"
        >
          <RefreshCw class="h-4 w-4" />
        </button>
        <button
          class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] bg-[linear-gradient(135deg,var(--qq-accent),var(--qq-accent-strong))] text-slate-950 transition hover:brightness-105"
          type="button"
          title="新建会话"
          @click="$emit('new-chat')"
        >
          <MessageSquarePlus class="h-4 w-4" />
        </button>
      </div>
    </div>

    <div class="border-b border-white/10 px-4 py-2.5 text-xs text-[color:var(--qq-text-tertiary)]">
      {{ loading ? '正在读取网关会话...' : streaming ? `${runningConversationIds.length} 个会话正在运行` : '按最后活动时间排序' }}
    </div>
    <div class="scrollbar-thin min-h-0 flex-1 overflow-y-auto">
      <div
        v-for="conversation in conversations"
        :key="conversation.id"
        :data-testid="`conversation-item-${conversation.id}`"
        class="block border-b border-white/8 px-4 py-3 transition"
        :class="
          conversation.id === activeId
            ? 'bg-[rgba(255,255,255,0.10)]'
            : 'hover:bg-[rgba(255,255,255,0.05)]'
        "
      >
        <div class="flex items-start justify-between gap-3">
          <RouterLink :to="`/chat/${conversation.id}`" :data-testid="`conversation-open-${conversation.id}`" class="min-w-0 flex-1">
            <h3 class="line-clamp-2 text-sm font-medium text-slate-100">
              {{ conversation.title || 'Untitled Conversation' }}
            </h3>
            <p class="mt-1.5 flex min-w-0 items-center gap-2 text-xs text-[color:var(--qq-text-tertiary)]">
              <span class="truncate">
                {{ conversation.agentId }}<span v-if="conversation.status"> · {{ conversation.status }}</span>
              </span>
              <span
                v-if="isRunning(conversation, runningConversationIds)"
                class="inline-flex shrink-0 items-center gap-1 rounded-[4px] bg-[rgba(72,255,209,0.14)] px-1.5 py-0.5 text-[10px] text-[color:var(--qq-accent)]"
              >
                <LoaderCircle class="h-3 w-3 animate-spin" />
                运行中
              </span>
              <span
                v-else-if="conversation.status === 'running'"
                class="inline-flex shrink-0 items-center gap-1 rounded-[4px] bg-[rgba(72,255,209,0.14)] px-1.5 py-0.5 text-[10px] text-[color:var(--qq-accent)]"
              >
                <LoaderCircle class="h-3 w-3 animate-spin" />
                运行中
              </span>
            </p>
          </RouterLink>
          <div class="flex shrink-0 items-start gap-2">
            <span class="pt-0.5 text-[11px] text-[color:var(--qq-text-tertiary)]">{{ formatTime(conversation.updatedAt || conversation.createdAt) }}</span>
            <button
              class="inline-flex h-7 w-7 items-center justify-center rounded-[4px] text-[color:var(--qq-text-tertiary)] transition hover:bg-[rgba(255,141,141,0.12)] hover:text-rose-100"
              :data-testid="`conversation-delete-${conversation.id}`"
              type="button"
              title="删除会话"
              @click="$emit('delete', conversation.id)"
            >
              <Trash2 v-if="deletingId !== conversation.id" class="h-3.5 w-3.5" />
              <span v-else class="text-[10px]">...</span>
            </button>
          </div>
        </div>
      </div>

      <div v-if="!loading && conversations.length === 0" class="px-4 py-8 text-sm leading-7 text-[color:var(--qq-text-tertiary)]">
        当前还没有会话。发送第一条消息后，会话会立即出现在这里。
      </div>
    </div>
    </template>
  </aside>
</template>
