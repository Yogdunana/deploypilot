<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Button from '@/components/ui/Button.vue'
import { ShieldCheck } from 'lucide-vue-next'

const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()

const username = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const twoFACode = ref('')
const twoFALoading = ref(false)

const show2FA = computed(() => authStore.requires2FA)

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
    const result = await authStore.login({ username: username.value.trim(), password: password.value })
    if (!result.requires_2fa) {
      router.push('/')
    }
  } catch (err: any) {
    error.value = err?.response?.data?.message || err?.message || t('login.failed')
  } finally {
    loading.value = false
  }
}

async function handle2FAVerify() {
  error.value = ''
  if (!twoFACode.value) {
    error.value = t('login.twoFACodeRequired')
    return
  }
  twoFALoading.value = true
  try {
    await authStore.verify2FACode(twoFACode.value)
    router.push('/')
  } catch (err: any) {
    if (err?.response?.status === 401) {
      authStore.clear2FAPending()
      error.value = t('login.failed')
    } else {
      error.value = err?.response?.data?.message || t('login.twoFAFailed')
    }
  } finally {
    twoFALoading.value = false
  }
}

function handleBackToLogin() {
  authStore.clear2FAPending()
  error.value = ''
  twoFACode.value = ''
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') handleLogin()
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background px-4 py-6 sm:p-4" style="padding-bottom: max(1.5rem, env(safe-area-inset-bottom, 1.5rem))">
    <div class="w-full max-w-sm">
      <div class="text-center mb-8">
        <img src="/icon.svg" alt="DeployPilot" class="h-12 mx-auto mb-4" />
        <h1 class="text-2xl font-bold text-foreground">DeployPilot</h1>
        <p class="mt-1 text-sm text-muted-foreground">{{ t('login.title') }}</p>
      </div>

      <Card class="shadow-lg border-border/50">
        <form v-if="!show2FA" class="space-y-4" @submit.prevent="handleLogin">
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('login.username') }}</label>
            <Input v-model="username" :placeholder="t('login.usernamePlaceholder')" :error="!!error" @keydown="handleKeydown" />
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('login.password') }}</label>
            <Input v-model="password" type="password" :placeholder="t('login.passwordPlaceholder')" :error="!!error" @keydown="handleKeydown" />
          </div>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
          <Button type="submit" class="w-full" size="lg" :loading="loading" :disabled="loading">{{ t('login.submit') }}</Button>
        </form>

        <form v-else class="space-y-4" @submit.prevent="handle2FAVerify">
          <div class="flex items-center gap-3 mb-2">
            <ShieldCheck class="w-8 h-8 text-primary" />
            <div>
              <h2 class="text-lg font-semibold text-foreground">{{ t('login.twoFATitle') }}</h2>
              <p class="text-sm text-muted-foreground">{{ t('login.twoFADescription') }}</p>
            </div>
          </div>
          <div>
            <label class="block text-sm font-medium text-foreground mb-1.5">{{ t('login.twoFACode') }}</label>
            <Input v-model="twoFACode" :placeholder="t('login.twoFACodePlaceholder')" :error="!!error" maxlength="6" inputmode="numeric" autocomplete="one-time-code" />
          </div>
          <p v-if="error" class="text-sm text-destructive">{{ error }}</p>
          <Button type="submit" class="w-full" size="lg" :loading="twoFALoading" :disabled="twoFALoading">{{ t('login.twoFAVerify') }}</Button>
          <Button variant="ghost" class="w-full" type="button" @click="handleBackToLogin">{{ t('login.backToLogin') }}</Button>
        </form>
      </Card>
    </div>
  </div>
</template>
