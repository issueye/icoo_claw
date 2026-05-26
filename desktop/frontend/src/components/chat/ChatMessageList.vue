<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import ChatMessageItem from './ChatMessageItem.vue'

const props = defineProps({
  messages: {
    type: Array,
    default: () => [],
  },
  showTimestamps: {
    type: Boolean,
    default: true,
  },
})

const viewport = ref(null)
const visibleMessages = computed(() => props.messages.filter((message) => (
  message.role !== 'assistant' ||
  String(message.content || '').trim() ||
  message.error ||
  message.metadata?.toolCallId ||
  message.metadata?.usage
)))

watch(
  () => visibleMessages.value.map((message) => `${message.id}:${message.content.length}`).join('|'),
  async () => {
    await nextTick()
    if (viewport.value) {
      viewport.value.scrollTop = viewport.value.scrollHeight
    }
  },
)
</script>

<template>
  <div ref="viewport" class="scrollbar-thin h-full overflow-y-auto">
    <ChatMessageItem
      v-for="message in visibleMessages"
      :key="message.id"
      :message="message"
      :show-timestamps="showTimestamps"
    />
  </div>
</template>
