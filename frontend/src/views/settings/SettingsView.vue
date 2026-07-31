<script setup lang="ts">
const tabs = [
  { label: 'General', to: '/settings' },
  { label: 'Users', to: '/settings/users' },
  { label: 'Debug', to: '/settings/debug' },
]
</script>

<template>
  <div>
    <h1>Settings</h1>
    <nav class="tabs">
      <RouterLink v-for="tab in tabs" :key="tab.to" :to="tab.to" class="tab" active-class="tab-active" exact-active-class="tab-active">
        {{ tab.label }}
      </RouterLink>
    </nav>
    <RouterView />
  </div>
</template>

<style scoped>
.tabs {
  display: flex;
  gap: 1rem;
  border-bottom: 1px solid var(--p-surface-200, #e5e7eb);
  margin-bottom: 1.25rem;
}

.tab {
  padding: 0.5rem 0.25rem;
  text-decoration: none;
  opacity: 0.65;
  border-bottom: 2px solid transparent;
}

.tab-active {
  opacity: 1;
  border-bottom-color: var(--p-primary-color, #6366f1);
}
</style>
