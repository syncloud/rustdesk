import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'server', component: () => import('./views/Server.vue') },
    { path: '/device', name: 'device', component: () => import('./views/DeviceLogin.vue') },
  ],
})

export default router
