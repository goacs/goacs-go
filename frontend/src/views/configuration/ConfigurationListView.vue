<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import { useServerTable } from '@/composables/useServerTable'
import { configurationApi } from '@/api/endpoints/configuration.api'
import type { Provision } from '@/api/types/configuration'

const router = useRouter()
const table = useServerTable<Provision>({ fetcher: (params) => configurationApi.list(params) })

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
      :value="table.items.value"
      :loading="table.loading.value"
      :total-records="table.total.value"
      :rows="table.perPage.value"
      :first="(table.page.value - 1) * table.perPage.value"
      lazy
      paginator
      size="small"
      data-key="id"
      @page="table.onPage"
      @row-click="(event) => openProvision(event.data.id)"
      style="cursor: pointer"
    >
      <Column field="name" header="Name" />
      <Column field="events" header="Events" />
      <Column field="requests" header="Requests" />
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
