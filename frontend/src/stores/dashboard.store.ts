import { defineStore } from 'pinia'
import { ref } from 'vue'
import { dashboardApi } from '@/api/endpoints/dashboard.api'
import type { DashboardData } from '@/api/types/dashboard'

export const useDashboardStore = defineStore('dashboard', () => {
  const data = ref<DashboardData | null>(null)
  const loading = ref(false)

  async function fetch() {
    loading.value = true
    try {
      data.value = await dashboardApi.get()
    } finally {
      loading.value = false
    }
  }

  return { data, loading, fetch }
})
