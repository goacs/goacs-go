<script setup lang="ts">
import { ref, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import { deviceApi } from '@/api/endpoints/device.api'

const props = defineProps<{ uuid: string }>()
const emit = defineEmits<{ saved: [] }>()
const visible = defineModel<boolean>('visible', { required: true })

const DIRECTIONS = [
  { label: 'Download', value: 'download' },
  { label: 'Upload', value: 'upload' },
]

const direction = ref<'download' | 'upload'>('download')
const url = ref('')
const bytes = ref<number | null>(null)
const testFileLength = ref<number | null>(20 * 1024 * 1024)
const numberOfConnections = ref<number | null>(null)
const saving = ref(false)

watch(visible, (isVisible) => {
  if (isVisible) {
    direction.value = 'download'
    url.value = ''
    bytes.value = null
    testFileLength.value = 20 * 1024 * 1024
    numberOfConnections.value = null
  }
})

async function save() {
  saving.value = true
  try {
    if (direction.value === 'download') {
      await deviceApi.runDownloadDiagnostics(props.uuid, {
        url: url.value || undefined,
        bytes: bytes.value ?? undefined,
        number_of_connections: numberOfConnections.value ?? undefined,
      })
    } else {
      await deviceApi.runUploadDiagnostics(props.uuid, {
        url: url.value || undefined,
        test_file_length: testFileLength.value ?? 20 * 1024 * 1024,
        number_of_connections: numberOfConnections.value ?? undefined,
      })
    }
    visible.value = false
    emit('saved')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <Dialog v-model:visible="visible" header="Run speed test" modal style="width: max(26rem, 50vw)">
    <div class="fields">
      <div class="field">
        <label>Direction</label>
        <Select v-model="direction" :options="DIRECTIONS" option-label="label" option-value="value" fluid />
      </div>
      <div class="field">
        <label>URL (optional)</label>
        <InputText v-model="url" fluid placeholder="Leave blank to use GoACS's own speedtest endpoint" />
      </div>
      <div class="field" v-if="direction === 'download'">
        <label>Bytes (optional)</label>
        <InputNumber v-model="bytes" fluid :min="1" placeholder="Default 20 MiB" />
      </div>
      <div class="field" v-if="direction === 'upload'">
        <label>Test file length (bytes)</label>
        <InputNumber v-model="testFileLength" fluid :min="1" />
      </div>
      <div class="field">
        <label>Number of connections (optional)</label>
        <InputNumber v-model="numberOfConnections" fluid :min="1" />
      </div>
    </div>

    <template #footer>
      <Button label="Cancel" text @click="visible = false" />
      <Button
        label="Run"
        :loading="saving"
        :disabled="direction === 'upload' && !testFileLength"
        @click="save"
      />
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
