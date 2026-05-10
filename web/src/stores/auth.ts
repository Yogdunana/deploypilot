import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin, register as apiRegister } from '@/api/modules/auth'
import { verify as apiVerify2FA } from '@/api/modules/twofa'
import { getMe } from '@/api/modules/users'
import type { User } from '@/types/models'
import type { LoginRequest, RegisterRequest, LoginResponse2FA } from '@/types/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string>(localStorage.getItem('deploypilot_token') || '')
  const pending2FAToken = ref<string | null>(null)
  const pending2FAUserId = ref<string | null>(null)

  const isLoggedIn = computed(() => !!token.value)
  const requires2FA = computed(() => !!pending2FAToken.value)
  const currentUser = computed(() => user.value)
  const userRole = computed(() => user.value?.role?.name || '')

  async function login(data: LoginRequest) {
    const res = await apiLogin(data)
    const responseData = res.data.data
    if (responseData.requires_2fa) {
      const r2fa = responseData as unknown as LoginResponse2FA
      pending2FAToken.value = r2fa.two_fa_token
      pending2FAUserId.value = r2fa.user_id
      return { requires_2fa: true } as const
    }
    const { token: newToken, user: newUser } = responseData
    token.value = newToken
    user.value = newUser as unknown as User
    localStorage.setItem('deploypilot_token', newToken)
    return { requires_2fa: false } as const
  }

  async function verify2FACode(code: string) {
    if (!pending2FAToken.value) throw new Error('No pending 2FA token')
    const res = await apiVerify2FA({ two_fa_token: pending2FAToken.value, code })
    const { token: newToken, user: newUser } = res.data.data
    token.value = newToken
    user.value = newUser as unknown as User
    localStorage.setItem('deploypilot_token', newToken)
    pending2FAToken.value = null
    pending2FAUserId.value = null
  }

  function clear2FAPending() {
    pending2FAToken.value = null
    pending2FAUserId.value = null
  }

  async function register(data: RegisterRequest) {
    const res = await apiRegister(data)
    const { token: newToken, user: newUser } = res.data.data
    token.value = newToken
    user.value = newUser as unknown as User
    localStorage.setItem('deploypilot_token', newToken)
  }

  function logout() {
    token.value = ''
    user.value = null
    pending2FAToken.value = null
    pending2FAUserId.value = null
    localStorage.removeItem('deploypilot_token')
  }

  async function fetchMe() {
    try {
      const res = await getMe()
      user.value = res.data.data
    } catch (err: any) {
      if (err.response?.status === 401) {
        logout()
      }
    }
  }

  return {
    user, token, isLoggedIn, requires2FA, currentUser, userRole,
    pending2FAToken, pending2FAUserId,
    login, verify2FACode, clear2FAPending,
    register, logout, fetchMe
  }
})
