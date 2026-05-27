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
    <!-- 收起状态 -->
    <div v-if="collapsed" class="flex min-h-0 flex-1 flex-col items-center gap-3 px-2 py-4">
      <button
        class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.06)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[rgba(255,255,255,0.12)] hover:text-white"
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
        class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.06)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[rgba(255,255,255,0.12)] hover:text-white"
        type="button"
        title="刷新会话"
        @click="$emit('refresh')"
      >
        <RefreshCw class="h-4 w-4" />
      </button>
      <div class="mt-2 flex flex-col items-center gap-2 text-[11px] text-[color:var(--qq-text-tertiary)]">
        <span class="rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.06)] px-2 py-1 text-[color:var(--qq-text-secondary)]">
          {{ conversations.length }}
        </span>
        <span
          v-if="streaming"
          class="inline-flex h-6 w-6 items-center justify-center rounded-full bg-[rgba(0,242,254,0.14)] text-[color:var(--qq-accent)]"
          title="有会话正在运行"
        >
          <LoaderCircle class="h-3.5 w-3.5 animate-spin" />
        </span>
      </div>
    </div>

    <!-- 展开状态 -->
    <template v-else>
      <div class="flex items-center justify-between border-b border-white/10 px-4 py-3.5">
        <div>
          <p class="text-xs uppercase tracking-[0.18em] text-[color:var(--qq-text-tertiary)] leading-4">Conversations</p>
          <h2 class="mt-1 text-sm font-semibold text-slate-50">会话历史</h2>
        </div>
        <div class="flex items-center gap-2">
          <button
            class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.06)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[rgba(255,255,255,0.12)] hover:text-white"
            type="button"
            title="收起会话列表"
            @click="$emit('toggle-collapse')"
          >
            <PanelLeftClose class="h-4 w-4" />
          </button>
          <button
            class="inline-flex h-8 w-8 items-center justify-center rounded-[4px] border border-white/10 bg-[rgba(255,255,255,0.06)] text-[color:var(--qq-text-secondary)] transition hover:border-white/20 hover:bg-[rgba(255,255,255,0.12)] hover:text-white"
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

      <div class="border-b border-white/10 px-4 py-2 text-xs text-[color:var(--qq-text-tertiary)] bg-white/2">
        {{ loading ? '正在读取网关会话...' : streaming ? `${runningConversationIds.length} 个会话正在运行` : '按最后活动时间排序' }}
      </div>

      <!-- 会话卡片列表 -->
      <div class="scrollbar-thin min-h-0 flex-1 overflow-y-auto px-2 py-2.5 space-y-1.5">
        <div
          v-for="conversation in conversations"
          :key="conversation.id"
          :data-testid="`conversation-item-${conversation.id}`"
          class="group relative block rounded-[6px] border px-3.5 py-3 transition-all duration-150 overflow-hidden"
          :class="
            conversation.id === activeId
              ? 'bg-[rgba(255,255,255,0.08)] border-white/15 shadow-[0_4px_16px_rgba(0,0,0,0.3)]'
              : 'border-transparent hover:bg-[rgba(255,255,255,0.03)] hover:border-white/5'
          "
        >
          <!-- 侧边荧光指示条 (仅选中态展示) -->
          <div
            v-if="conversation.id === activeId"
            class="absolute left-0 top-0 bottom-0 w-[3px] bg-gradient-to-b from-[color:var(--qq-accent)] to-[color:var(--qq-accent-strong)] shadow-[0_0_8px_var(--qq-accent)]"
          />

          <div class="flex items-start justify-between gap-3">
            <RouterLink :to="`/chat/${conversation.id}`" :data-testid="`conversation-open-${conversation.id}`" class="min-w-0 flex-1">
              <h3
                class="line-clamp-2 text-sm font-medium leading-5 transition-colors duration-150"
                :class="conversation.id === activeId ? 'text-slate-50' : 'text-slate-300 group-hover:text-slate-100'"
              >
                {{ conversation.title || '无标题会话' }}
              </h3>
              <p class="mt-1.5 flex min-w-0 items-center gap-2 text-xs text-[color:var(--qq-text-tertiary)]">
                <span class="truncate opacity-75">
                  {{ conversation.agentId }}<span v-if="conversation.status"> · {{ conversation.status }}</span>
                </span>
                <span
                  v-if="isRunning(conversation, runningConversationIds) || conversation.status === 'running'"
                  class="inline-flex shrink-0 items-center gap-1 rounded-[4px] bg-[rgba(0,242,254,0.12)] px-1.5 py-0.5 text-[10px] font-medium text-[color:var(--qq-accent)] animate-pulse"
                >
                  <LoaderCircle class="h-3 w-3 animate-spin" />
                  运行中
                </span>
              </p>
            </RouterLink>

            <div class="flex shrink-0 items-center gap-2.5 pt-0.5">
              <span class="text-[10px] text-[color:var(--qq-text-tertiary)] font-light opacity-80">
                {{ formatTime(conversation.updatedAt || conversation.createdAt) }}
              </span>
              <!-- 悬停即现垃圾桶按钮，减少界面噪点 -->
              <button
                class="inline-flex h-6.5 w-6.5 items-center justify-center rounded-[4px] text-[color:var(--qq-text-tertiary)] opacity-0 group-hover:opacity-100 focus:opacity-100 transition-all duration-150 hover:bg-rose-500/10 hover:text-rose-400"
                :data-testid="`conversation-delete-${conversation.id}`"
                type="button"
                title="删除会话"
                @click="$emit('delete', conversation.id)"
              >
                <Trash2 v-if="deletingId !== conversation.id" class="h-3.5 w-3.5" />
                <span v-else class="text-[9px] animate-pulse">...</span>
              </button>
            </div>
          </div>
        </div>

        <!-- 极富设计感的空状态引导 -->
        <div v-if="!loading && conversations.length === 0" class="mx-2 my-8 rounded-[8px] border border-dashed border-white/10 p-6 text-center bg-white/1">
          <MessageSquarePlus class="mx-auto h-8 w-8 text-[color:var(--qq-text-tertiary)] opacity-50" />
          <p class="mt-3 text-sm font-medium text-[color:var(--qq-text-secondary)]">暂无历史会话</p>
          <p class="mt-2 text-xs leading-5 text-[color:var(--qq-text-tertiary)]">发送第一条消息后，会话记录将自动保存在这里。</p>
          <button
            class="mt-4 inline-flex items-center gap-1.5 rounded-[4px] bg-[linear-gradient(135deg,var(--qq-accent),var(--qq-accent-strong))] px-3.5 py-1.5 text-xs font-semibold text-slate-950 transition hover:brightness-105"
            @click="$emit('new-chat')"
          >
            <MessageSquarePlus class="h-3.5 w-3.5" />
            开启新对话
          </button>
        </div>
      </div>
    </template>
  </aside>
</template>
