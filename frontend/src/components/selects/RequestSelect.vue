<script setup lang="ts">
import MultiSelect from 'primevue/multiselect'
import { CWMP_REQUESTS } from '@/api/types/configuration'

const value = defineModel<string[]>({ required: true })
</script>

<template>
  <MultiSelect v-model="value" :options="CWMP_REQUESTS" option-label="label" option-value="value" placeholder="Any request" fluid display="chip" />
</template>
