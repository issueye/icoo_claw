<script setup>
defineProps({
  message: {
    type: String,
    default: 'Render error',
  },
})
</script>

<template>
  <div class="d-page-card" role="alert" style="border-color: var(--d-page-danger); color: var(--d-page-danger);">
    {{ message }}
  </div>
</template>
