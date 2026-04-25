import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/Login.vue'),
      meta: { guest: true },
    },
    {
      path: '/register',
      name: 'Register',
      component: () => import('@/views/Register.vue'),
      meta: { guest: true },
    },
    {
      path: '/',
      component: () => import('@/layout/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: () => import('@/views/Dashboard.vue'),
        },
        {
          path: 'apps',
          name: 'Apps',
          component: () => import('@/views/Apps.vue'),
        },
        {
          path: 'apps/:id',
          name: 'AppDetail',
          component: () => import('@/views/AppDetail.vue'),
          props: true,
        },
        {
          path: 'apps/:id/logs',
          name: 'AppLogs',
          component: () => import('@/views/AppLogs.vue'),
          props: true,
          meta: { title: '应用日志' },
        },
        {
          path: 'apps/:id/env',
          name: 'AppEnv',
          component: () => import('@/views/AppEnv.vue'),
          props: true,
          meta: { title: '环境变量' },
        },
        {
          path: 'apps/:id/backups',
          name: 'AppBackups',
          component: () => import('@/views/AppBackups.vue'),
          props: true,
          meta: { title: '备份管理' },
        },
        {
          path: 'servers',
          name: 'Servers',
          component: () => import('@/views/Servers.vue'),
        },
        {
          path: 'servers/:id',
          name: 'ServerDetail',
          component: () => import('@/views/ServerDetail.vue'),
          props: true,
        },
        {
          path: 'servers/:id/terminal',
          name: 'ServerTerminal',
          component: () => import('@/views/ServerTerminal.vue'),
          props: true,
          meta: { title: '服务器终端' },
        },
        {
          path: 'servers/:id/environment',
          name: 'ServerEnvironment',
          component: () => import('@/views/ServerEnvironment.vue'),
          props: true,
          meta: { title: '环境检测' },
        },
        {
          path: 'deployments',
          name: 'Deployments',
          component: () => import('@/views/Deployments.vue'),
        },
        {
          path: 'credentials',
          name: 'Credentials',
          component: () => import('@/views/Credentials.vue'),
        },
        {
          path: 'dns',
          name: 'DNS',
          component: () => import('@/views/DNS.vue'),
        },
        {
          path: 'ssl',
          name: 'SSL',
          component: () => import('@/views/SSL.vue'),
        },
        {
          path: 'providers',
          name: 'Providers',
          component: () => import('@/views/Providers.vue'),
        },
        {
          path: 'cicd',
          name: 'CICD',
          component: () => import('@/views/CICD.vue'),
        },
        {
          path: 'monitor',
          name: 'Monitor',
          component: () => import('@/views/Monitor.vue'),
        },
        {
          path: 'monitor/containers',
          name: 'MonitorContainers',
          component: () => import('@/views/MonitorContainers.vue'),
          meta: { title: '容器监控' },
        },
        {
          path: 'monitor/alerts',
          name: 'MonitorAlerts',
          component: () => import('@/views/MonitorAlerts.vue'),
          meta: { title: '活跃告警' },
        },
        {
          path: 'monitor/alert-rules',
          name: 'MonitorAlertRules',
          component: () => import('@/views/MonitorAlertRules.vue'),
          meta: { title: '告警规则' },
        },
        {
          path: 'notifications',
          name: 'Notifications',
          component: () => import('@/views/Notifications.vue'),
        },
        {
          path: 'templates',
          name: 'Templates',
          component: () => import('@/views/Templates.vue'),
        },
        {
          path: 'users',
          name: 'Users',
          component: () => import('@/views/Users.vue'),
        },
        {
          path: 'audit',
          name: 'Audit',
          component: () => import('@/views/Audit.vue'),
        },
        {
          path: 'system',
          name: 'System',
          component: () => import('@/views/System.vue'),
        },
      ],
    },
  ],
})

export default router
