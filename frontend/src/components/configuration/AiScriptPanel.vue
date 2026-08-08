<script setup lang="ts">
import { ref } from 'vue'
import { AxiosError } from 'axios'
import Textarea from 'primevue/textarea'
import Button from 'primevue/button'
import Message from 'primevue/message'
import { RouterLink } from 'vue-router'
import { aiApi } from '@/api/endpoints/ai.api'
import type { ApiEnvelope } from '@/api/types/apiEnvelope'

const visible = defineModel<boolean>('visible', { required: true })
const props = defineProps<{ events?: string; requests?: string }>()
const emit = defineEmits<{ insert: [string] }>()

const prompt = ref('')
const generating = ref(false)
const script = ref('')
const explanation = ref('')
const notConfigured = ref(false)
const errorMessage = ref('')

// The backend puts the human-readable error text in data.data instead of data.message -
// see the comment on http/controllers/ai.go's GenerateAiScript for why (a pre-existing
// bug in response.ResponseError discards the message argument everywhere in this API).
function extractErrorMessage(error: unknown): string {
  if (error instanceof AxiosError) {
    const body = error.response?.data as ApiEnvelope<unknown> | undefined
    if (typeof body?.data === 'string' && body.data) return body.data
  }
  return 'Something went wrong. Please try again.'
}

async function generate() {
  if (!prompt.value.trim()) return

  generating.value = true
  notConfigured.value = false
  errorMessage.value = ''
  script.value = ''
  explanation.value = ''

  try {
    const result = await aiApi.generateScript({
      prompt: prompt.value,
      events: props.events,
      requests: props.requests,
    })
    script.value = result.script
    explanation.value = result.explanation
  } catch (error) {
    const message = extractErrorMessage(error)
    if (message.toLowerCase().includes('not configured')) {
      notConfigured.value = true
    } else {
      errorMessage.value = message
    }
  } finally {
    generating.value = false
  }
}

function insertScript() {
  emit('insert', script.value)
  script.value = ''
  explanation.value = ''
  prompt.value = ''
}
</script>

<template>
  <div v-if="visible" class="ai-script-panel">
    <Message v-if="notConfigured" severity="warn" :closable="false">
      AI assistant is not configured. Set it up in
      <RouterLink to="/settings">Settings</RouterLink>.
    </Message>
    <Message v-else-if="errorMessage" severity="error" :closable="false">{{ errorMessage }}</Message>

    <div class="field">
      <label>Describe the script you want</label>
      <Textarea
        v-model="prompt"
        rows="3"
        autoResize
        fluid
        placeholder="e.g. rename the first SSID from the last 4 bytes of the WAN MAC address"
      />
    </div>

    <div class="actions">
      <Button label="Generate" icon="pi pi-sparkles" :loading="generating" @click="generate" />
    </div>

    <div v-if="script" class="result">
      <p v-if="explanation" class="explanation">{{ explanation }}</p>
      <pre class="code-preview">{{ script }}</pre>
      <div class="actions">
        <Button label="Insert as new script" icon="pi pi-plus" @click="insertScript" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.ai-script-panel {
  background: var(--p-content-background, #fff);
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 12px;
  padding: 1rem;
  margin-bottom: 1rem;
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.actions {
  display: flex;
  justify-content: flex-end;
}

.result {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  border-top: 1px solid var(--p-surface-200, #e5e7eb);
  padding-top: 0.75rem;
}

.explanation {
  margin: 0;
  font-size: 0.9rem;
  opacity: 0.85;
}

.code-preview {
  background: var(--p-surface-900, #1e1e1e);
  color: var(--p-surface-0, #f1f1f1);
  border-radius: 8px;
  padding: 0.75rem;
  font-family: monospace;
  font-size: 0.85rem;
  overflow-x: auto;
  white-space: pre;
  margin: 0;
}
</style>
