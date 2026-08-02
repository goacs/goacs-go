<script setup lang="ts">
import VueJsonPretty from 'vue-json-pretty'
import 'vue-json-pretty/lib/styles.css'

defineProps<{ data: unknown }>()
</script>

<template>
  <VueJsonPretty :data="(data as any)" :deep="3" show-icon />
</template>
