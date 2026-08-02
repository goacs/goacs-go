<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { templateApi } from '@/api/endpoints/template.api'
import type { Template } from '@/api/types/template'
import TemplateParameterListPanel from './sections/TemplateParameterListPanel.vue'

const props = defineProps<{ id: string }>()
const templateId = computed(() => Number(props.id))
const template = ref<Template | null>(null)

onMounted(async () => {
  template.value = await templateApi.get(templateId.value)
})
</script>

<template>
  <div>
    <h1>{{ template?.name ?? 'Template' }}</h1>
    <TemplateParameterListPanel :template-id="templateId" />
  </div>
</template>
