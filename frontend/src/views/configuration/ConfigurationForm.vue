<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import Button from 'primevue/button'
import Message from 'primevue/message'
import EventSelect from '@/components/selects/EventSelect.vue'
import RequestSelect from '@/components/selects/RequestSelect.vue'
import RuleItem from '@/components/rules/RuleItem.vue'
import ScriptCodeEditor from '@/components/code/ScriptCodeEditor.vue'
import AiScriptPanel from '@/components/configuration/AiScriptPanel.vue'
import type { Provision, ProvisionRule, ProvisionStoreRequest } from '@/api/types/configuration'
import { useApiErrors } from '@/composables/useApiErrors'

const props = defineProps<{ initial?: Provision | null; saving?: boolean }>()
const emit = defineEmits<{ submit: [ProvisionStoreRequest] }>()

const { fieldErrors, generalError, run } = useApiErrors()
const aiPanelVisible = ref(false)

// A provision only ever runs a single script, so form.script is kept at exactly one
// element (never [] or >1) - see the "script" getter/setter below.
function insertAiScript(script: string) {
  form.script = [script]
}

const form = reactive<ProvisionStoreRequest>({
  name: '',
  events: '',
  requests: '',
  script: [''],
  rules: [],
})

const eventsList = reactive<{ value: string[] }>({ value: [] })
const requestsList = reactive<{ value: string[] }>({ value: [] })

function applyInitial(provision: Provision | null | undefined) {
  form.name = provision?.name ?? ''
  form.script = provision?.script?.[0] !== undefined ? [provision.script[0]] : ['']
  form.rules = provision?.rules ? provision.rules.map((r) => ({ ...r })) : []
  eventsList.value = provision?.events ? provision.events.split(',').filter(Boolean) : []
  requestsList.value = provision?.requests ? provision.requests.split(',').filter(Boolean) : []
}

watch(() => props.initial, applyInitial, { immediate: true })

function addRule() {
  form.rules.push({ parameter: '', operator: '==', value: '' } as ProvisionRule)
}

function removeRule(index: number) {
  form.rules.splice(index, 1)
}

async function submitForm() {
  const payload: ProvisionStoreRequest = {
    ...form,
    events: eventsList.value.join(','),
    requests: requestsList.value.join(','),
  }

  const result = await run(async () => payload)
  if (result) emit('submit', result)
}
</script>

<template>
  <form class="config-form" @submit.prevent="submitForm">
    <Message v-if="generalError" severity="error" :closable="false">{{ generalError }}</Message>

    <div class="field">
      <label>Name</label>
      <InputText v-model="form.name" fluid />
      <small v-if="fieldErrors.name" class="error">{{ fieldErrors.name }}</small>
    </div>

    <div class="field-row">
      <div class="field">
        <label>Trigger events</label>
        <EventSelect v-model="eventsList.value" />
      </div>
      <div class="field">
        <label>Trigger requests</label>
        <RequestSelect v-model="requestsList.value" />
      </div>
    </div>

    <div class="section">
      <div class="section-header">
        <h3>Conditions</h3>
        <Button label="Add rule" icon="pi pi-plus" text size="small" @click="addRule" />
      </div>
      <RuleItem
        v-for="(rule, index) in form.rules"
        :key="index"
        :index="index"
        :model-value="rule"
        @update:model-value="(v) => (form.rules[index] = v)"
        @remove="removeRule(index)"
      />
      <p v-if="form.rules.length === 0" class="hint">No conditions - this provision always applies.</p>
    </div>

    <div class="section">
      <div class="section-header">
        <h3>Script</h3>
        <Button
          label="AI Assistant"
          icon="pi pi-sparkles"
          text
          size="small"
          @click="aiPanelVisible = !aiPanelVisible"
        />
      </div>
      <AiScriptPanel
        v-model:visible="aiPanelVisible"
        :events="eventsList.value.join(',')"
        :requests="requestsList.value.join(',')"
        :current-script="form.script[0]"
        @insert="insertAiScript"
      />
      <ScriptCodeEditor :model-value="form.script[0]" @update:model-value="(v) => (form.script = [v])" />
    </div>

    <div class="actions">
      <slot name="extra-actions" />
      <Button type="submit" label="Save" :loading="saving" />
    </div>
  </form>
</template>

<style scoped>
.config-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  max-width: max(46rem, 50vw);
}

.field,
.field-row > .field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.field-row {
  display: flex;
  gap: 1rem;
}

.field-row > .field {
  flex: 1;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

h3 {
  margin: 0 0 0.5rem;
}

.hint {
  opacity: 0.6;
  font-size: 0.85rem;
}

.actions {
  display: flex;
  gap: 0.5rem;
  justify-content: flex-end;
}

.error {
  color: var(--p-red-500, #ef4444);
}
</style>
