<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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

const CUSTOM = 'custom'
const SIZE_PRESETS = [
  { label: '10 MB', value: 10 * 1024 * 1024 },
  { label: '50 MB', value: 50 * 1024 * 1024 },
  { label: '100 MB', value: 100 * 1024 * 1024 },
  { label: '200 MB', value: 200 * 1024 * 1024 },
  { label: '1 GB', value: 1024 * 1024 * 1024 },
  { label: 'Enter manually…', value: CUSTOM },
]
const DEFAULT_UPLOAD_SIZE = 20 * 1024 * 1024

const direction = ref<'download' | 'upload'>('download')
const url = ref('')
const sizePreset = ref<number | typeof CUSTOM>(CUSTOM)
const customSize = ref<number | null>(null)
const numberOfConnections = ref<number | null>(null)
const saving = ref(false)

const size = computed(() => (sizePreset.value === CUSTOM ? customSize.value : sizePreset.value))

watch(visible, (isVisible) => {
  if (isVisible) {
    direction.value = 'download'
    url.value = ''
    sizePreset.value = CUSTOM
    customSize.value = null
    numberOfConnections.value = null
  }
})

async function save() {
  saving.value = true
  try {
    if (direction.value === 'download') {
      await deviceApi.runDownloadDiagnostics(props.uuid, {
        url: url.value || undefined,
        bytes: size.value ?? undefined,
        number_of_connections: numberOfConnections.value ?? undefined,
      })
    } else {
      await deviceApi.runUploadDiagnostics(props.uuid, {
        url: url.value || undefined,
        test_file_length: size.value ?? DEFAULT_UPLOAD_SIZE,
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
      <div class="field">
        <label>Size</label>
        <Select v-model="sizePreset" :options="SIZE_PRESETS" option-label="label" option-value="value" fluid />
      </div>
      <div class="field" v-if="sizePreset === CUSTOM">
        <label>Size in bytes</label>
        <InputNumber v-model="customSize" fluid :min="1" placeholder="Default 20 MiB" />
      </div>
      <div class="field">
        <label>Number of connections (optional)</label>
        <InputNumber v-model="numberOfConnections" fluid :min="1" />
      </div>
    </div>

    <template #footer>
      <Button label="Cancel" text @click="visible = false" />
      <Button label="Run" :loading="saving" @click="save" />
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
