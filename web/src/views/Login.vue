<script setup lang="ts">
import { ref } from 'vue'
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
const password = ref('')
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  error.value = ''

  if (!username.value.trim()) {
    error.value = t('login.usernameRequired')
    return
  }
  if (!password.value) {
    error.value = t('login.passwordRequired')
    return
  }

  loading.value = true
  try {
    await authStore.login({ username: username.value.trim(), password: password.value })
    router.push('/dashboard')
  } catch (err: any) {
    const msg = err?.response?.data?.message || err?.message || t('login.failed')
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
  <div class="min-h-screen flex items-center justify-center bg-background px-4 py-6 sm:p-4" style="padding-bottom: max(1.5rem, env(safe-area-inset-bottom, 1.5rem))">
    <div class="w-full max-w-sm">
      <!-- Logo & Title -->
      <div class="text-center mb-8">
        <img src="/icon.svg" alt="DeployPilot" class="h-12 mx-auto mb-4" />
        <h1 class="text-2xl font-bold text-foreground">DeployPilot</h1>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('login.title') }}</p>
      </div>

      <!-- Login Card -->
      <Card class="shadow-lg border-border/50">
        <form class="space-y-4" @submit.prevent="handleLogin">
          <!-- Username -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('login.username') }}</label>
            <Input
              v-model="username"
              :placeholder="t('login.usernamePlaceholder')"
              :error="!!error"
              @keydown="handleKeydown"
            />
          </div>

          <!-- Password -->
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('login.password') }}</label>
            <Input
              v-model="password"
              type="password"
              :placeholder="t('login.passwordPlaceholder')"
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
            {{ t('login.submit') }}
          </Button>

          <!-- Register Link -->
          <p class="text-center text-sm text-muted-foreground pt-2">
            {{ t('login.noAccount') }}
            <RouterLink to="/register" class="text-primary hover:underline font-medium">
              {{ t('login.signupNow') }}
            </RouterLink>
          </p>
        </form>
      </Card>
    </div>
  </div>
</template>
