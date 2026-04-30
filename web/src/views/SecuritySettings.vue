<script setup lang="ts">
import { ref, computed, inject, onMounted } from 'vue'
import { ShieldCheck, ShieldOff, Copy } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Card from '@/components/ui/Card.vue'
import Dialog from '@/components/ui/Dialog.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as twofaApi from '@/api/modules/twofa'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
const authStore = useAuthStore()
const toast = inject<any>('toast')!
const twoFAEnabled = ref(false)
const loadingStatus = ref(true)
const setupStep = ref<'idle' | 'qr'>('idle')
const qrCodeUrl = ref('')
const secret = ref('')
const backupCodes = ref<string[]>([])
const confirmCode = ref('')
const setupLoading = ref(false)
const confirmLoading = ref(false)
const disableDialogOpen = ref(false)
const disableCode = ref('')
const disableLoading = ref(false)
const qrCodeImage = computed(() => {
  return `https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=${encodeURIComponent(qrCodeUrl.value)}`
})
async function fetch2FAStatus() {
  loadingStatus.value = true
  try {
    await authStore.fetchMe()
    twoFAEnabled.value = authStore.user?.totp_enabled ?? false
  } catch {
    toast(t('security.fetchStatusFailed'), 'destructive')
  } finally {
    loadingStatus.value = false
  }
}
async function startSetup() {
  setupLoading.value = true
  try {
    const res = await twofaApi.setup()
    const data = res.data.data
    secret.value = data.secret
    qrCodeUrl.value = data.qr_code_url
    backupCodes.value = data.backup_codes
    setupStep.value = 'qr'
  } catch {
    toast(t('security.setupFailed'), 'destructive')
  } finally {
    setupLoading.value = false
  }
}
async function confirmSetup() {
  if (!confirmCode.value.trim()) return
  confirmLoading.value = true
  try {
    await twofaApi.confirm({ code: confirmCode.value })
    twoFAEnabled.value = true
    setupStep.value = 'idle'
    confirmCode.value = ''
    toast(t('security.confirmSuccess'), 'success')
  } catch {
    toast(t('security.confirmFailed'), 'destructive')
  } finally {
    confirmLoading.value = false
  }
}
function openDisableDialog() {
  disableCode.value = ''
  disableDialogOpen.value = true
}
async function confirmDisable() {
  if (!disableCode.value.trim()) return
  disableLoading.value = true
  try {
    await twofaApi.disable({ code: disableCode.value })
    twoFAEnabled.value = false
    disableDialogOpen.value = false
    disableCode.value = ''
    toast(t('security.disableSuccess'), 'success')
  } catch {
    toast(t('security.disableFailed'), 'destructive')
  } finally {
    disableLoading.value = false
  }
}
async function copyBackupCodes() {
  try {
    await navigator.clipboard.writeText(backupCodes.value.join('\n'))
    toast(t('security.copied'), 'success')
  } catch {
    toast(t('security.copyFailed'), 'destructive')
  }
}
onMounted(() => {
  fetch2FAStatus()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <PageHeader :title="t('security.title')" :description="t('security.description')" />
    <!-- Status Card -->
    <Card>
      <div class="flex items-center gap-3">
        <template v-if="loadingStatus">
          <Skeleton class="h-10 w-10 rounded-full" />
          <Skeleton class="h-5 w-40" />
        </template>
        <template v-else>
          <ShieldCheck v-if="twoFAEnabled" class="h-10 w-10 text-green-500" />
          <ShieldOff v-else class="h-10 w-10 text-gray-400" />
          <div class="flex items-center gap-2">
            <span class="text-sm font-medium">
              {{ twoFAEnabled ? t('security.enabled') : t('security.disabled') }}
            </span>
            <Badge :variant="twoFAEnabled ? 'success' : 'secondary'">
              {{ twoFAEnabled ? t('security.on') : t('security.off') }}
            </Badge>
          </div>
        </template>
      </div>
    </Card>
    <!-- Setup Card -->
    <Card>
      <!-- idle + not enabled: show enable instructions -->
      <div v-if="setupStep === 'idle' && !twoFAEnabled" class="space-y-4">
        <p class="text-sm text-muted-foreground">{{ t('security.setupInstructions') }}</p>
        <Button :loading="setupLoading" @click="startSetup">
          {{ t('security.enable2FA') }}
        </Button>
      </div>
      <!-- qr step: show QR + manual key + verify + backup codes -->
      <div v-else-if="setupStep === 'qr'" class="space-y-6">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
          <!-- Left: QR image -->
          <div class="flex flex-col items-center gap-3">
            <img :src="qrCodeImage" :alt="t('security.qrCode')" class="rounded border" />
            <p class="text-xs text-muted-foreground">{{ t('security.scanQR') }}</p>
          </div>
          <!-- Right: manual key + code input + verify -->
          <div class="space-y-4">
            <div>
              <p class="text-sm font-medium mb-1">{{ t('security.manualKey') }}</p>
              <code class="block rounded bg-muted px-3 py-2 text-sm select-all break-all">
                {{ secret }}
              </code>
            </div>
            <div>
              <label class="text-sm font-medium mb-1 block">{{ t('security.verifyCode') }}</label>
              <Input
                v-model="confirmCode"
                :placeholder="t('security.enterCode')"
                @keyup.enter="confirmSetup"
              />
            </div>
            <Button :loading="confirmLoading" :disabled="!confirmCode.trim()" @click="confirmSetup">
              {{ t('security.verify') }}
            </Button>
          </div>
        </div>
        <!-- Backup codes -->
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <p class="text-sm font-medium">{{ t('security.backupCodes') }}</p>
            <Button variant="ghost" size="sm" @click="copyBackupCodes">
              <Copy class="h-4 w-4 mr-1" />
              {{ t('security.copy') }}
            </Button>
          </div>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
            <code
              v-for="(code, index) in backupCodes"
              :key="index"
              class="rounded bg-muted px-2 py-1 text-xs text-center select-all"
            >
              {{ code }}
            </code>
          </div>
          <p class="text-xs text-muted-foreground">{{ t('security.backupCodesWarning') }}</p>
        </div>
      </div>
      <!-- idle + enabled: show already enabled + disable button -->
      <div v-else-if="setupStep === 'idle' && twoFAEnabled" class="space-y-4">
        <p class="text-sm text-muted-foreground">{{ t('security.alreadyEnabled') }}</p>
        <Button variant="destructive" @click="openDisableDialog">
          {{ t('security.disable2FA') }}
        </Button>
      </div>
    </Card>
    <!-- Disable Dialog -->
    <Dialog :open="disableDialogOpen" @update:open="disableDialogOpen = $event">
      <template #title>{{ t('security.disableDialogTitle') }}</template>
      <template #description>{{ t('security.disableDialogDescription') }}</template>
      <div class="space-y-4">
        <Input
          v-model="disableCode"
          :placeholder="t('security.enterCode')"
          @keyup.enter="confirmDisable"
        />
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <Button variant="outline" @click="disableDialogOpen = false">
            {{ t('security.cancel') }}
          </Button>
          <Button
            variant="destructive"
            :loading="disableLoading"
            :disabled="!disableCode.trim()"
            @click="confirmDisable"
          >
            {{ t('security.confirmDisable') }}
          </Button>
        </div>
      </template>
    </Dialog>
  </div>
</template>
