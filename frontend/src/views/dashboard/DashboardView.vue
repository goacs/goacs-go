<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import dayjs from 'dayjs'
import StatTile from '@/components/common/StatTile.vue'
import JsonViewer from '@/components/common/JsonViewer.vue'
import { useDashboardStore } from '@/stores/dashboard.store'
import type { Fault } from '@/api/types/dashboard'

const dashboardStore = useDashboardStore()
const selectedFault = ref<Fault | null>(null)
const detailsVisible = ref(false)

function showDetails(fault: Fault) {
  selectedFault.value = fault
  detailsVisible.value = true
}

onMounted(() => dashboardStore.fetch())

function formatDate(value: string) {
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}
</script>

<template>
  <div class="dashboard">
    <h1>Dashboard</h1>

    <div class="tiles">
      <StatTile label="Devices" icon="pi pi-desktop" :value="dashboardStore.data?.devices_count ?? 0" />
      <StatTile
        label="Informs (24h)"
        icon="pi pi-refresh"
        severity="success"
        :value="dashboardStore.data?.informs_count ?? 0"
      />
      <StatTile
        label="Faults (24h)"
        icon="pi pi-exclamation-triangle"
        severity="danger"
        :value="dashboardStore.data?.faults_count ?? 0"
      />
    </div>

    <h2>Recent faults</h2>
    <DataTable :value="dashboardStore.data?.faults ?? []" :loading="dashboardStore.loading" size="small">
      <Column field="cpe_uuid" header="Device" />
      <Column field="code" header="Code" />
      <Column field="message" header="Message" />
      <Column header="When">
        <template #body="{ data }">{{ formatDate(data.created_at) }}</template>
      </Column>
      <Column header="">
        <template #body="{ data }">
          <Button icon="pi pi-eye" text size="small" @click="showDetails(data)" />
        </template>
      </Column>
    </DataTable>

    <Dialog v-model:visible="detailsVisible" header="Fault details" modal>
      <JsonViewer v-if="selectedFault" :data="selectedFault" />
    </Dialog>
  </div>
</template>

<style scoped>
.tiles {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1rem;
  margin-bottom: 2rem;
}
</style>
