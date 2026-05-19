<script setup>
defineProps({
  message: {
    type: Object,
    required: true,
  },
  showTimestamps: {
    type: Boolean,
    default: true,
  },
})

function roleLabel(role) {
  return role === 'user' ? 'You' : 'Assistant'
}

function formatTimestamp(value) {
  if (!value) {
    return ''
  }
  return new Date(value).toLocaleTimeString()
}
</script>

<template>
  <article class="border-b border-line/70 px-5 py-5 last:border-b-0">
    <header class="mb-3 flex items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <span
          class="inline-flex rounded-full px-2.5 py-1 text-[11px] uppercase tracking-[0.16em]"
          :class="message.role === 'user' ? 'bg-slate-800 text-slate-200' : 'bg-accent/10 text-accent'"
        >
          {{ roleLabel(message.role) }}
        </span>
        <span v-if="message.draft" class="text-xs text-slate-500">streaming</span>
      </div>
      <time v-if="showTimestamps" class="text-xs text-slate-500">
        {{ formatTimestamp(message.createdAt) }}
      </time>
    </header>

    <pre
      class="m-0 whitespace-pre-wrap break-words text-sm leading-7"
      :class="message.error ? 'text-rose-200' : 'text-slate-200'"
    >{{ message.content }}</pre>
  </article>
</template>
