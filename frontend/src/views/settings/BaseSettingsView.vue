<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Select from 'primevue/select'
import Button from 'primevue/button'
import Panel from 'primevue/panel'
import ToggleSwitch from 'primevue/toggleswitch'
import { useToast } from 'primevue/usetoast'
import { configApi } from '@/api/endpoints/config.api'
import type { ConfigValues } from '@/api/types/settings'

const toast = useToast()
const loading = ref(false)
const saving = ref(false)

const KNOWN_KEYS = [
  'read_behaviour',
  'lookup_cache_ttl',
  'periodic_inform_interval',
  'connection_request_user',
  'connection_request_password',
  'webhook_after_provision',
  'webhook_timeout',
  'webhook_ssl_verify',
]

const known = reactive<Record<string, string>>({})
const extra = reactive<Array<{ key: string; value: string }>>([])

const extraRaw = ref<ConfigValues>({})

function splitConfig(values: ConfigValues) {
  KNOWN_KEYS.forEach((key) => (known[key] = values[key] ?? ''))
  extra.splice(
    0,
    extra.length,
    ...Object.entries(values)
      .filter(([key]) => !KNOWN_KEYS.includes(key))
      .map(([key, value]) => ({ key, value })),
  )
}

async function load() {
  loading.value = true
  try {
    extraRaw.value = await configApi.get()
    splitConfig(extraRaw.value)
  } finally {
    loading.value = false
  }
}

function addExtraRow() {
  extra.push({ key: '', value: '' })
}

function removeExtraRow(index: number) {
  extra.splice(index, 1)
}

const webhookSslVerify = computed({
  get: () => known.webhook_ssl_verify === '1' || known.webhook_ssl_verify === 'true',
  set: (value: boolean) => (known.webhook_ssl_verify = value ? '1' : '0'),
})

async function save() {
  saving.value = true
  try {
    const merged: ConfigValues = { ...known }
    for (const row of extra) {
      if (row.key.trim()) merged[row.key.trim()] = row.value
    }
    await configApi.save(merged)
    toast.add({ severity: 'success', summary: 'Settings saved', life: 3000 })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="settings-form">
    <Panel header="ACS behaviour">
      <div class="fields">
        <div class="field">
          <label>Read behaviour</label>
          <Select v-model="known.read_behaviour" :options="['boot', 'new', 'none']" fluid />
        </div>
        <div class="field">
          <label>Lookup cache TTL (seconds)</label>
          <InputNumber :model-value="Number(known.lookup_cache_ttl) || 0" @update:model-value="(v) => (known.lookup_cache_ttl = String(v))" fluid />
        </div>
        <div class="field">
          <label>Periodic inform interval (seconds)</label>
          <InputNumber :model-value="Number(known.periodic_inform_interval) || 0" @update:model-value="(v) => (known.periodic_inform_interval = String(v))" fluid />
        </div>
      </div>
    </Panel>

    <Panel header="Connection request credentials">
      <div class="fields">
        <div class="field">
          <label>Username</label>
          <InputText v-model="known.connection_request_user" fluid />
        </div>
        <div class="field">
          <label>Password</label>
          <InputText v-model="known.connection_request_password" type="password" fluid />
        </div>
      </div>
    </Panel>

    <Panel header="Webhooks">
      <div class="fields">
        <div class="field">
          <label>After-provision URL</label>
          <InputText v-model="known.webhook_after_provision" fluid />
        </div>
        <div class="field">
          <label>Timeout (ms)</label>
          <InputNumber :model-value="Number(known.webhook_timeout) || 0" @update:model-value="(v) => (known.webhook_timeout = String(v))" fluid />
        </div>
        <div class="field-inline">
          <ToggleSwitch v-model="webhookSslVerify" />
          <label>Verify SSL certificate</label>
        </div>
      </div>
    </Panel>

    <Panel header="Additional settings" toggleable collapsed>
      <div v-for="(row, index) in extra" :key="index" class="extra-row">
        <InputText v-model="row.key" placeholder="key" />
        <InputText v-model="row.value" placeholder="value" />
        <Button icon="pi pi-trash" text severity="danger" @click="removeExtraRow(index)" />
      </div>
      <Button label="Add setting" icon="pi pi-plus" text @click="addExtraRow" />
    </Panel>

    <div class="actions">
      <Button label="Save" :loading="saving" @click="save" />
    </div>
  </div>
</template>

<style scoped>
.settings-form {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  max-width: 40rem;
}

.fields {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.375rem;
}

.field-inline {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.extra-row {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.actions {
  display: flex;
  justify-content: flex-end;
}
</style>
