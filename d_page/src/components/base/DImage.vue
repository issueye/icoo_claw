<script setup>
import { computed, ref, watch } from 'vue'

const props = defineProps({
  src: {
    type: String,
    default: '',
  },
  alt: {
    type: String,
    default: '',
  },
  caption: {
    type: String,
    default: '',
  },
  aspectRatio: {
    type: String,
    default: '16 / 9',
  },
  fit: {
    type: String,
    default: 'cover',
  },
  emptyText: {
    type: String,
    default: 'No image',
  },
  errorText: {
    type: String,
    default: 'Image unavailable',
  },
})

const failed = ref(false)

const hasSource = computed(() => String(props.src || '').trim().length > 0)
const statusText = computed(() => (hasSource.value ? props.errorText : props.emptyText))

watch(() => props.src, () => {
  failed.value = false
})

function handleError() {
  failed.value = true
}
</script>

<template>
  <figure class="d-page-image">
    <div class="d-page-image__frame" :style="{ aspectRatio }">
      <img
        v-if="hasSource && !failed"
        class="d-page-image__media"
        :src="src"
        :alt="alt"
        :style="{ objectFit: fit }"
        loading="lazy"
        @error="handleError"
      >
      <div v-else class="d-page-image__fallback" role="status">
        {{ statusText }}
      </div>
    </div>
    <figcaption v-if="caption" class="d-page-image__caption">{{ caption }}</figcaption>
  </figure>
</template>

<style scoped>
.d-page-image {
  display: grid;
  gap: 7px;
  margin: 0;
  color: var(--d-page-text, #172033);
}

.d-page-image__frame {
  display: grid;
  width: 100%;
  min-height: 120px;
  place-items: center;
  overflow: hidden;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: var(--d-page-bg, #f8fafc);
}

.d-page-image__media {
  width: 100%;
  height: 100%;
  display: block;
}

.d-page-image__fallback {
  width: 100%;
  height: 100%;
  min-height: 120px;
  display: grid;
  place-items: center;
  color: var(--d-page-muted, #64748b);
  font-size: 13px;
  line-height: 1.4;
  padding: 16px;
  text-align: center;
}

.d-page-image__caption {
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  line-height: 1.4;
  overflow-wrap: anywhere;
}
</style>
