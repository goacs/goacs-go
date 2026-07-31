<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import ConfigurationForm from './ConfigurationForm.vue'
import { configurationApi } from '@/api/endpoints/configuration.api'
import type { ProvisionStoreRequest } from '@/api/types/configuration'

const router = useRouter()
const saving = ref(false)

async function save(payload: ProvisionStoreRequest) {
  saving.value = true
  try {
    const created = await configurationApi.create(payload)
    await router.push({ name: 'configuration-edit', params: { id: created.id } })
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <h1>New provision</h1>
    <ConfigurationForm :saving="saving" @submit="save" />
  </div>
</template>
