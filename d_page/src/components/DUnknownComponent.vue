<script setup>
defineProps({
  type: {
    type: String,
    default: '',
  },
})
</script>

<template>
  <div class="d-page-card" role="status">
    Unknown component<span v-if="type">: {{ type }}</span>
  </div>
</template>
