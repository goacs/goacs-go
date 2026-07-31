<script setup lang="ts">
import Select from 'primevue/select'
import { TASK_TYPES } from '@/api/types/task'

const value = defineModel<string>({ required: true })
</script>

<template>
  <Select v-model="value" :options="[...TASK_TYPES]" placeholder="Task type" fluid />
</template>
