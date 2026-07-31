<script setup lang="ts">
import MultiSelect from 'primevue/multiselect'
import { CWMP_EVENTS } from '@/api/types/configuration'

const value = defineModel<string[]>({ required: true })
</script>

<template>
  <MultiSelect v-model="value" :options="CWMP_EVENTS" option-label="label" option-value="value" placeholder="Any event" fluid display="chip" />
</template>
