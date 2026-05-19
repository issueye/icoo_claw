<script setup>
import { onMounted, ref } from 'vue'
import { Events } from '@wailsio/runtime'
import { GreetService, SystemService } from '../bindings/icoo_claw/desktop_spike'

const name = ref('')
const result = ref('Waiting for a Go service call')
const currentTime = ref('Waiting for event stream')
const chosenDirectory = ref('No directory selected yet')
const appInfo = ref(null)
const error = ref('')

const doGreet = async () => {
  error.value = ''
  try {
    const localName = name.value.trim() || 'anonymous'
    result.value = await GreetService.Greet(localName)
  } catch (err) {
    error.value = String(err)
  }
}

const loadAppInfo = async () => {
  error.value = ''
  try {
    appInfo.value = await SystemService.GetAppInfo()
  } catch (err) {
    error.value = String(err)
  }
}

const chooseDirectory = async () => {
  error.value = ''
  try {
    const value = await SystemService.ChooseDirectory()
    chosenDirectory.value = value || 'Directory dialog cancelled'
  } catch (err) {
    error.value = String(err)
  }
}

onMounted(() => {
  Events.On('time', (payload) => {
    currentTime.value = payload.data
  })
  loadAppInfo()
})
</script>

<template>
  <main class="shell">
    <header class="hero">
      <div>
        <p class="eyebrow">Wails3 Technical Spike</p>
        <h1>Desktop wiring works end to end</h1>
        <p class="summary">
          This screen verifies Vue rendering, Go bindings, event delivery, and the native directory picker.
        </p>
      </div>
      <div class="logos">
        <a data-wml-openURL="https://wails.io">
          <img src="/wails.png" class="logo" alt="Wails logo" />
        </a>
        <a data-wml-openURL="https://vuejs.org/">
          <img src="/vue.svg" class="logo vue" alt="Vue logo" />
        </a>
      </div>
    </header>

    <section class="panel">
      <h2>Go Service Binding</h2>
      <div class="row">
        <input v-model="name" class="input" type="text" placeholder="Enter a name" />
        <button class="button" @click="doGreet">Call GreetService</button>
      </div>
      <p class="value">{{ result }}</p>
    </section>

    <section class="panel">
      <h2>Native Directory Picker</h2>
      <button class="button" @click="chooseDirectory">Choose Directory</button>
      <p class="value">{{ chosenDirectory }}</p>
    </section>

    <section class="panel grid">
      <div>
        <h2>App Info</h2>
        <pre class="code">{{ appInfo ? JSON.stringify(appInfo, null, 2) : 'Loading...' }}</pre>
      </div>
      <div>
        <h2>Runtime Event</h2>
        <p class="value">{{ currentTime }}</p>
      </div>
    </section>

    <section v-if="error" class="error-panel">
      <strong>Error</strong>
      <p>{{ error }}</p>
    </section>
  </main>
</template>

<style scoped>
:global(body) {
  margin: 0;
  font-family: Inter, "Segoe UI", sans-serif;
  background: #0f172a;
  color: #e2e8f0;
}

:global(*) {
  box-sizing: border-box;
}

.shell {
  min-height: 100vh;
  padding: 32px;
  display: grid;
  gap: 20px;
  background:
    radial-gradient(circle at top left, rgba(59, 130, 246, 0.16), transparent 28%),
    radial-gradient(circle at top right, rgba(16, 185, 129, 0.14), transparent 24%),
    #0f172a;
}

.hero,
.panel,
.error-panel {
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 16px;
  background: rgba(15, 23, 42, 0.72);
  backdrop-filter: blur(10px);
}

.hero {
  padding: 24px;
  display: flex;
  justify-content: space-between;
  gap: 24px;
  align-items: center;
}

.eyebrow {
  margin: 0 0 8px;
  font-size: 12px;
  color: #93c5fd;
  text-transform: uppercase;
}

h1,
h2,
p {
  margin: 0;
}

h1 {
  font-size: 30px;
  line-height: 1.2;
}

.summary {
  margin-top: 10px;
  max-width: 720px;
  color: #cbd5e1;
}

.logos {
  display: flex;
  align-items: center;
  gap: 8px;
}

.logo {
  height: 72px;
  width: 72px;
  padding: 12px;
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.04);
}

.panel,
.error-panel {
  padding: 20px;
}

.row {
  margin-top: 14px;
  display: flex;
  gap: 12px;
}

.input {
  flex: 1;
  min-width: 0;
  border: 1px solid rgba(148, 163, 184, 0.3);
  border-radius: 10px;
  padding: 12px 14px;
  background: rgba(15, 23, 42, 0.95);
  color: #f8fafc;
}

.button {
  border: 0;
  border-radius: 10px;
  padding: 12px 16px;
  background: #2563eb;
  color: white;
  cursor: pointer;
}

.value {
  margin-top: 14px;
  color: #cbd5e1;
  word-break: break-word;
}

.grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(280px, 1fr);
  gap: 20px;
}

.code {
  margin-top: 12px;
  padding: 14px;
  border-radius: 12px;
  background: rgba(2, 6, 23, 0.85);
  color: #bfdbfe;
  overflow: auto;
}

.error-panel {
  border-color: rgba(248, 113, 113, 0.4);
  background: rgba(127, 29, 29, 0.18);
}

@media (max-width: 860px) {
  .shell {
    padding: 20px;
  }

  .hero,
  .row,
  .grid {
    grid-template-columns: 1fr;
    flex-direction: column;
  }
}
</style>
