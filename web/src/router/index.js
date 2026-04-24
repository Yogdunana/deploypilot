import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from '../utils/auth'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/register',
    name: 'Register',
    component: () => import('../views/Register.vue'),
    meta: { requiresAuth: false },
  },
  {
    path: '/',
    component: () => import('../layout/MainLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      {
        path: '',
        redirect: '/dashboard',
      },
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { title: 'Dashboard' },
      },
      {
        path: 'apps',
        name: 'Apps',
        component: () => import('../views/Apps.vue'),
        meta: { title: 'Applications' },
      },
      {
        path: 'apps/:id',
        name: 'AppDetail',
        component: () => import('../views/AppDetail.vue'),
        meta: { title: 'App Detail' },
      },
      {
        path: 'servers',
        name: 'Servers',
        component: () => import('../views/Servers.vue'),
        meta: { title: 'Servers' },
      },
      {
        path: 'credentials',
        name: 'Credentials',
        component: () => import('../views/Credentials.vue'),
        meta: { title: 'Credentials' },
      },
      {
        path: 'dns',
        name: 'DNS',
        component: () => import('../views/DNS.vue'),
        meta: { title: 'DNS Records' },
      },
      {
        path: 'providers',
        name: 'Providers',
        component: () => import('../views/Providers.vue'),
        meta: { title: 'Providers' },
      },
      {
        path: 'notifications',
        name: 'Notifications',
        component: () => import('../views/Notifications.vue'),
        meta: { title: 'Notifications' },
      },
      {
        path: 'templates',
        name: 'Templates',
        component: () => import('../views/Templates.vue'),
        meta: { title: 'Templates' },
      },
      {
        path: 'deployments',
        name: 'Deployments',
        component: () => import('../views/Deployments.vue'),
        meta: { title: 'Deployments' },
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('../views/Users.vue'),
        meta: { title: 'Users' },
      },
      {
        path: 'system',
        name: 'System',
        component: () => import('../views/System.vue'),
        meta: { title: 'System' },
      },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  if (to.meta.requiresAuth !== false && !isAuthenticated()) {
    next('/login')
  } else if ((to.path === '/login' || to.path === '/register') && isAuthenticated()) {
    next('/dashboard')
  } else {
    next()
  }
})

export default router
