<script setup lang="ts">
import Checkbox from 'primevue/checkbox'
import type { Flag } from '@/api/types/device'

const flag = defineModel<Flag>({ required: true })

const options: Array<[keyof Flag, string]> = [
  ['read', 'Read'],
  ['write', 'Write'],
  ['send', 'Send'],
  ['add_object', 'Object'],
  ['system', 'System'],
  ['periodic_read', 'Periodic'],
  ['important', 'Important'],
]
</script>

<template>
  <div class="flags">
    <label v-for="[key, label] in options" :key="key" class="flag-option">
      <Checkbox v-model="flag[key]" binary />
      <span>{{ label }}</span>
    </label>
  </div>
</template>

<style scoped>
.flags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.flag-option {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.85rem;
}
</style>
