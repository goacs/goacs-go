<script setup lang="ts">
import { onMounted } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import { useDeviceStore } from '@/stores/device.store'
import DeviceHeaderBar from './sections/DeviceHeaderBar.vue'
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
    <ProgressSpinner v-if="deviceStore.loading && !deviceStore.currentDevice" style="width: 3rem" />

    <template v-else-if="deviceStore.currentDevice">
      <DeviceHeaderBar :device="deviceStore.currentDevice" />

      <div class="device-grid">
        <div class="device-grid__main">
          <DeviceParameterListPanel :uuid="props.uuid" />
          <DeviceLogsPanel :uuid="props.uuid" />
        </div>
        <div class="device-grid__side">
          <DeviceInfoPanel :device="deviceStore.currentDevice" />
          <DeviceSpeedtestPanel :uuid="props.uuid" />
          <DeviceQueuedTasksPanel :uuid="props.uuid" />
          <DeviceTemplatesPanel :uuid="props.uuid" />
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.device-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1.25rem;
  align-items: start;
}

.device-grid__main,
.device-grid__side {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
  min-width: 0;
}

@media (min-width: 1200px) {
  .device-grid {
    grid-template-columns: 1fr 300px;
  }
}
</style>
