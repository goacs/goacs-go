<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { FilterMatchMode } from '@primevue/core/api'
import DataTable, { type DataTableRowReorderEvent } from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import InputNumber, { type InputNumberBlurEvent } from 'primevue/inputnumber'
import Select from 'primevue/select'
import ToggleSwitch from 'primevue/toggleswitch'
import { useConfirm } from 'primevue/useconfirm'
import HowItWorksPanel from '@/components/configuration/HowItWorksPanel.vue'
import SimulatorDrawer from '@/components/configuration/SimulatorDrawer.vue'
import { useConfigurationStore } from '@/stores/configuration.store'
import { CWMP_EVENTS, CWMP_REQUESTS, type Provision } from '@/api/types/configuration'

const router = useRouter()
const confirm = useConfirm()
const store = useConfigurationStore()

const howItWorksVisible = ref(true)

const filters = ref({
  name: { value: null as string | null, matchMode: FilterMatchMode.CONTAINS },
  events: { value: null as string | null, matchMode: FilterMatchMode.CONTAINS },
  requests: { value: null as string | null, matchMode: FilterMatchMode.CONTAINS },
  enabled: { value: null as boolean | null, matchMode: FilterMatchMode.EQUALS },
})

const hasActiveFilters = computed(() => Object.values(filters.value).some((f) => f.value !== null && f.value !== ''))

const statusOptions = [
  { label: 'Enabled', value: true },
  { label: 'Disabled', value: false },
]

const eventLabels = new Map(CWMP_EVENTS.map((e) => [e.value, e.label]))
const requestLabels = new Map(CWMP_REQUESTS.map((r) => [r.value, r.label]))

function splitCsv(value: string): string[] {
  if (!value || !value.trim()) return []
  return value
    .split(',')
    .map((part) => part.trim())
    .filter((part) => part !== '')
}

function eventChips(provision: Provision) {
  return splitCsv(provision.events).map((v) => eventLabels.get(v) ?? v)
}

function requestChips(provision: Provision) {
  return splitCsv(provision.requests).map((v) => requestLabels.get(v) ?? v)
}

function onRowReorder(event: DataTableRowReorderEvent) {
  if (hasActiveFilters.value) return
  store.reorder(event.value as Provision[])
}

function onPriorityBlur(provision: Provision, event: InputNumberBlurEvent) {
  const newPriority = Number(event.value)
  if (!Number.isFinite(newPriority)) return

  const total = store.provisions.length
  const clamped = Math.min(Math.max(1, Math.round(newPriority)), total)
  if (clamped === provision.priority) return

  const others = store.provisions.filter((p) => p.id !== provision.id)
  others.splice(clamped - 1, 0, provision)
  store.reorder(others)
}

function edit(provision: Provision) {
  router.push({ name: 'configuration-edit', params: { id: provision.id } })
}

async function clone(provision: Provision) {
  await store.clone(provision)
}

function confirmRemove(provision: Provision) {
  confirm.require({
    message: `Delete provision "${provision.name}"?`,
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      await store.remove(provision)
    },
  })
}

onMounted(() => store.load())
</script>

