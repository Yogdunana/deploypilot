<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Button from '@/components/ui/Button.vue'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

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

  if (score <= 2) return { level: 1, label: t('register.weak'), color: 'bg-destructive' }
  if (score <= 3) return { level: 2, label: t('register.medium'), color: 'bg-warning' }
  return { level: 3, label: t('register.strong'), color: 'bg-success' }
})

const strengthBarWidth = computed(() => {
  return `${(passwordStrength.value.level / 3) * 100}%`
})

async function handleRegister() {
  error.value = ''

  if (!username.value.trim()) {
    error.value = t('register.usernameRequired')
    return
  }
  if (!email.value.trim()) {
    error.value = t('register.emailRequired')
    return
  }
  if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.value)) {
    error.value = t('register.emailInvalid')
    return
  }
  if (!password.value) {
    error.value = t('register.passwordRequired')
    return
  }
  if (password.value.length < 6) {
    error.value = t('register.passwordTooShort')
    return
  }
  if (password.value !== confirmPassword.value) {
    error.value = t('register.passwordMismatch')
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
    const msg = err?.response?.data?.message || err?.message || t('register.failed')
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
  <div class="min-h-screen flex items-center justify-center bg-background px-4 py-6 sm:p-4" style="padding-bottom: max(1.5rem, env(safe-area-inset-bottom, 1.5rem))">
    <div class="w-full max-w-sm">
      <!-- Logo & Title -->
      <div class="text-center mb-8">
        <img src="/logo.png" alt="DeployPilot" class="h-12 mx-auto mb-4" />
        <h1 class="text-2xl font-bold text-foreground">DeployPilot</h1>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('register.title') }}</p>
      </div>

      <!-- Register Card -->
      <Card class="shadow-lg border-border/50">
        <form class="space-y-4" @submit.prevent="handleRegister">
          <!-- Username -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('register.username') }}</label>
            <Input
              v-model="username"
              :placeholder="t('register.usernamePlaceholder')"
              :error="!!error"
              @keydown="handleKeydown"
            />
          </div>

          <!-- Email -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('register.email') }}</label>
            <Input
              v-model="email"
              type="email"
              :placeholder="t('register.emailPlaceholder')"
              :error="!!error"
              @keydown="handleKeydown"
            />
          </div>

          <!-- Password -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('register.password') }}</label>
            <Input
              v-model="password"
              type="password"
              :placeholder="t('register.passwordPlaceholder')"
              :error="!!error"
              @keydown="handleKeydown"
            />
            <!-- Password Strength Indicator -->
            <div v-if="password" class="mt-2">
              <div class="flex items-center justify-between mb-1">
                <span class="text-xs text-muted-foreground">{{ t('register.passwordStrength') }}</span>
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
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('register.confirmPassword') }}</label>
            <Input
              v-model="confirmPassword"
              type="password"
              :placeholder="t('register.confirmPasswordPlaceholder')"
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
            {{ t('register.submit') }}
          </Button>

          <!-- Login Link -->
          <p class="text-center text-sm text-muted-foreground pt-2">
            {{ t('register.hasAccount') }}
            <RouterLink to="/login" class="text-primary hover:underline font-medium">
              {{ t('register.loginNow') }}
            </RouterLink>
          </p>
        </form>
      </Card>
    </div>
  </div>
</template>
