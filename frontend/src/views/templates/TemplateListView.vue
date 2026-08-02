<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import Dialog from 'primevue/dialog'
import InputText from 'primevue/inputtext'
import { useServerTable } from '@/composables/useServerTable'
import { templateApi, type TemplateListItem } from '@/api/endpoints/template.api'

const router = useRouter()
const table = useServerTable<TemplateListItem>({ fetcher: (params) => templateApi.list(params) })

const dialogVisible = ref(false)
const newName = ref('')
const saving = ref(false)

async function createTemplate() {
  if (!newName.value.trim()) return
  saving.value = true
  try {
    await templateApi.create(newName.value.trim())
    dialogVisible.value = false
    newName.value = ''
    await table.reload()
  } finally {
    saving.value = false
  }
}

function openTemplate(id: number) {
  router.push({ name: 'template-view', params: { id } })
}

onMounted(() => table.load())
</script>

<template>
  <div>
    <div class="header-row">
      <h1>Templates</h1>
      <Button label="New template" icon="pi pi-plus" @click="dialogVisible = true" />
    </div>

    <DataTable
      :value="table.items.value"
      :loading="table.loading.value"
      :total-records="table.total.value"
      :rows="table.perPage.value"
      :first="(table.page.value - 1) * table.perPage.value"
      lazy
      paginator
      size="small"
      data-key="id"
      @page="table.onPage"
      @row-click="(event) => openTemplate(event.data.id)"
      style="cursor: pointer"
    >
      <Column field="name" header="Name" />
      <Column field="parameter_count" header="Parameters" />
    </DataTable>

    <Dialog v-model:visible="dialogVisible" header="New template" modal style="width: max(24rem, 50vw)">
      <div class="field">
        <label>Name</label>
        <InputText v-model="newName" fluid autofocus @keyup.enter="createTemplate" />
      </div>
      <template #footer>
        <Button label="Cancel" text @click="dialogVisible = false" />
        <Button label="Create" :loading="saving" @click="createTemplate" />
      </template>
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

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}
</style>
