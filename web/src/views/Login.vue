<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Button from '@/components/ui/Button.vue'
import { Rocket } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''

  if (!username.value.trim()) {
    error.value = '请输入用户名'
    return
  }
  if (!password.value) {
    error.value = '请输入密码'
    return
  }

  loading.value = true
  try {
    await authStore.login({ username: username.value.trim(), password: password.value })
    router.push('/dashboard')
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || '登录失败，请检查用户名和密码'
    error.value = msg
  } finally {
    loading.value = false
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    handleLogin()
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
        <p class="mt-1 text-sm text-muted-foreground">登录到 DeployPilot</p>
      </div>

      <!-- Login Card -->
      <Card class="shadow-lg border-border/50">
        <form class="space-y-4" @submit.prevent="handleLogin">
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

          <!-- Password -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">密码</label>
            <Input
              v-model="password"
              type="password"
              placeholder="请输入密码"
              :error="!!error"
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
            登录
          </Button>

          <!-- Register Link -->
          <p class="text-center text-sm text-muted-foreground pt-2">
            还没有账号？
            <RouterLink to="/register" class="text-primary hover:underline font-medium">
              立即注册
            </RouterLink>
          </p>
        </form>
      </Card>
    </div>
  </div>
</template>
