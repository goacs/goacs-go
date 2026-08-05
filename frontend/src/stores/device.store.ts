import { defineStore } from 'pinia'
import { ref } from 'vue'
import { deviceApi } from '@/api/endpoints/device.api'
import type { CPE } from '@/api/types/device'

export const useDeviceStore = defineStore('device', () => {
  const currentDevice = ref<CPE | null>(null)
  const loading = ref(false)

  async function fetchDevice(uuid: string) {
    loading.value = true
    try {
      currentDevice.value = await deviceApi.get(uuid)
    } finally {
      loading.value = false
    }
  }

  async function deleteDevice(uuid: string) {
    await deviceApi.delete(uuid)
  }

  async function provisionNow(uuid: string) {
    await deviceApi.provision(uuid)
  }

  async function lookupNow(uuid: string) {
    await deviceApi.lookup(uuid)
  }

  async function clearCache(uuid: string) {
    await deviceApi.clearCache(uuid)
  }

  return { currentDevice, loading, fetchDevice, deleteDevice, provisionNow, lookupNow, clearCache }
})
