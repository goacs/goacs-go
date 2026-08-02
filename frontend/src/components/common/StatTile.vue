<script setup lang="ts">
defineProps<{
  label: string
  value: number | string
  icon: string
  severity?: 'info' | 'success' | 'warn' | 'danger'
}>()
</script>

<template>
  <div class="tile" :class="severity ?? 'info'">
    <i :class="icon" class="tile-icon"></i>
    <div>
      <div class="tile-value">{{ value }}</div>
      <div class="tile-label">{{ label }}</div>
    </div>
  </div>
</template>

<style scoped>
.tile {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 1.25rem;
  border-radius: 10px;
  background: var(--p-surface-0, #fff);
  border: 1px solid var(--p-surface-200, #e5e7eb);
}

.tile-icon {
  font-size: 1.75rem;
  opacity: 0.8;
}

.tile-value {
  font-size: 1.5rem;
  font-weight: 700;
}

.tile-label {
  font-size: 0.85rem;
  opacity: 0.65;
}

.tile.danger .tile-icon {
  color: var(--p-red-500, #ef4444);
}

.tile.success .tile-icon {
  color: var(--p-green-500, #22c55e);
}

.tile.warn .tile-icon {
  color: var(--p-yellow-500, #eab308);
}
</style>
