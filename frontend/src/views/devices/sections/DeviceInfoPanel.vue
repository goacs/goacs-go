<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import Panel from 'primevue/panel'
import Button from 'primevue/button'
import ToggleSwitch from 'primevue/toggleswitch'
import { useConfirm } from 'primevue/useconfirm'
import { useToast } from 'primevue/usetoast'
import { useDeviceStore } from '@/stores/device.store'
import { configApi } from '@/api/endpoints/config.api'
import type { CPE } from '@/api/types/device'

const props = defineProps<{ device: CPE }>()
const router = useRouter()
const confirm = useConfirm()
const toast = useToast()
const deviceStore = useDeviceStore()
const busy = ref<string | null>(null)

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
</script>

<template>
  <Panel header="Device info">
    <dl class="info-grid">
      <dt>Serial number</dt>
      <dd>{{ device.serial_number }}</dd>
      <dt>OUI</dt>
      <dd>{{ device.oui }}</dd>
      <dt>Software version</dt>
      <dd>{{ device.software_version }}</dd>
      <dt>Hardware version</dt>
      <dd>{{ device.hardware_version }}</dd>
      <dt>IP address</dt>
      <dd>{{ device.ip_address?.IP || '-' }}</dd>
      <dt>Last update</dt>
      <dd>{{ device.updated_at }}</dd>
    </dl>

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
      <Button
        label="Kick"
        icon="pi pi-send"
        size="small"
        severity="secondary"
        :loading="busy === 'kick'"
        @click="run('kick', () => deviceStore.kick(device.uuid), 'Kick sent')"
      />
      <router-link :to="{ name: 'devices-cached-params', params: { uuid: device.uuid } }">
        <Button label="Cached parameters" icon="pi pi-database" size="small" severity="secondary" />
      </router-link>
      <Button label="Delete" icon="pi pi-trash" size="small" severity="danger" @click="confirmDelete" />
    </div>

    <div class="debug-toggle">
      <ToggleSwitch :model-value="device.debug" @update:model-value="toggleDebug" />
      <span>Debug logging for this device</span>
    </div>
  </Panel>
</template>

<style scoped>
.info-grid {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: 0.4rem 1rem;
  margin: 0 0 1rem;
}

dt {
  opacity: 0.6;
  font-size: 0.85rem;
}

dd {
  margin: 0;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-bottom: 1rem;
}

.debug-toggle {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  font-size: 0.85rem;
}
</style>
