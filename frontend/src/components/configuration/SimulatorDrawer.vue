<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import Drawer from 'primevue/drawer'
import Select from 'primevue/select'
import InputText from 'primevue/inputtext'
import Checkbox from 'primevue/checkbox'
import { useConfigurationStore } from '@/stores/configuration.store'
import { CWMP_EVENTS, CWMP_REQUESTS, type ProvisionSimulateResult } from '@/api/types/configuration'
import { configurationApi } from '@/api/endpoints/configuration.api'

const store = useConfigurationStore()

const allResults = ref<ProvisionSimulateResult[]>([])
const loading = ref(false)

let debounceHandle: ReturnType<typeof setTimeout> | undefined

async function runSimulation() {
  loading.value = true
  try {
    allResults.value = await configurationApi.simulate({
      event: store.simEvent,
      request: store.simRequest,
      root: store.simRoot,
      params: store.simParams,
    })
  } finally {
    loading.value = false
  }
}

function scheduleSimulation() {
  clearTimeout(debounceHandle)
  debounceHandle = setTimeout(runSimulation, 250)
}

watch(
  () => [store.simulatorOpen, store.simEvent, store.simRequest, store.simRoot, store.simParams] as const,
  ([open]) => {
    if (open) scheduleSimulation()
  },
  { deep: true },
)

const results = computed(() => (store.simOnlyMatches ? allResults.value.filter((r) => r.overall_match) : allResults.value))
const matchingResults = computed(() => allResults.value.filter((r) => r.overall_match))
const scriptsQueued = computed(() => matchingResults.value.reduce((sum, r) => sum + r.script_count, 0))

function addParam() {
  store.simParams = [...store.simParams, { key: '', value: '' }]
}

function removeParam(index: number) {
  store.simParams = store.simParams.filter((_, i) => i !== index)
}

function outcome(result: ProvisionSimulateResult): { label: string; cssClass: string } {
  if (!result.enabled) return { label: 'DISABLED', cssClass: 'pill-disabled' }
  if (result.overall_match) return { label: `RUNS ${result.script_count} SCRIPTS`, cssClass: 'pill-runs' }
  return { label: 'SKIPPED', cssClass: 'pill-skipped' }
}
</script>

<template>
  <Drawer v-model:visible="store.simulatorOpen" position="right" style="width: 460px" class="simulator-drawer">
    <template #header>
      <div>
        <div class="title">Simulate execution</div>
        <div class="subtext">{{ loading ? 'Updating…' : 'Updates as you change inputs' }}</div>
      </div>
    </template>

    <div class="inputs-section">
      <div class="trigger-inputs">
        <Select v-model="store.simEvent" :options="CWMP_EVENTS" option-label="label" option-value="value" placeholder="Trigger event" fluid show-clear />
        <Select v-model="store.simRequest" :options="CWMP_REQUESTS" option-label="label" option-value="value" placeholder="Trigger request" fluid show-clear />
      </div>
      <InputText v-model="store.simRoot" placeholder="Device root (optional, e.g. InternetGatewayDevice)" class="mono root-input" fluid />

      <div class="params-section">
        <div class="params-label">Simulated device parameters</div>
        <div v-for="(param, index) in store.simParams" :key="index" class="param-row">
          <InputText v-model="param.key" placeholder="parameter" class="mono" size="small" />
          <InputText v-model="param.value" placeholder="value" class="mono" size="small" />
          <button class="icon-btn" @click="removeParam(index)"><i class="pi pi-trash"></i></button>
        </div>
        <button class="add-param-btn" @click="addParam"><i class="pi pi-plus"></i> Add parameter</button>
      </div>
    </div>

    <div class="results-section">
      <div class="results-header">
        <span class="results-label">Evaluation, priority order</span>
        <label class="only-matches">
          <Checkbox v-model="store.simOnlyMatches" binary />
          Only matches
        </label>
      </div>

      <div v-for="result in results" :key="result.provision_id" class="result-card">
        <div class="result-top">
          <div class="priority-badge">{{ result.priority }}</div>
          <span class="result-name">{{ result.name }}</span>
          <span class="outcome-pill" :class="outcome(result).cssClass">{{ outcome(result).label }}</span>
        </div>

        <div class="status-badges">
          <span class="status-badge" :class="result.event_match ? 'status-pass' : 'status-fail'">
            <i class="pi" :class="result.event_match ? 'pi-check' : 'pi-times'"></i> Event
          </span>
          <span class="status-badge" :class="result.request_match ? 'status-pass' : 'status-fail'">
            <i class="pi" :class="result.request_match ? 'pi-check' : 'pi-times'"></i> Request
          </span>
          <span class="status-badge" :class="result.conditions_match ? 'status-pass' : 'status-fail'">
            <i class="pi" :class="result.conditions_match ? 'pi-check' : 'pi-times'"></i> Conditions
          </span>
        </div>

        <div v-for="(cr, i) in result.condition_results" :key="i" class="condition-line" :class="cr.passed ? 'status-pass' : 'status-fail'">
          <i class="pi" :class="cr.passed ? 'pi-check' : 'pi-times'"></i>
          <span class="mono">{{ cr.parameter }} {{ cr.operator }} {{ cr.value }} (actual: {{ cr.actual }})</span>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="footer-summary">
        <b>{{ matchingResults.length }}</b> of <b>{{ store.provisions.length }}</b> provisions run — <b>{{ scriptsQueued }}</b> scripts queued
      </div>
      <div v-if="matchingResults.length > 0" class="footer-order mono">
        {{ matchingResults.map((r) => r.name).join(' → ') }}
      </div>
    </template>
  </Drawer>
