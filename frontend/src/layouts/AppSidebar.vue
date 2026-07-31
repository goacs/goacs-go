<script setup lang="ts">
interface NavItem {
  label: string
  icon: string
  to: string
}

const items: NavItem[] = [
  { label: 'Dashboard', icon: 'pi pi-home', to: '/dashboard' },
  { label: 'Devices', icon: 'pi pi-desktop', to: '/devices' },
  { label: 'Configuration', icon: 'pi pi-sliders-h', to: '/configuration' },
  { label: 'Templates', icon: 'pi pi-clone', to: '/templates' },
  { label: 'Files', icon: 'pi pi-folder', to: '/files' },
  { label: 'Settings', icon: 'pi pi-cog', to: '/settings' },
]
</script>

<template>
  <aside class="sidebar">
    <div class="brand">GoACS</div>
    <nav class="nav">
      <RouterLink
        v-for="item in items"
        :key="item.to"
        :to="item.to"
        class="nav-link"
        active-class="nav-link-active"
      >
        <i :class="item.icon"></i>
        <span>{{ item.label }}</span>
      </RouterLink>
    </nav>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  flex-shrink: 0;
  background: var(--p-surface-900, #1e1e2d);
  color: var(--p-surface-100, #e5e7eb);
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.brand {
  padding: 1.25rem 1.5rem;
  font-weight: 700;
  font-size: 1.25rem;
  letter-spacing: 0.02em;
}

.nav {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  padding: 0.5rem;
}

.nav-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.65rem 1rem;
  border-radius: 8px;
  color: inherit;
  text-decoration: none;
  opacity: 0.85;
  transition: background-color 0.15s ease;
}

.nav-link:hover {
  background: rgba(255, 255, 255, 0.08);
  opacity: 1;
}

.nav-link-active {
  background: var(--p-primary-color, #6366f1);
  opacity: 1;
}
</style>
