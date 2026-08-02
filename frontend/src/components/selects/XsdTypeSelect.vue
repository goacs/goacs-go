<script setup lang="ts">
import Select from 'primevue/select'

const TYPES = ['xsd:string', 'xsd:int', 'xsd:unsignedInt', 'xsd:boolean', 'xsd:dateTime', 'xsd:base64']

const value = defineModel<string>({ required: true })
</script>

<template>
  <Select v-model="value" :options="TYPES" placeholder="Type" fluid />
</template>
