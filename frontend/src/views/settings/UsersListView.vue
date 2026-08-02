<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import { useConfirm } from 'primevue/useconfirm'
import { useServerTable } from '@/composables/useServerTable'
import { userApi } from '@/api/endpoints/user.api'
import type { User } from '@/api/types/user'
import { useAuthStore } from '@/stores/auth.store'
import UserDialog from '@/components/settings/UserDialog.vue'

const confirm = useConfirm()
const authStore = useAuthStore()
const table = useServerTable<User>({ fetcher: (params) => userApi.list(params) })
const dialogVisible = ref(false)
const editingUser = ref<User | null>(null)

function openCreate() {
  editingUser.value = null
  dialogVisible.value = true
}

function openEdit(user: User) {
  editingUser.value = user
  dialogVisible.value = true
}

function confirmDelete(user: User) {
  confirm.require({
    message: `Delete user "${user.username}"?`,
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    accept: async () => {
      await userApi.delete(user.uuid)
      await table.reload()
    },
  })
}

onMounted(() => table.load())
</script>

<template>
  <div>
    <div class="header-row">
      <Button label="New user" icon="pi pi-plus" @click="openCreate" />
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
      @page="table.onPage"
    >
      <Column field="username" header="Username" />
      <Column field="email" header="Email" />
      <Column header="">
        <template #body="{ data }">
          <Button icon="pi pi-pencil" text size="small" @click="openEdit(data)" />
          <Button
            icon="pi pi-trash"
            text
            severity="danger"
            size="small"
            :disabled="data.uuid === authStore.user?.uuid"
            @click="confirmDelete(data)"
          />
        </template>
      </Column>
    </DataTable>

    <UserDialog v-model:visible="dialogVisible" :user="editingUser" @saved="table.reload" />
  </div>
</template>

<style scoped>
.header-row {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 1rem;
}
</style>