</template>

<style scoped>
.title {
  font-size: 15px;
  font-weight: 600;
}

.subtext {
  font-size: 0.75rem;
  color: var(--p-text-muted-color, #6b7280);
}

.inputs-section {
  border-bottom: 1px solid var(--p-surface-200, #e5e7eb);
  padding-bottom: 1rem;
  margin-bottom: 1rem;
}

.trigger-inputs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.root-input {
  margin-bottom: 0.75rem;
  font-size: 0.8rem;
}

.params-label {
  font-size: 12px;
  font-weight: 600;
  margin-bottom: 0.4rem;
}

.param-row {
  display: flex;
  gap: 0.4rem;
  margin-bottom: 0.4rem;
  align-items: center;
}

.mono {
  font-family: ui-monospace, monospace;
}

.icon-btn {
  border: none;
  background: transparent;
  cursor: pointer;
  color: var(--p-text-muted-color, #6b7280);
  flex-shrink: 0;
}

.add-param-btn {
  border: none;
  background: transparent;
  color: var(--p-primary-color, #6366f1);
  cursor: pointer;
  font-size: 0.8rem;
  display: flex;
  align-items: center;
  gap: 0.3rem;
}

.results-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.5rem;
}

.results-label {
  font-size: 11px;
  font-weight: 700;
  color: var(--p-text-muted-color, #6b7280);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.only-matches {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  font-size: 0.8rem;
  cursor: pointer;
}

.result-card {
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 9px;
  padding: 0.6rem;
  margin-bottom: 0.5rem;
}

.result-top {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin-bottom: 0.4rem;
}

.priority-badge {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  background: var(--p-surface-100, #f0f1f3);
  color: var(--p-text-muted-color, #4b5563);
  font-size: 11px;
  font-weight: 700;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.result-name {
  font-size: 13px;
  font-weight: 600;
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.outcome-pill {
  font-size: 10px;
  font-weight: 700;
  padding: 2px 8px;
  border-radius: 999px;
  flex-shrink: 0;
}

.pill-runs {
  background: var(--p-green-50, #dcfce7);
  color: var(--p-green-600, #16a34a);
}

.pill-skipped {
  background: var(--p-surface-100, #f3f4f6);
  color: var(--p-text-muted-color, #9ca3af);
}

.pill-disabled {
  background: var(--p-surface-100, #f3f4f6);
  color: var(--p-text-muted-color, #9ca3af);
}

.status-badges {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.35rem;
  font-size: 11px;
}

.status-badge {
  display: flex;
  align-items: center;
  gap: 0.2rem;
}

.status-pass {
  color: var(--p-green-600, #16a34a);
}

.status-fail {
  color: var(--p-red-600, #dc2626);
}

.condition-line {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 11px;
  margin-top: 0.15rem;
}

.footer-summary {
  font-size: 0.8rem;
}

.footer-order {
  margin-top: 0.35rem;
  font-size: 0.75rem;
  color: var(--p-text-muted-color, #6b7280);
  overflow-wrap: break-word;
}
</style>
