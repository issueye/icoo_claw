<script setup>
import { computed } from 'vue'

const props = defineProps({
  columns: {
    type: Array,
    default: () => [],
  },
  rows: {
    type: Array,
    default: () => [],
  },
  caption: {
    type: String,
    default: '',
  },
  emptyText: {
    type: String,
    default: 'No rows',
  },
})

const emit = defineEmits(['rowSelect', 'interact'])

const normalizedRows = computed(() => props.rows.filter((row) => row && typeof row === 'object'))
const normalizedColumns = computed(() => {
  const configuredColumns = props.columns
    .filter((column) => column && typeof column === 'object' && column.key)
    .map((column) => ({
      key: String(column.key),
      label: String(column.label || column.key),
      align: column.align || 'left',
      width: column.width || undefined,
    }))

  if (configuredColumns.length > 0) return configuredColumns

  const firstRow = normalizedRows.value[0]
  if (!firstRow) return []

  return Object.keys(firstRow).map((key) => ({
    key,
    label: key,
    align: 'left',
    width: undefined,
  }))
})

const hasData = computed(() => normalizedRows.value.length > 0 && normalizedColumns.value.length > 0)

function formatCell(row, key) {
  const value = row[key]
  if (value === null || value === undefined || value === '') return '-'
  if (Array.isArray(value)) return value.join(', ')
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function selectRow(row, index) {
  const payload = {
    type: 'rowSelect',
    source: 'table',
    row,
    index,
  }

  emit('rowSelect', payload)
  emit('interact', payload)
}
</script>

<template>
  <div class="d-page-table-wrap">
    <table v-if="hasData" class="d-page-table">
      <caption v-if="caption" class="d-page-table__caption">{{ caption }}</caption>
      <thead>
        <tr>
          <th
            v-for="column in normalizedColumns"
            :key="column.key"
            scope="col"
            :style="{ textAlign: column.align, width: column.width }"
          >
            {{ column.label }}
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(row, rowIndex) in normalizedRows"
          :key="row.id || row.key || rowIndex"
          tabindex="0"
          @click="selectRow(row, rowIndex)"
          @keydown.enter="selectRow(row, rowIndex)"
        >
          <td
            v-for="column in normalizedColumns"
            :key="column.key"
            :style="{ textAlign: column.align }"
          >
            {{ formatCell(row, column.key) }}
          </td>
        </tr>
      </tbody>
    </table>
    <p v-else class="d-page-table__empty">{{ emptyText }}</p>
  </div>
</template>

<style scoped>
.d-page-table-wrap {
  width: 100%;
  overflow-x: auto;
  border: 1px solid var(--d-page-border, #dbe3ea);
  border-radius: var(--d-page-radius, 6px);
  background: var(--d-page-surface, #ffffff);
}

.d-page-table {
  width: 100%;
  min-width: 360px;
  border-collapse: collapse;
  color: var(--d-page-text, #172033);
  font-size: 13px;
}

.d-page-table__caption {
  padding: 10px 12px;
  color: var(--d-page-muted, #64748b);
  font-weight: 650;
  text-align: left;
}

.d-page-table th,
.d-page-table td {
  border-bottom: 1px solid var(--d-page-border, #dbe3ea);
  padding: 10px 12px;
  vertical-align: top;
  white-space: nowrap;
}

.d-page-table th {
  background: color-mix(in srgb, var(--d-page-bg, #f8fafc) 82%, white);
  color: var(--d-page-muted, #64748b);
  font-size: 12px;
  font-weight: 700;
}

.d-page-table tbody tr {
  cursor: pointer;
}

.d-page-table tbody tr:hover,
.d-page-table tbody tr:focus-visible {
  background: color-mix(in srgb, var(--d-page-accent, #14b8a6) 7%, white);
  outline: none;
}

.d-page-table tbody tr:last-child td {
  border-bottom: 0;
}

.d-page-table__empty {
  margin: 0;
  padding: 18px;
  color: var(--d-page-muted, #64748b);
  font-size: 13px;
  line-height: 1.5;
  text-align: center;
}
</style>
