<script setup lang="ts">
import Panel from 'primevue/panel'
import dayjs from 'dayjs'
import type { CPE } from '@/api/types/device'

defineProps<{ device: CPE }>()

function formatDate(value: string) {
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}
</script>

<template>
  <Panel header="Device info">
    <dl class="info-grid">
      <dt>Software version</dt>
      <dd>{{ device.software_version }}</dd>
      <dt>Hardware version</dt>
      <dd>{{ device.hardware_version }}</dd>
      <dt>IP address</dt>
      <dd>{{ device.ip_address.IP }}</dd>
      <dt>Connection Request URL</dt>
      <dd class="url">{{ device.connection_request_url }}</dd>
      <dt>Last inform</dt>
      <dd>{{ formatDate(device.updated_at) }}</dd>
    </dl>
  </Panel>
</template>

<style scoped>
.info-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 0.5rem;
  margin: 0;
  font-size: 0.85rem;
}

dt {
  color: rgba(0, 0, 0, 0.45);
  font-size: 0.72rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.03em;
  margin-top: 0.3rem;
}

dt:first-child {
  margin-top: 0;
}

dd {
  margin: 0.1rem 0 0;
  font-weight: 500;
}

.url {
  font-family: ui-monospace, monospace;
  font-size: 0.72rem;
  opacity: 0.75;
  word-break: break-all;
}
</style>
