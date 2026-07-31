import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import 'primeicons/primeicons.css'
import App from './App.vue'
import router from './router'
import { installAuthGuard } from './router/guards'
import { installPrimeVue } from './plugins/primevue'
import { setUnauthorizedHandler } from './api/http'
import { useAuthStore } from './stores/auth.store'

const app = createApp(App)

app.use(createPinia())
installPrimeVue(app)
app.use(router)
installAuthGuard(router)

const authStore = useAuthStore()
authStore.restoreSession()

setUnauthorizedHandler(() => {
  authStore.clearSession()
  router.push({ path: '/auth/login', query: { redirect: router.currentRoute.value.fullPath } })
})

app.mount('#app')
