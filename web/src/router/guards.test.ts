import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setupRouterGuards } from '@/router/guards'
import type { Router } from 'vue-router'

// 创建 mock 路由
function createMockRouter() {
  const router = {
    beforeEach: vi.fn(),
  } as unknown as Router
  return router
}

describe('setupRouterGuards', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('注册 beforeEach 守卫', () => {
    const router = createMockRouter()
    setupRouterGuards(router)
    expect(router.beforeEach).toHaveBeenCalledTimes(1)
  })

  describe('未认证用户', () => {
    it('访问 requiresAuth 路由时重定向到登录页', () => {
      const router = createMockRouter()
      setupRouterGuards(router)

      // 获取注册的守卫回调
      const guardFn = vi.mocked(router.beforeEach).mock.calls[0][0]

      const to = { meta: { requiresAuth: true }, fullPath: '/dashboard', name: 'Dashboard' }
      const from = {}
      const next = vi.fn()

      guardFn.call(router, to as any, from as any, next)

      expect(next).toHaveBeenCalledWith({
        name: 'Login',
        query: { redirect: '/dashboard' },
      })
    })

    it('访问非 requiresAuth 路由时正常放行', () => {
      const router = createMockRouter()
      setupRouterGuards(router)

      const guardFn = vi.mocked(router.beforeEach).mock.calls[0][0]

      const to = { meta: {}, fullPath: '/public', name: 'Public' }
      const from = {}
      const next = vi.fn()

      guardFn.call(router, to as any, from as any, next)

      expect(next).toHaveBeenCalledWith()
    })
  })

  describe('已认证用户', () => {
    beforeEach(() => {
      localStorage.setItem('deploypilot_token', 'valid-token')
    })

    it('访问 guest 路由时重定向到 Dashboard', () => {
      const router = createMockRouter()
      setupRouterGuards(router)

      const guardFn = vi.mocked(router.beforeEach).mock.calls[0][0]

      const to = { meta: { guest: true }, fullPath: '/login', name: 'Login' }
      const from = {}
      const next = vi.fn()

      guardFn.call(router, to as any, from as any, next)

      expect(next).toHaveBeenCalledWith({ name: 'Dashboard' })
    })

    it('访问普通路由时正常放行', () => {
      const router = createMockRouter()
      setupRouterGuards(router)

      const guardFn = vi.mocked(router.beforeEach).mock.calls[0][0]

      const to = { meta: {}, fullPath: '/dashboard', name: 'Dashboard' }
      const from = {}
      const next = vi.fn()

      guardFn.call(router, to as any, from as any, next)

      expect(next).toHaveBeenCalledWith()
    })

    it('访问 requiresAuth 路由时正常放行', () => {
      const router = createMockRouter()
      setupRouterGuards(router)

      const guardFn = vi.mocked(router.beforeEach).mock.calls[0][0]

      const to = { meta: { requiresAuth: true }, fullPath: '/settings', name: 'Settings' }
      const from = {}
      const next = vi.fn()

      guardFn.call(router, to as any, from as any, next)

      expect(next).toHaveBeenCalledWith()
    })
  })

  describe('边界情况', () => {
    it('路由 meta 为空对象时正常放行', () => {
      const router = createMockRouter()
      setupRouterGuards(router)

      const guardFn = vi.mocked(router.beforeEach).mock.calls[0][0]

      const to = { meta: {}, fullPath: '/', name: 'Home' }
      const from = {}
      const next = vi.fn()

      guardFn.call(router, to as any, from as any, next)

      expect(next).toHaveBeenCalledWith()
    })

    it('requiresAuth 优先于 guest 检查', () => {
      // 没有 token 的情况下，requiresAuth 优先触发
      const router = createMockRouter()
      setupRouterGuards(router)

      const guardFn = vi.mocked(router.beforeEach).mock.calls[0][0]

      const to = { meta: { requiresAuth: true, guest: true }, fullPath: '/page', name: 'Page' }
      const from = {}
      const next = vi.fn()

      guardFn.call(router, to as any, from as any, next)

      // 未认证，requiresAuth 优先，重定向到登录
      expect(next).toHaveBeenCalledWith({
        name: 'Login',
        query: { redirect: '/page' },
      })
    })
  })
})
