<script setup lang="ts">
import { reactive, watch } from 'vue'
import Dialog from 'primevue/dialog'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Message from 'primevue/message'
import { userApi } from '@/api/endpoints/user.api'
import type { User } from '@/api/types/user'
import { useApiErrors } from '@/composables/useApiErrors'

const props = defineProps<{ user: User | null }>()
const emit = defineEmits<{ saved: [] }>()
const visible = defineModel<boolean>('visible', { required: true })

const { fieldErrors, generalError, run } = useApiErrors()

const form = reactive({ username: '', email: '', password: '' })

watch(visible, (isVisible) => {
  if (!isVisible) return
  form.username = props.user?.username ?? ''
  form.email = props.user?.email ?? ''
  form.password = ''
})

async function save() {
  const result = await run(() =>
    props.user
      ? userApi.update(props.user.uuid, { username: form.username, email: form.email, password: form.password || undefined })
      : userApi.create({ username: form.username, email: form.email, password: form.password }),
  )
  if (result !== undefined) {
    visible.value = false
    emit('saved')
  }
}
</script>

<template>
  <Dialog v-model:visible="visible" :header="user ? 'Edit user' : 'New user'" modal style="width: 26rem">
    <div class="fields">
      <Message v-if="generalError" severity="error" :closable="false">{{ generalError }}</Message>

      <div class="field">
        <label>Username</label>
        <InputText v-model="form.username" fluid />
        <small v-if="fieldErrors.username" class="error">{{ fieldErrors.username }}</small>
      </div>

      <div class="field">
        <label>Email</label>
        <InputText v-model="form.email" fluid />
        <small v-if="fieldErrors.email" class="error">{{ fieldErrors.email }}</small>
      </div>

      <div class="field">
        <label>Password {{ user ? '(leave blank to keep current)' : '' }}</label>
        <Password v-model="form.password" :feedback="false" toggle-mask fluid />
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
