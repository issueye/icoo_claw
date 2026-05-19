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
  <div class="flex flex-wrap items-center gap-2 border-b border-line bg-[#0d1219] px-5 py-3 text-xs text-slate-400">
    <span class="rounded-full border border-line bg-panel px-2.5 py-1">{{ gatewayLabel(gatewayStatus) }}</span>
    <span class="rounded-full border border-line bg-panel px-2.5 py-1">{{ socketLabel(socketStatus) }}</span>
    <span class="rounded-full border border-line bg-panel px-2.5 py-1">Agent {{ agentName }}</span>
  </div>
</template>
