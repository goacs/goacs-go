<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import ConfigurationForm from './ConfigurationForm.vue'
import { configurationApi } from '@/api/endpoints/configuration.api'
import type { Provision, ProvisionStoreRequest } from '@/api/types/configuration'

const props = defineProps<{ id: string }>()
const provisionId = computed(() => Number(props.id))
const router = useRouter()
const confirm = useConfirm()
const toast = useToast()

const provision = ref<Provision | null>(null)
const saving = ref(false)

async function load() {
  provision.value = await configurationApi.get(provisionId.value)
}

async function save(payload: ProvisionStoreRequest) {
  saving.value = true
  try {
    provision.value = await configurationApi.update(provisionId.value, payload)
    toast.add({ severity: 'success', summary: 'Saved', life: 3000 })
  } finally {
    saving.value = false
  }
}

async function clone() {
  const cloned = await configurationApi.clone(provisionId.value)
  await router.push({ name: 'configuration-edit', params: { id: cloned.id } })
}

function confirmRemove() {
  confirm.require({
    message: `Delete provision "${provision.value?.name}"?`,
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      await configurationApi.delete(provisionId.value)
      await router.push({ name: 'configuration-list' })
    },
  })
}

onMounted(load)
</script>

<template>
  <div>
    <h1>{{ provision?.name ?? 'Provision' }}</h1>
    <ConfigurationForm :initial="provision" :saving="saving" @submit="save">
      <template #extra-actions>
        <Button label="Clone" icon="pi pi-copy" severity="secondary" text @click="clone" />
        <Button label="Delete" icon="pi pi-trash" severity="danger" text @click="confirmRemove" />
      </template>
    </ConfigurationForm>
  </div>
</template>
