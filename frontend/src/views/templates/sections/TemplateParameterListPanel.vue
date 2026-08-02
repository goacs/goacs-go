<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import Tag from 'primevue/tag'
import { useConfirm } from 'primevue/useconfirm'
import { useServerTable } from '@/composables/useServerTable'
import { templateApi } from '@/api/endpoints/template.api'
import type { TemplateParameter } from '@/api/types/template'
import { flagToString } from '@/helpers/flags'
import TemplateParameterDialog from '@/components/device/TemplateParameterDialog.vue'

const props = defineProps<{ templateId: number }>()
const confirm = useConfirm()

const table = useServerTable<TemplateParameter>({
  fetcher: (params) => templateApi.getParameters(props.templateId, params),
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
