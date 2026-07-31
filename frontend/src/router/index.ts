import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', redirect: '/dashboard' },
    {
      path: '/auth',
      component: () => import('@/layouts/AuthLayout.vue'),
      children: [
        { path: 'login', name: 'login', component: () => import('@/views/auth/LoginView.vue') },
        { path: 'logout', name: 'logout', component: () => import('@/views/auth/LogoutView.vue') },
      ],
    },
    {
      path: '/',
      component: () => import('@/layouts/AppLayout.vue'),
      meta: { auth: true },
      children: [
        { path: 'dashboard', name: 'dashboard', component: () => import('@/views/dashboard/DashboardView.vue') },
        {
          path: 'devices',
          name: 'devices-list',
          component: () => import('@/views/devices/DeviceListView.vue'),
        },
        {
          path: 'devices/:uuid',
          name: 'devices-view',
          component: () => import('@/views/devices/DeviceView.vue'),
          props: true,
        },
        {
          path: 'devices/:uuid/cached',
          name: 'devices-cached-params',
          component: () => import('@/views/devices/DeviceCachedParametersView.vue'),
          props: true,
        },
        {
          path: 'configuration',
          name: 'configuration-list',
          component: () => import('@/views/configuration/ConfigurationListView.vue'),
        },
        {
          path: 'configuration/create',
          name: 'configuration-create',
          component: () => import('@/views/configuration/ConfigurationCreateView.vue'),
        },
        {
          path: 'configuration/:id',
          name: 'configuration-edit',
          component: () => import('@/views/configuration/ConfigurationEditView.vue'),
          props: true,
        },
        {
          path: 'templates',
          name: 'template-list',
          component: () => import('@/views/templates/TemplateListView.vue'),
        },
        {
          path: 'templates/:id',
          name: 'template-view',
          component: () => import('@/views/templates/TemplateView.vue'),
          props: true,
        },
        { path: 'files', name: 'file-list', component: () => import('@/views/files/FileListView.vue') },
        {
          path: 'settings',
          component: () => import('@/views/settings/SettingsView.vue'),
          children: [
            { path: '', name: 'settings-main', component: () => import('@/views/settings/BaseSettingsView.vue') },
            { path: 'users', name: 'settings-users', component: () => import('@/views/settings/UsersListView.vue') },
            { path: 'debug', name: 'settings-debug', component: () => import('@/views/settings/DebugView.vue') },
          ],
        },
      ],
    },
  ],
})

export default router
