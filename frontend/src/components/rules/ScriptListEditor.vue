<script setup lang="ts">
import draggable from 'vuedraggable'
import Button from 'primevue/button'
import ScriptCodeEditor from '@/components/code/ScriptCodeEditor.vue'

const scripts = defineModel<string[]>({ required: true })

function addScript() {
  scripts.value = [...scripts.value, '']
}

function removeScript(index: number) {
  scripts.value = scripts.value.filter((_, i) => i !== index)
}

function updateScript(index: number, newValue: string) {
  const next = [...scripts.value]
  next[index] = newValue
  scripts.value = next
}
</script>

<template>
  <div class="script-list">
    <draggable v-model="scripts" item-key="index" handle=".drag-handle" class="script-items">
      <template #item="{ element, index }">
        <div class="script-item">
          <div class="script-item-header">
            <i class="pi pi-bars drag-handle"></i>
            <span>Script {{ index + 1 }}</span>
            <Button icon="pi pi-trash" text severity="danger" size="small" @click="removeScript(index)" />
          </div>
          <ScriptCodeEditor :model-value="element" @update:model-value="(v) => updateScript(index, v)" />
        </div>
      </template>
    </draggable>

    <Button label="Add script" icon="pi pi-plus" text @click="addScript" />
  </div>
</template>

<style scoped>
.script-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.script-item {
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 8px;
  padding: 0.5rem;
}

.script-item-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.4rem;
  font-size: 0.85rem;
}

.drag-handle {
  cursor: grab;
  opacity: 0.6;
}
</style>
