<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import { useConfirm } from 'primevue/useconfirm'
import { deviceApi } from '@/api/endpoints/device.api'
import type { CPETemplate } from '@/api/types/device'
import TemplateAssignDialog from '@/components/device/TemplateAssignDialog.vue'

const props = defineProps<{ uuid: string }>()
const confirm = useConfirm()

const templates = ref<CPETemplate[]>([])
const loading = ref(false)
const dialogVisible = ref(false)

async function load() {
  loading.value = true
  try {
    templates.value = await deviceApi.getTemplates(props.uuid)
  } finally {
    loading.value = false
  }
}

function confirmUnassign(template: CPETemplate) {
  confirm.require({
    message: `Unassign template "${template.name}"?`,
    header: 'Confirm',
    icon: 'pi pi-exclamation-triangle',
    accept: async () => {
      await deviceApi.unassignTemplate(props.uuid, template.id)
      await load()
    },
  })
}

onMounted(load)
</script>

<template>
  <Panel header="Templates" toggleable>
    <template #icons>
      <Button icon="pi pi-plus" text size="small" @click="dialogVisible = true" />
    </template>

    <DataTable :value="templates" :loading="loading" size="small">
      <Column field="name" header="Name" />
      <Column field="priority" header="Priority" />
      <Column header="">
        <template #body="{ data }">
          <Button icon="pi pi-times" text severity="danger" size="small" @click="confirmUnassign(data)" />
        </template>
      </Column>
      <template #empty>No templates assigned.</template>
    </DataTable>

    <TemplateAssignDialog v-model:visible="dialogVisible" :uuid="props.uuid" @saved="load" />
  </Panel>
</template>
