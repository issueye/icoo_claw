<script setup>
import { computed, reactive, ref, shallowRef, watch } from 'vue'
import { DPageRenderer, createDPageRuntime } from '../index.js'
import chatToolResultSchema from '../schemas/examples/chat-tool-result.json'
import simpleFormSchema from '../schemas/examples/simple-form.json'
import tableCardSchema from '../schemas/examples/table-card.json'
import liveInputPreviewSchema from '../schemas/examples/live-input-preview.json'
import componentGallerySchema from '../schemas/examples/component-gallery.json'

const examples = [
  {
    id: 'chat-tool-result',
    label: 'Chat 结果卡片',
    description: '验证工具结果展示、复制动作、emit 事件和条件显示。',
    schema: chatToolResultSchema,
  },
  {
    id: 'simple-form',
    label: '基础表单',
    description: '验证 input 输入、按钮点击、setState 和状态绑定刷新。',
    schema: simpleFormSchema,
  },
  {
    id: 'table-card',
    label: '表格卡片',
    description: '验证 data.rows 绑定、表格渲染和 rowSelect 事件。',
    schema: tableCardSchema,
  },
  {
    id: 'live-input-preview',
    label: '即时输入预览',
    description: '验证 input、select、textarea、alert、stat、list 等组件的实时绑定效果。',
    schema: liveInputPreviewSchema,
  },
  {
    id: 'component-gallery',
    label: '组件 Gallery',
    description: '验证 tag、divider、checkbox、switch、image 的基础展示和降级状态。',
    schema: componentGallerySchema,
  },
]

const activeId = ref(examples[0].id)
const runtime = shallowRef(null)
const eventLog = ref([])
const copiedText = ref('')

const activeExample = computed(() => examples.find((item) => item.id === activeId.value) || examples[0])
const runtimeState = computed(() => stringify(runtime.value?.state || {}))
const runtimeData = computed(() => stringify(runtime.value?.data || {}))

watch(activeId, resetRuntime, { immediate: true })

function resetRuntime() {
  copiedText.value = ''
  eventLog.value = []

  const nextRuntime = createDPageRuntime({
    schema: clone(activeExample.value.schema),
    context: {
      demoId: activeExample.value.id,
      source: 'd_page_demo',
    },
    adapters: {
      copyText: copyTextAdapter,
      openUrl: openUrlAdapter,
    },
    onEmit(event) {
      addLog(`emit: ${event.event}`, event.payload)
    },
  })

  nextRuntime.state = reactive(nextRuntime.state)
  nextRuntime.data = reactive(nextRuntime.data)
  nextRuntime.context = reactive(nextRuntime.context)
  runtime.value = nextRuntime
}

async function copyTextAdapter(text) {
  copiedText.value = text

  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text)
    } catch {
      // Demo keeps running even when clipboard permission is unavailable.
    }
  }

  addLog('adapter: copyText', { text })
}

async function openUrlAdapter(url, options = {}) {
  addLog('adapter: openUrl', { url, target: options.target })
  window.open(url, options.target || '_blank', 'noopener,noreferrer')
}

function addLog(label, payload) {
  eventLog.value = [
    {
      id: `${Date.now()}-${Math.random().toString(16).slice(2)}`,
      time: new Date().toLocaleTimeString(),
      label,
      payload,
    },
    ...eventLog.value,
  ].slice(0, 8)
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function stringify(value) {
  return JSON.stringify(value, null, 2)
}
</script>

<template>
  <div class="demo-shell">
    <header class="demo-header">
      <div>
        <p class="demo-eyebrow">@icoo-claw/d-page</p>
        <h1>d_page 测试 Demo</h1>
        <p class="demo-summary">切换示例 schema，验证动态卡片渲染、状态绑定、事件动作和宿主适配器。</p>
      </div>
      <button class="demo-reset" type="button" @click="resetRuntime">重置当前示例</button>
    </header>

    <nav class="demo-tabs" aria-label="Demo examples">
      <button
        v-for="example in examples"
        :key="example.id"
        class="demo-tab"
        :class="{ 'demo-tab--active': activeId === example.id }"
        type="button"
        @click="activeId = example.id"
      >
        {{ example.label }}
      </button>
    </nav>

    <main class="demo-layout">
      <section class="demo-preview" aria-label="Schema preview">
        <div class="demo-panel-heading">
          <div>
            <h2>{{ activeExample.label }}</h2>
            <p>{{ activeExample.description }}</p>
          </div>
          <span class="demo-badge">{{ activeExample.id }}</span>
        </div>
        <div class="demo-render-frame">
          <DPageRenderer v-if="runtime" :key="activeId" :schema="activeExample.schema" :runtime="runtime" />
        </div>
      </section>

      <aside class="demo-inspector" aria-label="Runtime inspector">
        <section class="demo-inspector-block">
          <h2>运行状态</h2>
          <pre>{{ runtimeState }}</pre>
        </section>

        <section class="demo-inspector-block">
          <h2>数据源</h2>
          <pre>{{ runtimeData }}</pre>
        </section>

        <section class="demo-inspector-block">
          <h2>宿主事件</h2>
          <p v-if="copiedText" class="demo-copy-result">最近复制：{{ copiedText }}</p>
          <ol v-if="eventLog.length" class="demo-event-list">
            <li v-for="item in eventLog" :key="item.id">
              <span>{{ item.time }}</span>
              <strong>{{ item.label }}</strong>
              <code>{{ JSON.stringify(item.payload) }}</code>
            </li>
          </ol>
          <p v-else class="demo-empty">暂无事件，试试点击预览里的按钮或表格行。</p>
        </section>
      </aside>
    </main>
  </div>
</template>
