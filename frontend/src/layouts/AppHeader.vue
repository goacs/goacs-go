<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import Menu from 'primevue/menu'
import Button from 'primevue/button'
import { useAuthStore } from '@/stores/auth.store'

const authStore = useAuthStore()
const router = useRouter()
const menu = ref()

const menuItems = [
  {
    label: 'Logout',
    icon: 'pi pi-sign-out',
    command: async () => {
      await authStore.logout()
      await router.push('/auth/login')
    },
  },
]

function toggleMenu(event: MouseEvent) {
  menu.value?.toggle(event)
}
</script>

<template>
  <header class="header">
    <div />
    <div class="account">
      <span class="username">{{ authStore.user?.username }}</span>
      <Button icon="pi pi-user" rounded text @click="toggleMenu" aria-haspopup="true" />
      <Menu ref="menu" :model="menuItems" :popup="true" />
    </div>
  </header>
</template>

<style scoped>
.header {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 1.25rem;
  border-bottom: 1px solid var(--p-surface-200, #e5e7eb);
}

.account {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.username {
  font-size: 0.9rem;
  opacity: 0.75;
}
</style>
