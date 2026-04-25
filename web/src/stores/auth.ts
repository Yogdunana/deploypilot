import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin, register as apiRegister } from '@/api/modules/auth'
import { getMe } from '@/api/modules/users'
import type { User } from '@/types/models'
import type { LoginRequest, RegisterRequest } from '@/types/api'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string>(localStorage.getItem('deploypilot_token') || '')

  const isLoggedIn = computed(() => !!token.value)
  const currentUser = computed(() => user.value)
  const userRole = computed(() => user.value?.role?.name || '')

  async function login(data: LoginRequest) {
    const res = await apiLogin(data)
    const { token: newToken, user: newUser } = res.data.data
    token.value = newToken
    user.value = newUser as unknown as User
    localStorage.setItem('deploypilot_token', newToken)
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
    localStorage.removeItem('deploypilot_token')
  }

  async function fetchMe() {
    try {
      const res = await getMe()
      user.value = res.data.data
    } catch {
      logout()
    }
  }

  return { user, token, isLoggedIn, currentUser, userRole, login, register, logout, fetchMe }
})
