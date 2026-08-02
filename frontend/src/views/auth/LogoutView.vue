<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth.store'

const authStore = useAuthStore()
const router = useRouter()

onMounted(async () => {
  await authStore.logout()
  await router.push('/auth/login')
})
</script>

<template>
  <p>Signing out...</p>
</template>
