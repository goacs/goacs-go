import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth.store'

export function installAuthGuard(router: Router): void {
  router.beforeEach((to) => {
    const authStore = useAuthStore()

    if (to.meta.auth && !authStore.isAuthenticated) {
      return { path: '/auth/login', query: { redirect: to.fullPath } }
    }

    if (to.path === '/auth/login' && authStore.isAuthenticated) {
      return { path: '/dashboard' }
    }

    return true
  })
}
