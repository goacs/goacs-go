import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/endpoints/auth.api'
import { authState } from '@/api/http'
import type { User } from '@/api/types/user'
import { connectSocket, disconnectSocket } from '@/sockets/socket'

const STORAGE_KEY = 'goacs.auth.token'
const STORAGE_USER_KEY = 'goacs.auth.user'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(null)
  const user = ref<User | null>(null)

  const isAuthenticated = computed(() => token.value !== null)

  function applySession(newToken: string, newUser: User) {
    token.value = newToken
    user.value = newUser
    authState.token = newToken
    localStorage.setItem(STORAGE_KEY, newToken)
    localStorage.setItem(STORAGE_USER_KEY, JSON.stringify(newUser))
    connectSocket(newToken)
  }

  function clearSession() {
    token.value = null
    user.value = null
    authState.token = null
    localStorage.removeItem(STORAGE_KEY)
    localStorage.removeItem(STORAGE_USER_KEY)
    disconnectSocket()
  }

  async function login(username: string, password: string) {
    const response = await authApi.login({ username, password })
    applySession(response.token, response.user)
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      clearSession()
    }
  }

  // Called once at app startup to rehydrate the session from localStorage,
  // since a full page reload loses all in-memory Pinia state.
  function restoreSession() {
    const storedToken = localStorage.getItem(STORAGE_KEY)
    const storedUser = localStorage.getItem(STORAGE_USER_KEY)

    if (!storedToken || !storedUser) return

    try {
      token.value = storedToken
      user.value = JSON.parse(storedUser) as User
      authState.token = storedToken
      connectSocket(storedToken)
    } catch {
      clearSession()
    }
  }

  return { token, user, isAuthenticated, login, logout, restoreSession, clearSession }
})
