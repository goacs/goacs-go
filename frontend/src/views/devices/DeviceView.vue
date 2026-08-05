<script setup lang="ts">
import { onMounted } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import { useDeviceStore } from '@/stores/device.store'
import DeviceInfoPanel from './sections/DeviceInfoPanel.vue'
import DeviceQueuedTasksPanel from './sections/DeviceQueuedTasksPanel.vue'
import DeviceTemplatesPanel from './sections/DeviceTemplatesPanel.vue'
import DeviceSpeedtestPanel from './sections/DeviceSpeedtestPanel.vue'
import DeviceLogsPanel from './sections/DeviceLogsPanel.vue'
import DeviceParameterListPanel from './sections/DeviceParameterListPanel.vue'

const props = defineProps<{ uuid: string }>()
const deviceStore = useDeviceStore()

onMounted(() => deviceStore.fetchDevice(props.uuid))
</script>

<template>
  <div>
    <h1>Device</h1>

    <ProgressSpinner v-if="deviceStore.loading && !deviceStore.currentDevice" style="width: 3rem" />

    <div v-else-if="deviceStore.currentDevice" class="device-grid">
      <div class="device-grid__left">
        <DeviceInfoPanel :device="deviceStore.currentDevice" />
      </div>
      <div class="device-grid__right">
        <DeviceQueuedTasksPanel :uuid="props.uuid" />
        <DeviceTemplatesPanel :uuid="props.uuid" />
        <DeviceSpeedtestPanel :uuid="props.uuid" />
        <DeviceLogsPanel :uuid="props.uuid" />
      </div>
      <DeviceParameterListPanel :uuid="props.uuid" class="device-grid__full" />
    </div>
  </div>
</template>

<style scoped>
.device-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.25rem;
}

.device-grid__left,
.device-grid__right {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  min-width: 0;
}

@media (min-width: 1200px) {
  .device-grid {
    grid-template-columns: 1fr 1fr;
  }

  .device-grid__full {
    grid-column: 1 / -1;
  }
}
</style>
