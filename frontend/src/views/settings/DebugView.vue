<script setup lang="ts">
import { onMounted, ref } from 'vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import ToggleSwitch from 'primevue/toggleswitch'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import { useToast } from 'primevue/usetoast'
import { FilterMatchMode } from '@primevue/core/api'
import { useServerTable } from '@/composables/useServerTable'
import { configApi } from '@/api/endpoints/config.api'
import { deviceApi } from '@/api/endpoints/device.api'
import type { CPE } from '@/api/types/device'

const toast = useToast()
const debug = ref(false)
const debugNewDevices = ref(false)
const selectedUuids = ref<string[]>([])
const saving = ref(false)

const table = useServerTable<CPE>({
  fetcher: (params) => deviceApi.list(params),
  filters: { serial_number: { value: '', matchMode: FilterMatchMode.CONTAINS } },
})
const selectedDevices = ref<CPE[]>([])

async function load() {
  const settings = await configApi.getDebug()
  debug.value = settings.debug
  debugNewDevices.value = settings.debug_new_devices
  selectedUuids.value = settings.devices.map((d) => d.uuid)
  await table.load()
  selectedDevices.value = table.items.value.filter((d) => selectedUuids.value.includes(d.uuid))
}

async function save() {
  saving.value = true
  try {
    await configApi.saveDebug({
      debug: debug.value,
      debug_new_devices: debugNewDevices.value,
      devices: selectedDevices.value.map((d) => d.uuid),
    })
    toast.add({ severity: 'success', summary: 'Debug settings saved', life: 3000 })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="debug-view">
    <div class="toggle-row">
      <ToggleSwitch v-model="debug" />
      <span>Log full CWMP conversation (all devices)</span>
    </div>

    <div class="toggle-row">
      <ToggleSwitch v-model="debugNewDevices" />
      <span>Enable debug automatically for newly discovered devices</span>
    </div>

    <h3>Devices with debug enabled</h3>

    <DataTable
      v-model:selection="selectedDevices"
      v-model:filters="table.filters.value"
      :value="table.items.value"
      :loading="table.loading.value"
      :total-records="table.total.value"
      :rows="table.perPage.value"
      :first="(table.page.value - 1) * table.perPage.value"
      lazy
      paginator
      filter-display="row"
      size="small"
      data-key="uuid"
      @page="table.onPage"
      @filter="table.onFilter"
    >
      <Column selection-mode="multiple" header-style="width: 3rem" />
      <Column field="serial_number" header="Serial number" :show-filter-menu="false">
        <template #filter="{ filterModel, filterCallback }">
          <InputText v-model="filterModel.value" size="small" placeholder="Filter by serial..." @input="filterCallback()" />
        </template>
      </Column>
      <Column field="oui" header="OUI" />
    </DataTable>

    <div class="actions">
      <Button label="Save" :loading="saving" @click="save" />
    </div>
  </div>
</template>

<style scoped>
.debug-view {
  display: flex;
  flex-direction: column;
  gap: 1rem;
  max-width: 40rem;
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
}

.actions {
  display: flex;
  justify-content: flex-end;
}
</style>
