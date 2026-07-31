<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import Select from 'primevue/select'
import InputNumber from 'primevue/inputnumber'
import { deviceApi } from '@/api/endpoints/device.api'
import { templateApi, type TemplateListItem } from '@/api/endpoints/template.api'

const props = defineProps<{ uuid: string }>()
const emit = defineEmits<{ saved: [] }>()
const visible = defineModel<boolean>('visible', { required: true })

const templates = ref<TemplateListItem[]>([])
const templateId = ref<number | null>(null)
const priority = ref(100)
const saving = ref(false)

async function loadTemplates() {
  const response = await templateApi.list({ page: 1, per_page: 200 })
  templates.value = response.data
}

watch(visible, (isVisible) => {
  if (isVisible) {
    templateId.value = null
    priority.value = 100
    loadTemplates()
  }
})

onMounted(loadTemplates)

async function save() {
  if (!templateId.value) return
  saving.value = true
  try {
    await deviceApi.assignTemplate(props.uuid, { template_id: templateId.value, priority: priority.value })
    visible.value = false
    emit('saved')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Dialog v-model:visible="visible" header="Assign template" modal style="width: 26rem">
    <div class="fields">
      <div class="field">
        <label>Template</label>
        <Select v-model="templateId" :options="templates" option-label="name" option-value="id" fluid filter />
      </div>
      <div class="field">
        <label>Priority</label>
        <InputNumber v-model="priority" fluid />
      </div>
    </div>

    <template #footer>
      <Button label="Cancel" text @click="visible = false" />
      <Button label="Assign" :loading="saving" :disabled="!templateId" @click="save" />
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
</style>
