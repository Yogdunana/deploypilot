import type { Router } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

export function setupRouterGuards(router: Router) {
  router.beforeEach(async (to, _from, next) => {
    const authStore = useAuthStore()
    const token = localStorage.getItem('deploypilot_token')

    // 未认证用户访问需要认证的路由
    if (to.meta.requiresAuth && !token) {
      return next({ name: 'Login', query: { redirect: to.fullPath } })
    }

    // 已认证用户访问访客页面
    if (to.meta.guest && token) {
      return next({ name: 'Dashboard' })
    }

    // 角色权限检查
    if (to.meta.roles && token) {
      // 确保用户信息已加载
      if (!authStore.user) {
        try {
          await authStore.fetchMe()
        } catch {
          localStorage.removeItem('deploypilot_token')
          return next({ name: 'Login' })
        }
      }

      const userRole = authStore.userRole?.toLowerCase() || ''
      const requiredRoles = to.meta.roles as string[]

      // 修复：即使 userRole 为空也要检查权限
      if (!requiredRoles.includes(userRole)) {
        return next({ name: 'Forbidden' })
      }
    }

    next()
  })
}
