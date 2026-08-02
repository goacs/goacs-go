<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import InputText from 'primevue/inputtext'
import Tag from 'primevue/tag'
import { useConfirm } from 'primevue/useconfirm'
import { useServerTable } from '@/composables/useServerTable'
import { deviceApi } from '@/api/endpoints/device.api'
import type { Parameter } from '@/api/types/device'
import { flagToString } from '@/helpers/flags'
import ParameterDialog from '@/components/device/ParameterDialog.vue'

const props = defineProps<{ uuid: string }>()
const confirm = useConfirm()

const table = useServerTable<Parameter>({ fetcher: (params) => deviceApi.getParameters(props.uuid, params) })
const dialogVisible = ref(false)
const editingParameter = ref<Parameter | null>(null)
const nameFilter = ref('')

function openCreate() {
  editingParameter.value = null
  dialogVisible.value = true
}

function openEdit(parameter: Parameter) {
  editingParameter.value = parameter
  dialogVisible.value = true
}

function applyNameFilter() {
  table.onFilterChange({ ...table.filter.value, name: nameFilter.value })
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

onMounted(() => table.load())
</script>

<template>
  <Panel header="Parameters" toggleable>
    <template #icons>
      <Button icon="pi pi-plus" text size="small" @click="openCreate" />
    </template>

    <div class="filter-row">
      <InputText v-model="nameFilter" placeholder="Filter by name..." size="small" @input="applyNameFilter" />
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
      @page="table.onPage"
    >
      <Column field="name" header="Name" />
      <Column header="Value">
        <template #body="{ data }">{{ data.valuestruct.value }}</template>
      </Column>
      <Column header="Flags">
        <template #body="{ data }"><Tag :value="flagToString(data.flag)" severity="secondary" /></template>
      </Column>
      <Column header="Cached value">
        <template #body="{ data }">
          <span v-if="data.cached_value == null">—</span>
          <Tag
            v-else
            :value="data.cached_value"
            :severity="data.cached_value === data.valuestruct.value ? 'success' : 'warning'"
          />
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

<style scoped>
.filter-row {
  margin-bottom: 0.75rem;
}
</style>
