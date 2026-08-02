<script setup lang="ts">
import { onMounted } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import { useDeviceStore } from '@/stores/device.store'
import DeviceInfoPanel from './sections/DeviceInfoPanel.vue'
import DeviceQueuedTasksPanel from './sections/DeviceQueuedTasksPanel.vue'
import DeviceTemplatesPanel from './sections/DeviceTemplatesPanel.vue'
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
      <DeviceInfoPanel :device="deviceStore.currentDevice" />
      <DeviceQueuedTasksPanel :uuid="props.uuid" />
      <DeviceTemplatesPanel :uuid="props.uuid" />
      <DeviceLogsPanel :uuid="props.uuid" />
      <DeviceParameterListPanel :uuid="props.uuid" />
    </div>
  </div>
</template>

<style scoped>
.device-grid {
  display: flex;
  flex-direction: column;
  gap: 1.25rem;
}
</style>
