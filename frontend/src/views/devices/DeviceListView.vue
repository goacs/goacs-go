<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import InputText from 'primevue/inputtext'
import dayjs from 'dayjs'
import { useServerTable } from '@/composables/useServerTable'
import { deviceApi } from '@/api/endpoints/device.api'

const router = useRouter()

const table = useServerTable({ fetcher: (params) => deviceApi.list(params) })

function updateFilter(field: string, value: string) {
  table.onFilterChange({ ...table.filter.value, [field]: value })
}

function openDevice(uuid: string) {
  router.push({ name: 'devices-view', params: { uuid } })
}

function formatDate(value: string) {
  return value ? dayjs(value).format('YYYY-MM-DD HH:mm:ss') : '-'
}

onMounted(() => table.load())
</script>

<template>
  <div>
    <h1>Devices</h1>

    <DataTable
      :value="table.items.value"
      :loading="table.loading.value"
      :total-records="table.total.value"
      :rows="table.perPage.value"
      :first="(table.page.value - 1) * table.perPage.value"
      lazy
      paginator
      :rows-per-page-options="[25, 50, 100]"
      size="small"
      data-key="uuid"
      @page="table.onPage"
      @row-click="(event) => openDevice(event.data.uuid)"
      style="cursor: pointer"
    >
      <Column field="serial_number" header="Serial number">
        <template #header>
          <div class="col-filter">
            Serial number
            <InputText size="small" @click.stop @input="(e: Event) => updateFilter('serial_number', (e.target as HTMLInputElement).value)" />
          </div>
        </template>
      </Column>
      <Column field="oui" header="OUI" />
      <Column field="software_version" header="Software" />
      <Column field="hardware_version" header="Hardware" />
      <Column header="Last update">
        <template #body="{ data }">{{ formatDate(data.updated_at) }}</template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped>
.col-filter {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}
</style>
