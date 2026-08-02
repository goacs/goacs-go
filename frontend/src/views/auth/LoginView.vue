<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { useAuthStore } from '@/stores/auth.store'

const authStore = useAuthStore()
const router = useRouter()
const route = useRoute()

const username = ref('')
const password = ref('')
const error = ref<string | null>(null)
const loading = ref(false)

async function submit() {
  error.value = null
  loading.value = true

  try {
    await authStore.login(username.value, password.value)
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/dashboard'
    await router.push(redirect)
  } catch {
    error.value = 'Invalid username or password'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <form class="login-form" @submit.prevent="submit">
    <h1>GoACS</h1>
    <Message v-if="error" severity="error" :closable="false">{{ error }}</Message>

    <div class="field">
      <label for="username">Username</label>
      <InputText id="username" v-model="username" autofocus fluid required />
    </div>

    <div class="field">
      <label for="password">Password</label>
      <Password id="password" v-model="password" :feedback="false" toggle-mask fluid required />
    </div>

    <Button type="submit" label="Sign in" :loading="loading" fluid />
  </form>
</template>

<style scoped>
.login-form {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

h1 {
  margin: 0 0 0.5rem;
  text-align: center;
}
</style>
