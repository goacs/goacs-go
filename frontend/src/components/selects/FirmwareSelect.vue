<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Select from 'primevue/select'
import { fileApi } from '@/api/endpoints/file.api'

const value = defineModel<string>({ required: true })
const files = ref<string[]>([])

onMounted(async () => {
  const list = await fileApi.list()
  files.value = (list ?? []).filter((f) => !f.is_dir).map((f) => f.filename)
})
</script>

<template>
  <Select v-model="value" :options="files" placeholder="Firmware file" fluid filter />
</template>
