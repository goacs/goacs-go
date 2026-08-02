<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import { FilterMatchMode } from '@primevue/core/api'
import { useServerTable } from '@/composables/useServerTable'
import { configurationApi } from '@/api/endpoints/configuration.api'
import { CWMP_EVENTS, CWMP_REQUESTS, type Provision } from '@/api/types/configuration'

const router = useRouter()
const table = useServerTable<Provision>({
  fetcher: (params) => configurationApi.list(params),
  filters: {
    name: { value: '', matchMode: FilterMatchMode.CONTAINS },
    events: { value: '', matchMode: FilterMatchMode.CONTAINS },
    requests: { value: '', matchMode: FilterMatchMode.CONTAINS },
  },
})

function openProvision(id: number) {
  router.push({ name: 'configuration-edit', params: { id } })
}

onMounted(() => table.load())
</script>

<template>
  <div>
    <div class="header-row">
      <h1>Configuration</h1>
      <Button label="New provision" icon="pi pi-plus" @click="router.push({ name: 'configuration-create' })" />
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
      data-key="id"
      @page="table.onPage"
      @filter="table.onFilter"
      @row-click="(event) => openProvision(event.data.id)"
      style="cursor: pointer"
    >
      <Column field="name" header="Name" :show-filter-menu="false">
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" size="small" placeholder="Search by name..." @input="filterCallback()" />
        </template>
      </Column>
      <Column field="events" header="Events" :show-filter-menu="false">
        <template #filter="{ filterModel, filterCallback }">
          <Select
            v-model="filterModel.value"
            :options="CWMP_EVENTS"
            option-label="label"
            option-value="value"
            placeholder="Any event"
            show-clear
            size="small"
            @change="filterCallback()"
          />
        </template>
      </Column>
      <Column field="requests" header="Requests" :show-filter-menu="false">
        <template #filter="{ filterModel, filterCallback }">
          <Select
            v-model="filterModel.value"
            :options="CWMP_REQUESTS"
            option-label="label"
            option-value="value"
            placeholder="Any request"
            show-clear
            size="small"
            @change="filterCallback()"
          />
        </template>
      </Column>
    </DataTable>
  </div>
</template>

<style scoped>
.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}
</style>
