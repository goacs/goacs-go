<script setup lang="ts">
import { onMounted } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Tag from 'primevue/tag'
import { FilterMatchMode } from '@primevue/core/api'
import { useServerTable } from '@/composables/useServerTable'
import { deviceApi } from '@/api/endpoints/device.api'
import type { Parameter } from '@/api/types/device'
import { flagToString } from '@/helpers/flags'
import { downloadBlob } from '@/composables/useDownload'

const props = defineProps<{ uuid: string }>()

const table = useServerTable<Parameter>({
  fetcher: (params) => deviceApi.getCachedParameters(props.uuid, params),
  filters: { name: { value: '', matchMode: FilterMatchMode.CONTAINS } },
})

async function download() {
  const response = await deviceApi.downloadCachedParametersCsv(props.uuid)
  downloadBlob(response.data as Blob, `${props.uuid}-cached-parameters.csv`)
}

onMounted(() => table.load())
</script>

<template>
  <div>
    <h1>Cached parameters</h1>

    <div class="toolbar">
      <Button label="Download CSV" icon="pi pi-download" severity="secondary" @click="download" />
    </div>

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
      size="small"
      @page="table.onPage"
      @filter="table.onFilter"
    >
      <Column field="name" header="Name" :show-filter-menu="false">
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" size="small" placeholder="Filter by name..." @input="filterCallback()" />
        </template>
      </Column>
      <Column header="Value">
        <template #body="{ data }">{{ data.valuestruct.value }}</template>
      </Column>
      <Column header="Type">
        <template #body="{ data }">{{ data.valuestruct.type }}</template>
      </Column>
      <Column header="Flags">
        <template #body="{ data }"><Tag :value="flagToString(data.flag)" severity="secondary" /></template>
      </Column>
      <template #empty>No cached lookup yet. Use "Lookup now" on the device page first.</template>
    </DataTable>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  gap: 0.75rem;
  margin-bottom: 1rem;
}
</style>
