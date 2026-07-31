<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import TaskTypeSelect from '@/components/selects/TaskTypeSelect.vue'
import RequestSelect from '@/components/selects/RequestSelect.vue'
import FirmwareSelect from '@/components/selects/FirmwareSelect.vue'
import { deviceApi } from '@/api/endpoints/device.api'
import { payloadForTaskType } from '@/helpers/task'

const props = defineProps<{ uuid: string }>()
const emit = defineEmits<{ saved: [] }>()

const visible = defineModel<boolean>('visible', { required: true })

const taskType = ref('RunScript')
const events = ref<string[]>([])
const form = reactive<Record<string, string>>({ script: '', filename: '', path: '' })
const saving = ref(false)

watch(visible, (isVisible) => {
  if (isVisible) {
    taskType.value = 'RunScript'
    events.value = []
    form.script = ''
    form.filename = ''
    form.path = ''
  }
})

async function save() {
  saving.value = true
  try {
    await deviceApi.addTask(props.uuid, {
      event: events.value.join(','),
      task: taskType.value,
      payload: payloadForTaskType(taskType.value, form),
    })
    visible.value = false
    emit('saved')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Dialog v-model:visible="visible" header="Add task" modal style="width: 32rem">
    <div class="fields">
      <div class="field">
        <label>Task type</label>
        <TaskTypeSelect v-model="taskType" />
      </div>

      <div class="field">
        <label>Run on request</label>
        <RequestSelect v-model="events" />
      </div>

      <div v-if="taskType === 'RunScript'" class="field">
        <label>Script</label>
        <textarea v-model="form.script" rows="6" class="script-input"></textarea>
      </div>

      <div v-if="taskType === 'UploadFirmware'" class="field">
        <label>Firmware</label>
        <FirmwareSelect v-model="form.filename" />
      </div>

      <div v-if="taskType === 'AddObject' || taskType === 'DeleteObject'" class="field">
        <label>Object path</label>
        <InputText v-model="form.path" fluid />
      </div>
    </div>

    <template #footer>
      <Button label="Cancel" text @click="visible = false" />
      <Button label="Add task" :loading="saving" @click="save" />
    </template>
  </Dialog>
</template>

<style scoped>
.fields {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.script-input {
  font-family: ui-monospace, monospace;
  padding: 0.5rem;
  border: 1px solid var(--p-surface-300, #cbd5e1);
  border-radius: 6px;
}
</style>
