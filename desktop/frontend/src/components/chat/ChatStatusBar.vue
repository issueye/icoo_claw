<script setup>
defineProps({
  gatewayStatus: {
    type: String,
    default: 'unknown',
  },
  socketStatus: {
    type: String,
    default: 'idle',
  },
  agentName: {
    type: String,
    default: '未选择',
  },
  projectContext: {
    type: Object,
    default: null,
  },
})

function gatewayLabel(status) {
  if (status === 'connected') return 'Gateway Online'
  if (status === 'unconfigured') return 'Gateway Unconfigured'
  if (status === 'offline') return 'Gateway Offline'
  return 'Gateway Pending'
}

function socketLabel(status) {
  if (status === 'open') return 'Socket Ready'
  if (status === 'connecting') return 'Socket Connecting'
  if (status === 'closed') return 'Socket Closed'
  return 'Socket Idle'
}
</script>

<template>
  <div class="flex flex-wrap items-center gap-3 border-b border-white/8 qq-status-bar-bg px-4 py-1.5 text-[11px] text-[color:var(--qq-text-tertiary)] backdrop-blur-xl">
    <!-- 网关状态 -->
    <span class="inline-flex items-center gap-1.5">
      <span
        class="h-1.5 w-1.5 rounded-full shrink-0"
        :class="{
          'bg-emerald-400 shadow-[0_0_4px_rgba(52,211,153,0.8)] animate-pulse': gatewayStatus === 'connected',
          'bg-rose-400 shadow-[0_0_4px_rgba(248,113,113,0.8)]': gatewayStatus === 'offline',
          'bg-amber-400': gatewayStatus === 'unconfigured',
          'bg-slate-500': !['connected','offline','unconfigured'].includes(gatewayStatus),
        }"
      />
      <span>{{ gatewayLabel(gatewayStatus) }}</span>
    </span>
    <span class="text-[color:var(--qq-border-strong)]">·</span>
    <!-- Socket 状态 -->
    <span class="inline-flex items-center gap-1.5">
      <span
        class="h-1.5 w-1.5 rounded-full shrink-0"
        :class="{
          'bg-sky-400 animate-pulse': socketStatus === 'open' || socketStatus === 'connecting',
          'bg-slate-600': socketStatus === 'idle' || socketStatus === 'closed',
        }"
      />
      <span>{{ socketLabel(socketStatus) }}</span>
    </span>
    <span class="text-[color:var(--qq-border-strong)]">·</span>
    <!-- Agent -->
    <span class="inline-flex items-center gap-1">
      <span>Agent</span>
      <span class="font-medium text-[color:var(--qq-accent)] opacity-90">{{ agentName }}</span>
    </span>
    <template v-if="projectContext">
      <span class="text-[color:var(--qq-border-strong)]">·</span>
      <span class="max-w-[200px] truncate">{{ projectContext.name }}</span>
    </template>
  </div>
</template>
