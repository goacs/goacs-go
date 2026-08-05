<script setup lang="ts">
import { computed } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import dayjs from 'dayjs'
import type { LogEntry } from '@/api/types/log'
import { parseCwmpXml } from '@/helpers/cwmpXml'

const props = defineProps<{ entry: LogEntry | null }>()
const emit = defineEmits<{ downloadSession: [entry: LogEntry] }>()
const visible = defineModel<boolean>('visible', { required: true })

const parsed = computed(() => (props.entry ? parseCwmpXml(props.entry.full_xml) : null))

function severity(type: string) {
  if (type === 'FAULT' || type === 'ERROR') return 'danger'
  if (type === 'REQUEST' || type === 'RESPONSE') return 'info'
  return 'secondary'
}

function formatDate(value: string) {
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}

async function copyXml() {
  if (props.entry) await navigator.clipboard.writeText(props.entry.full_xml)
}
</script>

<template>
  <Dialog v-model:visible="visible" modal style="width: max(38rem, 55vw)">
    <template #header>
      <div v-if="entry" class="hd">
        <span class="title">Log details</span>
        <Tag v-if="parsed" :value="parsed.method" severity="contrast" />
        <Tag :value="entry.type" :severity="severity(entry.type)" />
      </div>
    </template>

    <div v-if="entry" class="body">
      <dl class="meta-grid">
        <div>
          <dt>Direction</dt>
          <dd>{{ entry.from === 'acs' ? 'ACS → device' : 'Device → ACS' }}</dd>
        </div>
        <div>
          <dt>When</dt>
          <dd>{{ formatDate(entry.created_at) }}</dd>
        </div>
        <div>
          <dt>Session</dt>
          <dd class="mono">{{ entry.session_id }}</dd>
        </div>
        <div>
          <dt>Log ID</dt>
          <dd class="mono">{{ entry.id }}</dd>
        </div>
      </dl>

      <div v-if="entry.message || entry.code" class="section">
        <div class="section-title">Message</div>
        <div class="msg"><span v-if="entry.code" class="mono">{{ entry.code }} — </span>{{ entry.message }}</div>
      </div>

      <template v-if="parsed">
        <div v-if="parsed.deviceId" class="section">
          <div class="section-title">Reporting device</div>
          <div class="kv-card">
            <div v-if="parsed.deviceId.manufacturer"><span>Manufacturer</span><b>{{ parsed.deviceId.manufacturer }}</b></div>
            <div v-if="parsed.deviceId.serialNumber"><span>Serial number</span><b class="mono">{{ parsed.deviceId.serialNumber }}</b></div>
            <div v-if="parsed.deviceId.oui"><span>OUI</span><b class="mono">{{ parsed.deviceId.oui }}</b></div>
            <div v-if="parsed.deviceId.productClass"><span>Product class</span><b>{{ parsed.deviceId.productClass }}</b></div>
          </div>
        </div>

        <div v-if="parsed.events.length" class="section">
          <div class="section-title">Events</div>
          <div class="chips">
            <Tag v-for="e in parsed.events" :key="e" :value="e" severity="success" />
          </div>
        </div>

        <div v-if="parsed.parameters.length" class="section">
          <div class="section-title-row">
            <span class="section-title">Parameters</span>
            <span class="count">{{ parsed.parameters.length }}</span>
          </div>
          <table class="param-table">
            <thead>
              <tr><th>Name</th><th>Value</th><th>Type</th></tr>
            </thead>
            <tbody>
              <tr v-for="p in parsed.parameters" :key="p.name">
                <td class="mono">{{ p.name }}</td>
                <td>{{ p.value || '—' }}</td>
                <td class="type">{{ p.type }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </template>

      <div v-if="!entry.full_xml" class="empty-post">EMPTY POST</div>
      <details v-else class="raw">
        <summary>Raw SOAP/XML</summary>
        <pre>{{ entry.full_xml }}</pre>
      </details>
    </div>

    <template #footer>
      <Button v-if="entry" label="Download session log" icon="pi pi-download" text @click="emit('downloadSession', entry)" />
      <Button v-if="entry?.full_xml" label="Copy XML" text @click="copyXml" />
      <Button label="Close" @click="visible = false" />
    </template>
  </Dialog>
</template>

<style scoped>
.hd {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.title {
  font-weight: 700;
  font-size: 1.05rem;
}

.body {
  display: flex;
  flex-direction: column;
  gap: 1.1rem;
  max-height: 65vh;
  overflow: auto;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 0.6rem 1rem;
  margin: 0;
  padding-bottom: 0.9rem;
  border-bottom: 1px solid var(--p-surface-200, #e5e7eb);
  font-size: 0.8rem;
}

.meta-grid dt {
  color: rgba(0, 0, 0, 0.45);
  font-size: 0.7rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.meta-grid dd {
  margin: 0.15rem 0 0;
  font-weight: 600;
}

.section-title,
.section-title-row .section-title {
  font-size: 0.7rem;
  font-weight: 700;
  color: rgba(0, 0, 0, 0.5);
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-bottom: 0.5rem;
}

.section-title-row {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 0.5rem;
}

.section-title-row .count {
  font-size: 0.75rem;
  color: rgba(0, 0, 0, 0.4);
}

.msg {
  font-size: 0.85rem;
  background: var(--p-surface-50, #fafafa);
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 8px;
  padding: 0.6rem 0.75rem;
}

.kv-card {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 0.5rem 1.25rem;
  font-size: 0.85rem;
  background: var(--p-surface-50, #fafafa);
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 8px;
  padding: 0.75rem 0.9rem;
}

.kv-card > div {
  display: flex;
  justify-content: space-between;
  gap: 0.5rem;
}

.kv-card span {
  color: rgba(0, 0, 0, 0.45);
}

.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem;
}

.param-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.78rem;
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 8px;
  overflow: hidden;
}

.param-table th {
  text-align: left;
  font-size: 0.68rem;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  color: rgba(0, 0, 0, 0.45);
  padding: 0.5rem 0.6rem;
  border-bottom: 1px solid var(--p-surface-200, #e5e7eb);
}

.param-table td {
  padding: 0.45rem 0.6rem;
  border-bottom: 1px solid var(--p-surface-100, #f1f5f9);
}

.param-table tr:last-child td {
  border-bottom: none;
}

.mono {
  font-family: ui-monospace, monospace;
  font-size: 0.75rem;
}

.type {
  color: rgba(0, 0, 0, 0.45);
}

.empty-post {
  text-align: center;
  font-size: 0.8rem;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.4);
  background: var(--p-surface-50, #fafafa);
  border: 1px dashed var(--p-surface-200, #e5e7eb);
  border-radius: 8px;
  padding: 1rem;
}

.raw {
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 8px;
}

.raw summary {
  padding: 0.6rem 0.75rem;
  font-size: 0.8rem;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.55);
  cursor: pointer;
}

.raw pre {
  margin: 0;
  padding: 0 0.75rem 0.75rem;
  white-space: pre-wrap;
  word-break: break-word;
  font-family: ui-monospace, monospace;
  font-size: 0.72rem;
  line-height: 1.55;
  color: rgba(0, 0, 0, 0.65);
  max-height: 40vh;
  overflow: auto;
}
</style>
