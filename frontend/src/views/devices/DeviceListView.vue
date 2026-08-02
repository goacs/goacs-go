<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import InputText from 'primevue/inputtext'
import dayjs from 'dayjs'
import { FilterMatchMode } from '@primevue/core/api'
import { useServerTable } from '@/composables/useServerTable'
import { deviceApi } from '@/api/endpoints/device.api'

const router = useRouter()

const table = useServerTable({
  fetcher: (params) => deviceApi.list(params),
  filters: {
    serial_number: { value: '', matchMode: FilterMatchMode.CONTAINS },
  },
})

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
      v-model:filters="table.filters.value"
      :value="table.items.value"
      :loading="table.loading.value"
      :total-records="table.total.value"
      :rows="table.perPage.value"
      :first="(table.page.value - 1) * table.perPage.value"
      lazy
      paginator
      filter-display="row"
      :rows-per-page-options="[25, 50, 100]"
      size="small"
      data-key="uuid"
      @page="table.onPage"
      @filter="table.onFilter"
      @row-click="(event) => openDevice(event.data.uuid)"
      style="cursor: pointer"
    >
      <Column field="serial_number" header="Serial number" :show-filter-menu="false">
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" size="small" placeholder="Search" @input="filterCallback()" />
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
