<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Panel from 'primevue/panel'
import Button from 'primevue/button'
import { deviceApi } from '@/api/endpoints/device.api'
import type { DiagnosticsReport, DiagnosticsResult } from '@/api/types/device'
import SpeedTestDialog from '@/components/device/SpeedTestDialog.vue'

const props = defineProps<{ uuid: string }>()

const report = ref<DiagnosticsReport | null>(null)
const loading = ref(false)
const dialogVisible = ref(false)

async function load() {
  loading.value = true
  try {
    report.value = await deviceApi.getDiagnosticsReport(props.uuid)
  } finally {
    loading.value = false
  }
}

function formatMbps(result: DiagnosticsResult) {
  return `${result.throughput_mbps.toFixed(1)} Mbps`
}

function formatMeta(result: DiagnosticsResult) {
  const mib = (result.bytes / (1024 * 1024)).toFixed(1)
  return `${mib} MiB in ${result.duration_seconds.toFixed(1)}s · finished ${new Date(result.end_time).toLocaleString()}`
}

onMounted(load)
</script>

<template>
  <Panel header="Speedtest" toggleable>
    <template #icons>
      <Button icon="pi pi-play" text size="small" @click="dialogVisible = true" />
    </template>

    <div class="results" v-if="!loading && (report?.download || report?.upload)">
      <div class="result" v-if="report?.download">
        <div class="result__label">Download</div>
        <div class="result__value">{{ formatMbps(report.download) }}</div>
        <div class="result__meta">{{ formatMeta(report.download) }}</div>
      </div>
      <div class="result" v-if="report?.upload">
        <div class="result__label">Upload</div>
        <div class="result__value">{{ formatMbps(report.upload) }}</div>
        <div class="result__meta">{{ formatMeta(report.upload) }}</div>
      </div>
    </div>
    <div class="empty" v-else-if="!loading">No speed test result in the last 24h.</div>

    <SpeedTestDialog v-model:visible="dialogVisible" :uuid="props.uuid" @saved="load" />
  </Panel>
</template>

<style scoped>
.results {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.result__label {
  font-weight: 600;
}

.result__value {
  font-size: 1.5rem;
}

.result__meta {
  font-size: 0.85rem;
  opacity: 0.7;
}

.empty {
  opacity: 0.7;
  font-size: 0.9rem;
}
</style>
