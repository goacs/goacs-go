import { defineStore } from 'pinia'
import { ref } from 'vue'
import { configurationApi } from '@/api/endpoints/configuration.api'
import type { Provision, ProvisionSimulateParam } from '@/api/types/configuration'

export const useConfigurationStore = defineStore('configuration', () => {
  const provisions = ref<Provision[]>([])
  const loading = ref(false)

  const simulatorOpen = ref(false)
  const simEvent = ref('')
  const simRequest = ref('')
  const simRoot = ref('')
  const simParams = ref<ProvisionSimulateParam[]>([])
  const simOnlyMatches = ref(false)

  async function load() {
    loading.value = true
    try {
      const res = await configurationApi.list({ page: 1, per_page: 10000 })
      provisions.value = res.data
    } finally {
      loading.value = false
    }
  }

  async function reorder(newOrder: Provision[]) {
    const previous = provisions.value
    provisions.value = newOrder
    try {
      await configurationApi.reorder(newOrder.map((p) => p.id))
      newOrder.forEach((p, i) => {
        p.priority = i + 1
      })
    } catch (e) {
      provisions.value = previous
      throw e
    }
  }

  async function toggleEnabled(provision: Provision) {
    const next = !provision.enabled
    provision.enabled = next
    try {
      await configurationApi.updateEnabled(provision.id, next)
    } catch (e) {
      provision.enabled = !next
      throw e
    }
  }

  async function clone(provision: Provision) {
    await configurationApi.clone(provision.id)
    await load()
  }

  async function remove(provision: Provision) {
    await configurationApi.delete(provision.id)
    provisions.value = provisions.value.filter((p) => p.id !== provision.id)
  }

  return {
    provisions,
    loading,
    simulatorOpen,
    simEvent,
    simRequest,
    simRoot,
    simParams,
    simOnlyMatches,
    load,
    reorder,
    toggleEnabled,
    clone,
    remove,
  }
})
