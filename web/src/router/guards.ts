import type { Router } from 'vue-router'

export function setupRouterGuards(router: Router) {
  router.beforeEach((to, _from, next) => {
    const token = localStorage.getItem('deploypilot_token')

    if (to.meta.requiresAuth && !token) {
      next({ name: 'Login', query: { redirect: to.fullPath } })
    } else if (to.meta.guest && token) {
      next({ name: 'Dashboard' })
    } else {
      next()
    }
  })
}
