<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import Tag from 'primevue/tag'
import { useConfirm } from 'primevue/useconfirm'
import dayjs from 'dayjs'
import { useServerTable } from '@/composables/useServerTable'
import { useDeviceLogsSocket } from '@/composables/useDeviceLogsSocket'
import { deviceApi } from '@/api/endpoints/device.api'
import { downloadBlob } from '@/composables/useDownload'
import type { LogEntry } from '@/api/types/log'
import LogDetailsDialog from '@/components/device/LogDetailsDialog.vue'

const props = defineProps<{ uuid: string }>()
const confirm = useConfirm()

const table = useServerTable<LogEntry>({ fetcher: (params) => deviceApi.getLogs(props.uuid, params), perPage: 10 })
const selected = ref<LogEntry | null>(null)
const detailsVisible = ref(false)

useDeviceLogsSocket(props.uuid, (entry) => {
  table.items.value = [entry, ...table.items.value]
  table.total.value += 1
})

function showDetails(entry: LogEntry) {
  selected.value = entry
  detailsVisible.value = true
}

async function download(entry: LogEntry) {
  const response = await deviceApi.downloadLogs(props.uuid, entry.session_id)
  downloadBlob(response.data as Blob, `${props.uuid}-${entry.session_id}.log`)
}

function severity(type: string) {
  if (type === 'FAULT' || type === 'ERROR') return 'danger'
  if (type === 'REQUEST' || type === 'RESPONSE') return 'info'
  return 'secondary'
}

function formatDate(value: string) {
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}

function confirmDeleteAll() {
  confirm.require({
    message: 'Delete all logs for this device? This cannot be undone.',
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    accept: async () => {
      await deviceApi.deleteLogs(props.uuid)
      await table.load()
    },
  })
}

onMounted(() => table.load())
</script>

<template>
  <Panel header="Logs (live)" toggleable>
    <template #icons>
      <Button icon="pi pi-trash" text severity="danger" size="small" @click="confirmDeleteAll" />
    </template>

    <DataTable
      :value="table.items.value"
      :loading="table.loading.value"
      :total-records="table.total.value"
      :rows="table.perPage.value"
      :first="(table.page.value - 1) * table.perPage.value"
      lazy
      paginator
      size="small"
      @page="table.onPage"
    >
      <Column header="Type">
        <template #body="{ data }"><Tag :value="data.type" :severity="severity(data.type)" /></template>
      </Column>
      <Column field="from" header="From" />
      <Column field="message" header="Message" />
      <Column header="When">
        <template #body="{ data }">{{ formatDate(data.created_at) }}</template>
      </Column>
      <Column header="">
        <template #body="{ data }">
          <Button icon="pi pi-eye" text size="small" @click="showDetails(data)" />
          <Button icon="pi pi-download" text size="small" @click="download(data)" />
        </template>
      </Column>
      <template #empty>No logs yet.</template>
    </DataTable>

    <LogDetailsDialog v-model:visible="detailsVisible" :entry="selected" @download-session="download" />
  </Panel>
</template>
