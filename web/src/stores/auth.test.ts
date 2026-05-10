import { describe, it, expect, vi, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useAuthStore } from '@/stores/auth'

// Mock API 模块
vi.mock('@/api/modules/auth', () => ({
  login: vi.fn(),
  register: vi.fn(),
}))

vi.mock('@/api/modules/twofa', () => ({
  verify: vi.fn(),
}))

vi.mock('@/api/modules/users', () => ({
  getMe: vi.fn(),
}))

import { login as apiLogin, register as apiRegister } from '@/api/modules/auth'
import { verify as apiVerify2FA } from '@/api/modules/twofa'
import { getMe } from '@/api/modules/users'

const mockUser = {
  id: 1,
  tenant_id: 1,
  role_id: 1,
  username: 'testuser',
  email: 'test@example.com',
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z',
  role: { id: 1, name: 'admin', permissions: ['*'], created_at: '2026-01-01T00:00:00Z' },
}

describe('useAuthStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  describe('初始状态', () => {
    it('token 默认为空字符串', () => {
      const store = useAuthStore()
      expect(store.token).toBe('')
    })

    it('user 默认为 null', () => {
      const store = useAuthStore()
      expect(store.user).toBeNull()
    })

    it('isLoggedIn 默认为 false', () => {
      const store = useAuthStore()
      expect(store.isLoggedIn).toBe(false)
    })

    it('userRole 默认为空字符串', () => {
      const store = useAuthStore()
      expect(store.userRole).toBe('')
    })

    it('requires2FA 默认为 false', () => {
      const store = useAuthStore()
      expect(store.requires2FA).toBe(false)
    })

    it('从 localStorage 恢复 token', () => {
      localStorage.setItem('deploypilot_token', 'saved-token')
      setActivePinia(createPinia())
      const store = useAuthStore()
      expect(store.token).toBe('saved-token')
      expect(store.isLoggedIn).toBe(true)
    })
  })

  describe('login', () => {
    it('登录成功后设置 token 和 user', async () => {
      vi.mocked(apiLogin).mockResolvedValue({
        data: { data: { token: 'new-token', user: mockUser } },
      } as any)

      const store = useAuthStore()
      const result = await store.login({ username: 'testuser', password: 'password' })

      expect(result.requires_2fa).toBe(false)
      expect(store.token).toBe('new-token')
      expect(store.user).toEqual(mockUser)
      expect(store.isLoggedIn).toBe(true)
      expect(localStorage.getItem('deploypilot_token')).toBe('new-token')
    })

    it('需要 2FA 时设置 pending 状态', async () => {
      vi.mocked(apiLogin).mockResolvedValue({
        data: {
          data: {
            requires_2fa: true,
            two_fa_token: '2fa-token',
            user_id: 'user-123',
          },
        },
      } as any)

      const store = useAuthStore()
      const result = await store.login({ username: 'testuser', password: 'password' })

      expect(result.requires_2fa).toBe(true)
      expect(store.requires2FA).toBe(true)
      expect(store.pending2FAToken).toBe('2fa-token')
      expect(store.pending2FAUserId).toBe('user-123')
      expect(store.token).toBe('')
      expect(store.user).toBeNull()
    })
  })

  describe('verify2FACode', () => {
    it('验证 2FA 成功后设置 token 和 user', async () => {
      // 先设置 pending 状态
      const store = useAuthStore()
      store.pending2FAToken = '2fa-token'
      store.pending2FAUserId = 'user-123'

      vi.mocked(apiVerify2FA).mockResolvedValue({
        data: { data: { token: 'verified-token', user: mockUser } },
      } as any)

      await store.verify2FACode('123456')

      expect(store.token).toBe('verified-token')
      expect(store.user).toEqual(mockUser)
      expect(store.requires2FA).toBe(false)
      expect(store.pending2FAToken).toBeNull()
      expect(store.pending2FAUserId).toBeNull()
    })

    it('没有 pending token 时抛出错误', async () => {
      const store = useAuthStore()

      await expect(store.verify2FACode('123456')).rejects.toThrow('No pending 2FA token')
    })
  })

  describe('clear2FAPending', () => {
    it('清除 2FA pending 状态', () => {
      const store = useAuthStore()
      store.pending2FAToken = '2fa-token'
      store.pending2FAUserId = 'user-123'

      store.clear2FAPending()

      expect(store.pending2FAToken).toBeNull()
      expect(store.pending2FAUserId).toBeNull()
      expect(store.requires2FA).toBe(false)
    })
  })

  describe('logout', () => {
    it('清除所有状态', () => {
      const store = useAuthStore()
      store.token = 'some-token'
      store.user = mockUser
      store.pending2FAToken = '2fa-token'
      store.pending2FAUserId = 'user-123'
      localStorage.setItem('deploypilot_token', 'some-token')

      store.logout()

      expect(store.token).toBe('')
      expect(store.user).toBeNull()
      expect(store.isLoggedIn).toBe(false)
      expect(store.pending2FAToken).toBeNull()
      expect(store.pending2FAUserId).toBeNull()
      expect(localStorage.getItem('deploypilot_token')).toBeNull()
    })
  })

  describe('register', () => {
    it('注册成功后设置 token 和 user', async () => {
      vi.mocked(apiRegister).mockResolvedValue({
        data: { data: { token: 'reg-token', user: mockUser } },
      } as any)

      const store = useAuthStore()
      await store.register({
        username: 'newuser',
        email: 'new@example.com',
        password: 'password123',
      })

      expect(store.token).toBe('reg-token')
      expect(store.user).toEqual(mockUser)
      expect(store.isLoggedIn).toBe(true)
    })
  })

  describe('fetchMe', () => {
    it('成功获取用户信息', async () => {
      vi.mocked(getMe).mockResolvedValue({
        data: { data: mockUser },
      } as any)

      const store = useAuthStore()
      await store.fetchMe()

      expect(store.user).toEqual(mockUser)
    })

    it('401 错误时调用 logout', async () => {
      const err = new Error('Unauthorized')
      ;(err as any).response = { status: 401 }
      vi.mocked(getMe).mockRejectedValue(err)

      const store = useAuthStore()
      store.token = 'some-token'
      store.user = mockUser

      await store.fetchMe()

      expect(store.token).toBe('')
      expect(store.user).toBeNull()
    })

    it('非 401 错误时不调用 logout', async () => {
      const err = new Error('Network Error')
      ;(err as any).response = { status: 500 }
      vi.mocked(getMe).mockRejectedValue(err)

      const store = useAuthStore()
      store.token = 'some-token'
      store.user = mockUser

      await store.fetchMe()

      expect(store.token).toBe('some-token')
      expect(store.user).toEqual(mockUser)
    })
  })

  describe('computed', () => {
    it('isLoggedIn 根据 token 计算', () => {
      const store = useAuthStore()
      expect(store.isLoggedIn).toBe(false)

      store.token = 'has-token'
      expect(store.isLoggedIn).toBe(true)

      store.token = ''
      expect(store.isLoggedIn).toBe(false)
    })

    it('userRole 根据 user.role.name 计算', () => {
      const store = useAuthStore()
      expect(store.userRole).toBe('')

      store.user = mockUser
      expect(store.userRole).toBe('admin')

      store.user = { ...mockUser, role: { ...mockUser.role!, name: 'viewer' } }
      expect(store.userRole).toBe('viewer')
    })

    it('userRole 在 user 没有 role 时返回空字符串', () => {
      const store = useAuthStore()
      store.user = { ...mockUser, role: undefined }
      expect(store.userRole).toBe('')
    })

    it('currentUser 返回 user', () => {
      const store = useAuthStore()
      expect(store.currentUser).toBeNull()

      store.user = mockUser
      expect(store.currentUser).toEqual(mockUser)
    })
  })
})
