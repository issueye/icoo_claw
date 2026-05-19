<script setup>
import { nextTick, ref, watch } from 'vue'
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

watch(
  () => props.messages.map((message) => `${message.id}:${message.content.length}`).join('|'),
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
      v-for="message in messages"
      :key="message.id"
      :message="message"
      :show-timestamps="showTimestamps"
    />
  </div>
</template>
