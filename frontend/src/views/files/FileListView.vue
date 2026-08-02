<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import FileUpload from 'primevue/fileupload'
import ProgressBar from 'primevue/progressbar'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import dayjs from 'dayjs'
import { fileApi } from '@/api/endpoints/file.api'
import type { FileInfo } from '@/api/types/file'

const files = ref<FileInfo[]>([])
const loading = ref(false)
const uploadDialogVisible = ref(false)
const uploadProgress = ref(0)
const uploading = ref(false)
const confirm = useConfirm()
const toast = useToast()

async function load() {
  loading.value = true
  try {
    files.value = (await fileApi.list()) ?? []
  } finally {
    loading.value = false
  }
}

async function upload(event: { files: File[] }) {
  const file = event.files[0]
  if (!file) return

  uploading.value = true
  uploadProgress.value = 0
  try {
    await fileApi.upload(file, (percent) => (uploadProgress.value = percent))
    uploadDialogVisible.value = false
    toast.add({ severity: 'success', summary: 'Uploaded', life: 3000 })
    await load()
  } catch {
    toast.add({ severity: 'error', summary: 'Upload failed', life: 3000 })
  } finally {
    uploading.value = false
  }
}

function confirmDelete(file: FileInfo) {
  confirm.require({
    message: `Delete file "${file.filename}"?`,
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    accept: async () => {
      await fileApi.delete(file.filename)
      await load()
    },
  })
}

function formatSize(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function formatDate(value: string) {
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}

onMounted(load)
</script>

<template>
  <div>
    <div class="header-row">
      <h1>Files</h1>
      <Button label="Upload" icon="pi pi-upload" @click="uploadDialogVisible = true" />
    </div>

    <DataTable :value="files" :loading="loading" size="small">
      <Column field="filename" header="Filename" />
      <Column header="Size">
        <template #body="{ data }">{{ formatSize(data.size) }}</template>
      </Column>
      <Column header="Modified">
        <template #body="{ data }">{{ formatDate(data.mod_time) }}</template>
      </Column>
      <Column header="">
        <template #body="{ data }">
          <a :href="fileApi.downloadUrl(data.filename)" target="_blank" rel="noopener">
            <Button icon="pi pi-download" text size="small" />
          </a>
          <Button icon="pi pi-trash" text severity="danger" size="small" @click="confirmDelete(data)" />
        </template>
      </Column>
      <template #empty>No files uploaded yet.</template>
    </DataTable>

    <Dialog v-model:visible="uploadDialogVisible" header="Upload firmware image" modal style="width: max(28rem, 50vw)">
      <FileUpload
        mode="basic"
        :auto="false"
        custom-upload
        choose-label="Choose file"
        @select="upload({ files: $event.files })"
      />
      <ProgressBar v-if="uploading" :value="uploadProgress" class="progress" />
    </Dialog>
  </div>
</template>

<style scoped>
.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 1rem;
}

.progress {
  margin-top: 1rem;
}
</style>
