<script setup lang="ts">
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'
</script>

<template>
  <div class="shell">
    <AppSidebar />
    <div class="content">
      <AppHeader />
      <main class="page">
        <RouterView />
      </main>
    </div>
    <Toast />
    <ConfirmDialog />
  </div>
</template>

<style scoped>
.shell {
  display: flex;
  min-height: 100vh;
}

.content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.page {
  flex: 1;
  padding: 1.5rem;
  overflow-x: auto;
}
</style>
