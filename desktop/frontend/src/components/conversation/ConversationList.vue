<script setup>
import { MessageSquarePlus, RefreshCw, Trash2 } from 'lucide-vue-next'

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
  deletingId: {
    type: String,
    default: '',
  },
})

defineEmits(['delete', 'new-chat', 'refresh'])

function formatTime(value) {
  if (!value) return ''
  return new Date(value).toLocaleDateString()
}
</script>

<template>
  <aside class="min-h-0 w-80 shrink-0 flex-col border-r border-line bg-panel">
    <div class="flex items-center justify-between border-b border-line px-4 py-4">
      <div>
        <p class="text-xs uppercase tracking-[0.18em] text-slate-500">Conversations</p>
        <h2 class="mt-1 text-sm font-semibold text-slate-50">会话列表</h2>
      </div>
      <div class="flex items-center gap-2">
        <button
          class="inline-flex h-9 w-9 items-center justify-center rounded-md border border-line bg-panelSoft text-slate-300 transition hover:border-accent/60 hover:text-accent"
          type="button"
          title="刷新"
          @click="$emit('refresh')"
        >
          <RefreshCw class="h-4 w-4" />
        </button>
        <button
          class="inline-flex h-9 w-9 items-center justify-center rounded-md bg-accent text-slate-950 transition hover:bg-accentStrong"
          type="button"
          title="新建会话"
          @click="$emit('new-chat')"
        >
          <MessageSquarePlus class="h-4 w-4" />
        </button>
      </div>
    </div>

    <div class="border-b border-line px-4 py-3 text-xs text-slate-500">
      {{ loading ? '正在读取网关会话...' : streaming ? '流式响应中，会话列表稍后会同步刷新' : '按最后活动时间排序' }}
    </div>
    <div class="scrollbar-thin min-h-0 flex-1 overflow-y-auto">
      <div
        v-for="conversation in conversations"
        :key="conversation.id"
        :data-testid="`conversation-item-${conversation.id}`"
        class="block border-b border-line/70 px-4 py-4 transition"
        :class="conversation.id === activeId ? 'bg-[#111823]' : 'hover:bg-[#101722]'"
      >
        <div class="flex items-start justify-between gap-3">
          <RouterLink :to="`/chat/${conversation.id}`" :data-testid="`conversation-open-${conversation.id}`" class="min-w-0 flex-1">
            <h3 class="line-clamp-2 text-sm font-medium text-slate-100">
              {{ conversation.title || 'Untitled Conversation' }}
            </h3>
            <p class="mt-2 truncate text-xs text-slate-500">
              {{ conversation.agentId }}<span v-if="conversation.status"> · {{ conversation.status }}</span>
            </p>
          </RouterLink>
          <div class="flex shrink-0 items-start gap-2">
            <span class="pt-0.5 text-[11px] text-slate-500">{{ formatTime(conversation.updatedAt || conversation.createdAt) }}</span>
            <button
              class="inline-flex h-7 w-7 items-center justify-center rounded-md text-slate-500 transition hover:bg-danger/10 hover:text-rose-200"
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

      <div v-if="!loading && conversations.length === 0" class="px-4 py-8 text-sm leading-7 text-slate-500">
        当前还没有会话。发送第一条消息后，会话会立即出现在这里。
      </div>
    </div>
  </aside>
</template>
