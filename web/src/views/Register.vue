<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Button from '@/components/ui/Button.vue'
import { Rocket } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const email = ref('')
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

// Password strength
const passwordStrength = computed(() => {
  const pwd = password.value
  if (!pwd) return { level: 0, label: '', color: '' }

  let score = 0
  if (pwd.length >= 6) score++
  if (pwd.length >= 10) score++
  if (/[A-Z]/.test(pwd)) score++
  if (/[0-9]/.test(pwd)) score++
  if (/[^A-Za-z0-9]/.test(pwd)) score++

  if (score <= 2) return { level: 1, label: '弱', color: 'bg-destructive' }
  if (score <= 3) return { level: 2, label: '中', color: 'bg-warning' }
  return { level: 3, label: '强', color: 'bg-success' }
})

const strengthBarWidth = computed(() => {
  return `${(passwordStrength.value.level / 3) * 100}%`
})

async function handleRegister() {
  error.value = ''

  if (!username.value.trim()) {
    error.value = '请输入用户名'
    return
  }
  if (!email.value.trim()) {
    error.value = '请输入邮箱'
    return
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) {
    error.value = '请输入有效的邮箱地址'
    return
  }
  if (!password.value) {
    error.value = '请输入密码'
    return
  }
  if (password.value.length < 6) {
    error.value = '密码长度至少为 6 位'
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = '两次输入的密码不一致'
    return
  }

  loading.value = true
  try {
    await authStore.register({
      username: username.value.trim(),
      email: email.value.trim(),
      password: password.value,
    })
    router.push('/login')
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '注册失败，请稍后重试'
    error.value = msg
  } finally {
    loading.value = false
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    handleRegister()
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background p-4">
    <div class="w-full max-w-sm">
      <!-- Logo & Title -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-12 h-12 rounded-xl bg-primary/10 mb-4">
          <Rocket class="w-6 h-6 text-primary" />
        </div>
        <h1 class="text-2xl font-bold text-foreground">DeployPilot</h1>
        <p class="mt-1 text-sm text-muted-foreground">创建新账号</p>
      </div>

      <!-- Register Card -->
      <Card class="shadow-lg border-border/50">
        <form class="space-y-4" @submit.prevent="handleRegister">
          <!-- Username -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">用户名</label>
            <Input
              v-model="username"
              placeholder="请输入用户名"
              :error="!!error"
              @keydown="handleKeydown"
            />
          </div>

          <!-- Email -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">邮箱</label>
            <Input
              v-model="email"
              type="email"
              placeholder="请输入邮箱"
              :error="!!error"
              @keydown="handleKeydown"
            />
          </div>

          <!-- Password -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">密码</label>
            <Input
              v-model="password"
              type="password"
              placeholder="请输入密码（至少 6 位）"
              :error="!!error"
              @keydown="handleKeydown"
            />
            <!-- Password Strength Indicator -->
            <div v-if="password" class="mt-2">
              <div class="flex items-center justify-between mb-1">
                <span class="text-xs text-muted-foreground">密码强度</span>
                <span
                  class="text-xs font-medium"
                  :class="{
                    'text-destructive': passwordStrength.level === 1,
                    'text-warning': passwordStrength.level === 2,
                    'text-success': passwordStrength.level === 3,
                  }"
                >
                  {{ passwordStrength.label }}
                </span>
              </div>
              <div class="h-1.5 w-full rounded-full bg-accent overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-300"
                  :class="passwordStrength.color"
                  :style="{ width: strengthBarWidth }"
                />
              </div>
            </div>
          </div>

          <!-- Confirm Password -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">确认密码</label>
            <Input
              v-model="confirmPassword"
              type="password"
              placeholder="请再次输入密码"
              :error="!!error && confirmPassword !== password"
              @keydown="handleKeydown"
            />
          </div>

          <!-- Error Message -->
          <p v-if="error" class="text-sm text-destructive">
            {{ error }}
          </p>

          <!-- Submit Button -->
          <Button
            type="submit"
            class="w-full"
            size="lg"
            :loading="loading"
            :disabled="loading"
          >
            注册
          </Button>

          <!-- Login Link -->
          <p class="text-center text-sm text-muted-foreground pt-2">
            已有账号？
            <RouterLink to="/login" class="text-primary hover:underline font-medium">
              立即登录
            </RouterLink>
          </p>
        </form>
      </Card>
    </div>
  </div>
</template>
