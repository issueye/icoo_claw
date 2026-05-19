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
  <article class="flex px-4 py-3" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
    <div class="max-w-[min(820px,78%)]">
      <header class="mb-2 flex items-center gap-2" :class="message.role === 'user' ? 'justify-end' : 'justify-start'">
        <span
          class="inline-flex rounded-[4px] px-2 py-0.5 text-[11px] uppercase tracking-[0.12em]"
          :class="
            message.role === 'user'
              ? 'bg-[rgba(255,255,255,0.14)] text-white'
              : 'bg-[rgba(54,220,200,0.16)] text-[var(--qq-accent)]'
          "
        >
          {{ roleLabel(message.role) }}
        </span>
        <span v-if="message.draft" class="text-xs text-[color:var(--qq-text-tertiary)]">streaming</span>
        <time v-if="showTimestamps" class="text-xs text-[color:var(--qq-text-tertiary)]">
          {{ formatTimestamp(message.createdAt) }}
        </time>
      </header>

      <div
        class="border border-white/10 px-3 py-2.5"
        :class="message.role === 'user' ? 'bg-[rgba(255,255,255,0.12)]' : 'bg-[rgba(18,58,51,0.36)]'"
        style="border-radius: 6px;"
      >
        <pre
          class="m-0 whitespace-pre-wrap break-words text-sm leading-7"
          :class="message.error ? 'text-rose-100' : 'text-slate-50'"
        >{{ message.content }}</pre>
      </div>
    </div>
  </article>
</template>