<template>
  <div>
    <div class="header-row">
      <div>
        <h1>Configuration</h1>
        <p class="description">Provisioning rules run in priority order, filtered by trigger and conditions, before their scripts run.</p>
      </div>
      <div class="header-actions">
        <Button label="How it works" severity="secondary" outlined @click="howItWorksVisible = !howItWorksVisible" />
        <Button label="Simulate" icon="pi pi-play" severity="secondary" outlined @click="store.simulatorOpen = true" />
        <Button label="New provision" icon="pi pi-plus" @click="router.push({ name: 'configuration-create' })" />
      </div>
    </div>

    <HowItWorksPanel v-model:visible="howItWorksVisible" />

    <p v-if="hasActiveFilters" class="filter-hint">Reordering is disabled while a filter is active. Clear filters, or type a target position in the "#" column to move a provision regardless of filters.</p>

    <DataTable
      :value="store.provisions"
      :loading="store.loading"
      data-key="id"
      v-model:filters="filters"
      filter-display="row"
      paginator
      :rows="20"
      :rows-per-page-options="[10, 20, 50]"
      current-page-report-template="Showing {first} to {last} of {totalRecords}"
      paginator-template="FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink CurrentPageReport"
      scrollable
      scroll-height="600px"
      size="small"
      @row-reorder="onRowReorder"
    >
      <Column :row-reorder="!hasActiveFilters" style="width: 3rem" />

      <Column field="priority" header="#" style="width: 90px">
        <template #body="{ data }">
          <InputNumber
            :model-value="data.priority"
            :min="1"
            :max="store.provisions.length"
            :input-style="{ width: '4rem' }"
            @blur="(e: InputNumberBlurEvent) => onPriorityBlur(data, e)"
          />
        </template>
      </Column>

      <Column field="name" header="Name" :show-filter-menu="false" style="min-width: 220px">
        <template #body="{ data }">
          <div class="name-cell">
            <ToggleSwitch :model-value="data.enabled" @update:model-value="store.toggleEnabled(data)" />
            <span class="name" :title="data.name">{{ data.name }}</span>
            <span v-if="!data.enabled" class="disabled-tag">DISABLED</span>
          </div>
        </template>
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" placeholder="Search name" @input="filterCallback()" />
        </template>
      </Column>

      <Column field="events" header="Trigger events" :show-filter-menu="false" style="width: 180px">
        <template #body="{ data }">
          <div class="chip-line">
            <span v-if="eventChips(data).length === 0" class="chip chip-muted">Any event</span>
            <span v-for="c in eventChips(data)" :key="c" class="chip chip-event">{{ c }}</span>
          </div>
        </template>
        <template #filter="{ filterModel, filterCallback }">
          <Select
            v-model="filterModel.value"
            :options="CWMP_EVENTS"
            option-label="label"
            option-value="value"
            placeholder="Any"
            show-clear
            @change="filterCallback()"
          />
        </template>
      </Column>

      <Column field="requests" header="Trigger requests" :show-filter-menu="false" style="width: 180px">
        <template #body="{ data }">
          <div class="chip-line">
            <span v-if="requestChips(data).length === 0" class="chip chip-muted">Any request</span>
            <span v-for="c in requestChips(data)" :key="c" class="chip chip-request">{{ c }}</span>
          </div>
        </template>
        <template #filter="{ filterModel, filterCallback }">
          <Select
            v-model="filterModel.value"
            :options="CWMP_REQUESTS"
            option-label="label"
            option-value="value"
            placeholder="Any"
            show-clear
            @change="filterCallback()"
          />
        </template>
      </Column>

      <Column field="enabled" header="Status" :show-filter-menu="false" style="width: 130px">
        <template #body="{ data }">
          <span class="status-dot" :class="data.enabled ? 'status-on' : 'status-off'"></span>
        </template>
        <template #filter="{ filterModel, filterCallback }">
          <Select
            v-model="filterModel.value"
            :options="statusOptions"
            option-label="label"
            option-value="value"
            placeholder="All"
            show-clear
            @change="filterCallback()"
          />
        </template>
      </Column>

      <Column header="Conditions" style="width: 130px">
        <template #body="{ data }">
          <span class="badge" :class="(data.rules?.length ?? 0) > 0 ? 'badge-conditions' : 'badge-muted'">
            <i class="pi pi-filter"></i>
            {{ (data.rules?.length ?? 0) > 0 ? `${data.rules.length} conditions` : 'No conditions' }}
          </span>
        </template>
      </Column>

      <Column header="Scripts" style="width: 100px">
        <template #body="{ data }">
          <span class="badge badge-scripts"><i class="pi pi-code"></i> {{ data.script.length }} scripts</span>
        </template>
      </Column>

      <Column header="" style="width: 110px">
        <template #body="{ data }">
          <div class="actions-cell">
            <button class="action-btn" title="Edit" @click="edit(data)"><i class="pi pi-pencil"></i></button>
            <button class="action-btn" title="Clone" @click="clone(data)"><i class="pi pi-copy"></i></button>
            <button class="action-btn action-danger" title="Delete" @click="confirmRemove(data)"><i class="pi pi-trash"></i></button>
          </div>
        </template>
      </Column>
    </DataTable>

    <SimulatorDrawer />
  </div>
</template>

<style scoped>
.header-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 1rem;
}

.header-row h1 {
  font-size: 22px;
  margin: 0 0 0.25rem;
}

.description {
  margin: 0;
  color: var(--p-text-muted-color, #6b7280);
  font-size: 0.85rem;
}

.header-actions {
  display: flex;
  gap: 0.5rem;
  flex-shrink: 0;
}

.filter-hint {
  font-size: 0.75rem;
  color: var(--p-text-muted-color, #9ca3af);
  margin: 0 0 0.5rem;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}

.name {
  font-size: 13.5px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.disabled-tag {
  font-size: 10px;
  font-weight: 700;
  color: var(--p-text-muted-color, #9ca3af);
  background: var(--p-surface-100, #f3f4f6);
  border-radius: 5px;
  padding: 1px 5px;
  flex-shrink: 0;
}

.chip-line {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  flex-wrap: wrap;
}

.chip {
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 10.5px;
}

.chip-muted {
  background: var(--p-surface-100, #f3f4f6);
  color: var(--p-text-muted-color, #9ca3af);
}

.chip-event {
  background: var(--p-primary-50, #eef2ff);
  color: var(--p-primary-700, #4338ca);
}

.chip-request {
  background: var(--p-purple-50, #f3e8ff);
  color: var(--p-purple-700, #7e22ce);
}

.status-dot {
  display: inline-block;
  width: 9px;
  height: 9px;
  border-radius: 999px;
}

.status-on {
  background: var(--p-green-500, #22c55e);
}

.status-off {
  background: var(--p-surface-300, #d1d5db);
}

.badge {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  white-space: nowrap;
}

.badge-muted {
  background: var(--p-surface-100, #f3f4f6);
  color: var(--p-text-muted-color, #9ca3af);
}

.badge-conditions {
  background: var(--p-amber-50, #fef3c7);
  color: var(--p-amber-700, #b45309);
}

.badge-scripts {
  background: var(--p-primary-50, #eef2ff);
  color: var(--p-primary-700, #4338ca);
}

.actions-cell {
  display: flex;
  gap: 0.25rem;
  justify-content: flex-end;
}

.action-btn {
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--p-text-muted-color, #6b7280);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}

.action-btn:hover {
  background: var(--p-surface-100, #f3f4f6);
}

.action-btn.action-danger {
  color: var(--p-red-600, #dc2626);
}
</style>
