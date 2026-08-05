<script setup lang="ts">
import { reactive, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Message from 'primevue/message'
import FlagSelect from '@/components/selects/FlagSelect.vue'
import XsdTypeSelect from '@/components/selects/XsdTypeSelect.vue'
import { templateApi } from '@/api/endpoints/template.api'
import { emptyFlag, type Flag } from '@/api/types/device'
import type { TemplateParameter } from '@/api/types/template'
import { useApiErrors } from '@/composables/useApiErrors'

const props = defineProps<{ templateId: number; parameter: TemplateParameter | null }>()
const emit = defineEmits<{ saved: [] }>()
const visible = defineModel<boolean>('visible', { required: true })

const { fieldErrors, generalError, run } = useApiErrors()

const form = reactive<{ name: string; value: string; type: string; flag: Flag }>({
  name: '',
  value: '',
  type: '',
  flag: emptyFlag(),
})

watch(visible, (isVisible) => {
  if (!isVisible) return
  if (props.parameter) {
    form.name = props.parameter.name
    form.value = props.parameter.valuestruct.value
    form.type = props.parameter.valuestruct.type
    form.flag = { ...props.parameter.flag }
  } else {
    form.name = ''
    form.value = ''
    form.type = ''
    form.flag = emptyFlag()
  }
})

async function save() {
  const result = await run(() =>
    props.parameter
      ? templateApi.updateParameter(props.templateId, props.parameter.uuid, form)
      : templateApi.createParameter(props.templateId, form),
  )
  if (result !== undefined) {
    visible.value = false
    emit('saved')
  }
}
</script>

<template>
  <Dialog v-model:visible="visible" :header="parameter ? 'Edit parameter' : 'Add parameter'" modal style="width: max(30rem, 50vw)">
    <div class="fields">
      <Message v-if="generalError" severity="error" :closable="false">{{ generalError }}</Message>

      <div class="field">
        <label>Name</label>
        <InputText v-model="form.name" fluid />
        <small v-if="fieldErrors.name" class="error">{{ fieldErrors.name }}</small>
      </div>

      <div class="field">
        <label>Value</label>
        <InputText v-model="form.value" fluid />
      </div>

      <div class="field">
        <label>Type</label>
        <XsdTypeSelect v-model="form.type" />
      </div>

      <div class="field">
        <label>Flags</label>
        <FlagSelect v-model="form.flag" />
      </div>
    </div>

    <template #footer>
      <Button label="Cancel" text @click="visible = false" />
      <Button label="Save" @click="save" />
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

.error {
  color: var(--p-red-500, #ef4444);
}
</style>
