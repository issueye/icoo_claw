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
  <div class="flex flex-wrap items-center gap-2 border-b border-white/10 bg-[rgba(18,58,51,0.34)] px-4 py-2 text-xs text-[color:var(--qq-text-secondary)] backdrop-blur-xl">
    <span class="qq-badge rounded-[4px] px-2 py-0.5">{{ gatewayLabel(gatewayStatus) }}</span>
    <span class="qq-badge rounded-[4px] px-2 py-0.5">{{ socketLabel(socketStatus) }}</span>
    <span class="qq-badge rounded-[4px] px-2 py-0.5">Agent {{ agentName }}</span>
  </div>
</template>
