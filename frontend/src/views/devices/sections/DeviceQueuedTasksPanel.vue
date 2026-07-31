<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import { useConfirm } from 'primevue/useconfirm'
import { deviceApi } from '@/api/endpoints/device.api'
import type { Task } from '@/api/types/task'
import TaskDialog from '@/components/device/TaskDialog.vue'

const props = defineProps<{ uuid: string }>()
const confirm = useConfirm()

const tasks = ref<Task[]>([])
const loading = ref(false)
const dialogVisible = ref(false)

async function load() {
  loading.value = true
  try {
    tasks.value = await deviceApi.getTasks(props.uuid)
  } finally {
    loading.value = false
  }
}

function confirmDelete(task: Task) {
  confirm.require({
    message: `Delete task "${task.task}"?`,
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    accept: async () => {
      await deviceApi.deleteTask(props.uuid, task.id)
      await load()
    },
  })
}

onMounted(load)
</script>

<template>
  <Panel header="Queued tasks" toggleable>
    <template #icons>
      <Button icon="pi pi-plus" text size="small" @click="dialogVisible = true" />
    </template>

    <DataTable :value="tasks" :loading="loading" size="small">
      <Column field="task" header="Type" />
      <Column field="event" header="On request" />
      <Column field="not_before" header="Not before" />
      <Column header="">
        <template #body="{ data }">
          <Button icon="pi pi-trash" text severity="danger" size="small" @click="confirmDelete(data)" />
        </template>
      </Column>
      <template #empty>No queued tasks.</template>
    </DataTable>

    <TaskDialog v-model:visible="dialogVisible" :uuid="props.uuid" @saved="load" />
  </Panel>
</template>
