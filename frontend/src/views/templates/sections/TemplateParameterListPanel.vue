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
import { templateApi } from '@/api/endpoints/template.api'
import type { TemplateParameter } from '@/api/types/template'
import { flagToString } from '@/helpers/flags'
import TemplateParameterDialog from '@/components/device/TemplateParameterDialog.vue'

const props = defineProps<{ templateId: number }>()
const confirm = useConfirm()

const table = useServerTable<TemplateParameter>({
  fetcher: (params) => templateApi.getParameters(props.templateId, params),
  filters: {
    name: { value: '', matchMode: FilterMatchMode.CONTAINS },
    value: { value: '', matchMode: FilterMatchMode.CONTAINS },
    type: { value: '', matchMode: FilterMatchMode.CONTAINS },
  },
})
const dialogVisible = ref(false)
const editingParameter = ref<TemplateParameter | null>(null)

function openCreate() {
  editingParameter.value = null
  dialogVisible.value = true
}

function openEdit(parameter: TemplateParameter) {
  editingParameter.value = parameter
  dialogVisible.value = true
}

function confirmDelete(parameter: TemplateParameter) {
  confirm.require({
    message: `Delete parameter "${parameter.name}"?`,
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    accept: async () => {
      await templateApi.deleteParameter(props.templateId, parameter.uuid)
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
      <Column header="Flags">
        <template #body="{ data }"><Tag :value="flagToString(data.flag)" severity="secondary" /></template>
      </Column>
      <Column header="">
        <template #body="{ data }">
          <Button icon="pi pi-pencil" text size="small" @click="openEdit(data)" />
          <Button icon="pi pi-trash" text severity="danger" size="small" @click="confirmDelete(data)" />
        </template>
      </Column>
    </DataTable>

    <TemplateParameterDialog
      v-model:visible="dialogVisible"
      :template-id="props.templateId"
      :parameter="editingParameter"
      @saved="table.reload"
    />
  </Panel>
</template>
