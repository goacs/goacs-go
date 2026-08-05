<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import InputText from 'primevue/inputtext'
import Tag from 'primevue/tag'
import { useConfirm } from 'primevue/useconfirm'
import { FilterMatchMode } from '@primevue/core/api'
import { useServerTable } from '@/composables/useServerTable'
import { deviceApi } from '@/api/endpoints/device.api'
import type { Parameter } from '@/api/types/device'
import { flagToString } from '@/helpers/flags'
import ParameterDialog from '@/components/device/ParameterDialog.vue'

const props = defineProps<{ uuid: string }>()
const confirm = useConfirm()
const hasCachedParameters = ref(false)

const table = useServerTable<Parameter>({
  fetcher: (params) => deviceApi.getParameters(props.uuid, params),
  filters: {
    name: { value: '', matchMode: FilterMatchMode.CONTAINS },
    value: { value: '', matchMode: FilterMatchMode.CONTAINS },
    type: { value: '', matchMode: FilterMatchMode.CONTAINS },
    flags: { value: '', matchMode: FilterMatchMode.CONTAINS },
    cached_value: { value: '', matchMode: FilterMatchMode.CONTAINS },
  },
  perPage: 15,
})
const dialogVisible = ref(false)
const editingParameter = ref<Parameter | null>(null)

function openCreate() {
  editingParameter.value = null
  dialogVisible.value = true
}

function openEdit(parameter: Parameter) {
  editingParameter.value = parameter
  dialogVisible.value = true
}

function confirmDelete(parameter: Parameter) {
  confirm.require({
    message: `Delete parameter "${parameter.name}"?`,
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    accept: async () => {
      await deviceApi.deleteParameter(props.uuid, parameter.name)
      await table.reload()
    },
  })
}

onMounted(async () => {
  table.load()
  const cached = await deviceApi.getCachedParameters(props.uuid, { page: 1, per_page: 1 })
  hasCachedParameters.value = cached.total > 0
})
</script>

<template>
  <Panel header="Parameters" toggleable>
    <template #icons>
      <router-link v-if="hasCachedParameters" :to="{ name: 'devices-cached-params', params: { uuid: props.uuid } }">
        <Button icon="pi pi-database" text size="small" severity="secondary" />
      </router-link>
      <Button icon="pi pi-plus" text size="small" @click="openCreate" />
    </template>

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
      <Column header="Value" filter-field="value" :show-filter-menu="false">
        <template #body="{ data }">{{ data.valuestruct.value }}</template>
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" size="small" placeholder="Filter by value..." @input="filterCallback()" />
        </template>
      </Column>
      <Column header="Type" filter-field="type" :show-filter-menu="false">
        <template #body="{ data }">{{ data.valuestruct.type }}</template>
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" size="small" placeholder="Filter by type..." @input="filterCallback()" />
        </template>
      </Column>
      <Column field="flags" header="Flags" :show-filter-menu="false">
        <template #body="{ data }"><Tag :value="flagToString(data.flag)" severity="secondary" /></template>
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" size="small" placeholder="Filter by flags..." @input="filterCallback()" />
        </template>
      </Column>
      <Column header="Cached value" filter-field="cached_value" :show-filter-menu="false">
        <template #body="{ data }">
          <span v-if="data.cached_value == null">—</span>
          <Tag
            v-else
            :value="data.cached_value"
            :severity="data.cached_value === data.valuestruct.value ? 'success' : 'warning'"
          />
        </template>
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" size="small" placeholder="Filter by cached value..." @input="filterCallback()" />
        </template>
      </Column>
      <Column header="">
        <template #body="{ data }">
          <Button icon="pi pi-pencil" text size="small" @click="openEdit(data)" />
          <Button icon="pi pi-trash" text severity="danger" size="small" @click="confirmDelete(data)" />
        </template>
      </Column>
    </DataTable>

    <ParameterDialog
      v-model:visible="dialogVisible"
      :uuid="props.uuid"
      :parameter="editingParameter"
      @saved="table.reload"
    />
  </Panel>
</template>
