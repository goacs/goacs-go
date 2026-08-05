<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import ToggleSwitch from 'primevue/toggleswitch'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import dayjs from 'dayjs'
import { useDeviceStore } from '@/stores/device.store'
import { configApi } from '@/api/endpoints/config.api'
import type { CPE } from '@/api/types/device'

const props = defineProps<{ device: CPE }>()
const router = useRouter()
const confirm = useConfirm()
const toast = useToast()
const deviceStore = useDeviceStore()
const busy = ref<string | null>(null)

// No dedicated online/offline field from the API today - approximate from recency
// of the last inform. Tune the threshold (or swap for a real field) as needed.
const isOnline = computed(() => dayjs().diff(dayjs(props.device.updated_at), 'minute') < 15)

async function run(action: string, fn: () => Promise<unknown>, successMessage: string) {
  busy.value = action
  try {
    await fn()
    toast.add({ severity: 'success', summary: successMessage, life: 3000 })
  } catch {
    toast.add({ severity: 'error', summary: 'Action failed', life: 3000 })
  } finally {
    busy.value = null
  }
}

// There's no per-device debug endpoint - toggling means re-sending the whole
// debug-enabled device list via /api/settings/debug (see DebugView).
async function toggleDebug(value: boolean) {
  await run(
    'debug',
    async () => {
      const current = await configApi.getDebug()
      const uuids = new Set(current.devices.map((d) => d.uuid))
      if (value) uuids.add(props.device.uuid)
      else uuids.delete(props.device.uuid)

      await configApi.saveDebug({
        debug: current.debug,
        debug_new_devices: current.debug_new_devices,
        devices: Array.from(uuids),
      })
      props.device.debug = value
    },
    'Debug flag updated',
  )
}

function formatDate(value: string) {
  return dayjs(value).format('YYYY-MM-DD HH:mm:ss')
}

function confirmDelete() {
  confirm.require({
    message: `Delete device "${props.device.serial_number}"? This cannot be undone.`,
    header: 'Confirm delete',
    icon: 'pi pi-exclamation-triangle',
    acceptClass: 'p-button-danger',
    accept: async () => {
      await deviceStore.deleteDevice(props.device.uuid)
      await router.push({ name: 'devices-list' })
    },
  })
}
</script>

<template>
  <div class="header-bar">
    <div class="identity">
      <div class="crumb">Devices / {{ device.serial_number }}</div>
      <div class="title-row">
        <h1>{{ device.serial_number }}</h1>
        <Tag :value="isOnline ? 'Online' : 'Offline'" :severity="isOnline ? 'success' : 'secondary'" />
      </div>
      <div class="sub">{{ device.manufacturer }} · OUI {{ device.oui }} · last inform {{ formatDate(device.updated_at) }}</div>
    </div>

    <div class="actions">
      <Button
        label="Provision now"
        icon="pi pi-bolt"
        size="small"
        :loading="busy === 'provision'"
        @click="run('provision', () => deviceStore.provisionNow(device.uuid), 'Provision requested')"
      />
      <Button
        label="Lookup now"
        icon="pi pi-search"
        size="small"
        severity="secondary"
        :loading="busy === 'lookup'"
        @click="run('lookup', () => deviceStore.lookupNow(device.uuid), 'Lookup requested')"
      />
      <Button
        label="Clear cache"
        icon="pi pi-eraser"
        size="small"
        severity="secondary"
        :loading="busy === 'cache'"
        @click="run('cache', () => deviceStore.clearCache(device.uuid), 'Cache cleared')"
      />
      <div class="divider" />
      <div class="debug">
        <ToggleSwitch :model-value="device.debug" @update:model-value="toggleDebug" />
        <span>Debug</span>
      </div>
      <Button label="Delete" icon="pi pi-trash" size="small" severity="danger" outlined @click="confirmDelete" />
    </div>
  </div>
</template>

<style scoped>
.header-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 1rem;
  padding: 1rem 1.5rem;
  background: var(--p-surface-0, #fff);
  border: 1px solid var(--p-surface-200, #e5e7eb);
  border-radius: 10px;
  margin-bottom: 1.25rem;
}

.crumb {
  font-size: 0.7rem;
  font-weight: 600;
  color: rgba(0, 0, 0, 0.4);
  margin-bottom: 0.2rem;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.title-row h1 {
  margin: 0;
  font-size: 1.15rem;
  font-family: ui-monospace, monospace;
}

.sub {
  font-size: 0.75rem;
  color: rgba(0, 0, 0, 0.45);
  margin-top: 0.2rem;
}

.actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.divider {
  width: 1px;
  height: 22px;
  background: var(--p-surface-200, #e5e7eb);
  margin: 0 0.15rem;
}

.debug {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  font-size: 0.75rem;
  color: rgba(0, 0, 0, 0.6);
  padding: 0 0.25rem;
}
</style>
