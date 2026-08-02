<script setup lang="ts">
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import Button from 'primevue/button'
import { RULE_OPERATORS, type ProvisionRule } from '@/api/types/configuration'

defineProps<{ index: number }>()
const emit = defineEmits<{ remove: [] }>()
const rule = defineModel<ProvisionRule>({ required: true })
</script>

<template>
  <div class="rule-row">
    <InputText v-model="rule.parameter" placeholder="Parameter name" class="parameter" />
    <Select v-model="rule.operator" :options="RULE_OPERATORS" placeholder="Operator" class="operator" />
    <InputText v-model="rule.value" placeholder="Value" class="value" />
    <Button icon="pi pi-trash" text severity="danger" @click="emit('remove')" />
  </div>
</template>

<style scoped>
.rule-row {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  margin-bottom: 0.5rem;
}

.parameter {
  flex: 2;
}

.operator {
  flex: 1;
}

.value {
  flex: 1;
}
</style>
