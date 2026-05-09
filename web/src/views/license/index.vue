<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useToast } from '@/composables/useToast'
import { RefreshCw, Key, ShieldCheck } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Dialog from '@/components/ui/Dialog.vue'
import Textarea from '@/components/ui/Textarea.vue'
import * as licenseApi from '@/api/modules/license'
import type { LicenseInfo } from '@/types/models'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

import LicenseInfoCard from './LicenseInfoCard.vue'
import ActivateDialog from './ActivateDialog.vue'
import IssueDialog from './IssueDialog.vue'
import IssuedLicensesTable from './IssuedLicensesTable.vue'

const { toast } = useToast()
const { t } = useI18n()
const authStore = useAuthStore()

const loading = ref(true)
const licenseInfo = ref<LicenseInfo | null>(null)
const issuedLicenses = ref<any[]>([])
const issuedLoading = ref(false)

// 是否为管理员 (owner/admin)
const isAdmin = computed(() => {
  const role = authStore.userRole?.toLowerCase()
  return role === 'owner' || role === 'admin'
})

// 激活对话框
const activateDialogOpen = ref(false)
const activating = ref(false)

// 停用对话框
const deactivateDialogOpen = ref(false)
const deactivating = ref(false)

// 撤销对话框
const revokeDialogOpen = ref(false)
const revoking = ref(false)
const revokeForm = ref({
  id: '',
  reason: '',
})

// 签发对话框
const issueDialogOpen = ref(false)
const issuing = ref(false)

// 获取许可证状态
async function fetchLicenseStatus() {
  loading.value = true
  try {
    const res = await licenseApi.getLicenseStatus()
    if (res.data.status === 'success') {
      licenseInfo.value = res.data.data || null
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('license.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// 获取已签发许可证列表 (管理员)
async function fetchIssuedLicenses() {
  issuedLoading.value = true
  try {
    const res = await licenseApi.listLicenses()
    if (res.data.status === 'success') {
      issuedLicenses.value = res.data.data || []
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('license.fetchFailed'), 'destructive')
  } finally {
    issuedLoading.value = false
  }
}

// 打开激活对话框
function openActivateDialog() {
  activateDialogOpen.value = true
}

// 激活许可证
async function handleActivate(data: { license_key?: string; use_type?: string; agree_terms?: boolean }) {
  activating.value = true
  try {
    await licenseApi.activateLicense(data)
    toast(t('license.activated'), 'success')
    activateDialogOpen.value = false
    fetchLicenseStatus()
  } catch (err: any) {
    toast(err.response?.data?.message || t('license.activateFailed'), 'destructive')
  } finally {
    activating.value = false
  }
}

// 打开停用对话框
function openDeactivateDialog() {
  deactivateDialogOpen.value = true
}

// 确认停用
async function confirmDeactivate() {
  deactivating.value = true
  try {
    await licenseApi.deactivateLicense()
    toast(t('license.deactivated'), 'success')
    deactivateDialogOpen.value = false
    fetchLicenseStatus()
  } catch (err: any) {
    toast(err.response?.data?.message || t('license.activateFailed'), 'destructive')
  } finally {
    deactivating.value = false
  }
}

// 打开撤销对话框
function openRevokeDialog(lic: any) {
  revokeForm.value = { id: lic.id || lic.license_key || '', reason: '' }
  revokeDialogOpen.value = true
}

// 确认撤销
async function confirmRevoke() {
  if (!revokeForm.value.reason.trim()) {
    toast(t('license.revokeReason'), 'destructive')
    return
  }
  revoking.value = true
  try {
    await licenseApi.revokeLicense(revokeForm.value.id, revokeForm.value.reason.trim())
    toast(t('license.revokeSuccess'), 'success')
    revokeDialogOpen.value = false
    fetchIssuedLicenses()
  } catch (err: any) {
    toast(err.response?.data?.message || t('license.activateFailed'), 'destructive')
  } finally {
    revoking.value = false
  }
}

// 打开签发对话框
function openIssueDialog() {
  issueDialogOpen.value = true
}

// 签发许可证
async function handleIssue(data: { tenant: string; tier: string; use_type: string; duration_days: number }) {
  if (!data.tenant.trim()) {
    toast(t('common.nameRequired'), 'destructive')
    return
  }
  issuing.value = true
  try {
    await licenseApi.issueLicense(data)
    toast(t('license.activated'), 'success')
    issueDialogOpen.value = false
    fetchIssuedLicenses()
  } catch (err: any) {
    toast(err.response?.data?.message || t('license.activateFailed'), 'destructive')
  } finally {
    issuing.value = false
  }
}

// 刷新
async function handleRefresh() {
  await fetchLicenseStatus()
  if (isAdmin.value) {
    await fetchIssuedLicenses()
  }
  toast(t('common.dataRefreshed'), 'success')
}

onMounted(() => {
  fetchLicenseStatus()
  if (isAdmin.value) {
    fetchIssuedLicenses()
  }
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Page header -->
    <PageHeader :title="t('license.title')" :description="t('license.description')">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="handleRefresh">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          <span class="hidden sm:inline">{{ t('common.refresh') }}</span>
        </Button>
        <Button v-if="!licenseInfo" size="sm" @click="openActivateDialog">
          <template #icon><Key class="w-4 h-4" /></template>
          <span class="hidden sm:inline">{{ t('license.activate') }}</span>
        </Button>
        <Button v-if="isAdmin" size="sm" @click="openIssueDialog">
          <template #icon><ShieldCheck class="w-4 h-4" /></template>
          <span class="hidden sm:inline">{{ t('license.issueLicense') }}</span>
        </Button>
      </template>
    </PageHeader>

    <!-- Loading state -->
    <div v-if="loading" class="space-y-4">
      <Card>
        <template #header>
          <Skeleton class="h-5 w-40" />
        </template>
        <div class="space-y-3">
          <Skeleton class="h-4 w-full" />
          <Skeleton class="h-4 w-3/4" />
          <Skeleton class="h-4 w-1/2" />
        </div>
      </Card>
      <Card>
        <template #header>
          <Skeleton class="h-5 w-32" />
        </template>
        <div class="space-y-3">
          <Skeleton class="h-4 w-full" />
          <Skeleton class="h-4 w-2/3" />
        </div>
      </Card>
    </div>

    <!-- No license -->
    <Card v-else-if="!licenseInfo">
      <template #header>
        <div class="flex items-center gap-2">
          <ShieldCheck class="w-5 h-5 text-muted-foreground" />
          <h3 class="text-sm font-medium text-foreground">{{ t('license.noLicense') }}</h3>
        </div>
      </template>
      <div class="flex flex-col items-center justify-center py-8 text-center">
        <Key class="w-12 h-12 text-muted-foreground mb-4" />
        <p class="text-sm text-muted-foreground mb-4">{{ t('license.activateDesc') }}</p>
        <Button @click="openActivateDialog">
          <template #icon><Key class="w-4 h-4" /></template>
          {{ t('license.activate') }}
        </Button>
      </div>
    </Card>

    <!-- License info -->
    <template v-else>
      <LicenseInfoCard
        :license-info="licenseInfo"
        @deactivate="openDeactivateDialog"
      />

      <!-- Admin: Issued licenses -->
      <Card v-if="isAdmin">
        <template #header>
          <div class="flex items-center gap-2">
            <ShieldCheck class="w-5 h-5 text-primary" />
            <h3 class="text-sm font-medium text-foreground">{{ t('license.issuedLicenses') }}</h3>
          </div>
        </template>
        <IssuedLicensesTable
          :issued-licenses="issuedLicenses"
          :loading="issuedLoading"
          @revoke="openRevokeDialog"
        />
      </Card>
    </template>

    <!-- Activate license dialog -->
    <ActivateDialog
      v-model:open="activateDialogOpen"
      :activating="activating"
      @activate="handleActivate"
    />

    <!-- Deactivate confirmation dialog -->
    <AlertDialog
      v-model:open="deactivateDialogOpen"
      :title="t('license.deactivateTitle')"
      :description="t('license.deactivateDesc')"
      :confirm-text="t('common.confirm')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDeactivate"
    />

    <!-- Revoke confirmation dialog -->
    <Dialog
      v-model:open="revokeDialogOpen"
      :title="t('license.revokeTitle')"
      :description="t('license.revokeDesc')"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('license.revokeReason') }}</label>
          <Textarea
            v-model="revokeForm.reason"
            :placeholder="t('license.revokeReason')"
            :rows="3"
          />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="revokeDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button
            variant="destructive"
            :loading="revoking"
            :disabled="!revokeForm.reason.trim()"
            @click="confirmRevoke"
          >
            {{ t('license.revoke') }}
          </Button>
        </div>
      </div>
    </Dialog>

    <!-- Issue license dialog -->
    <IssueDialog
      v-model:open="issueDialogOpen"
      :issuing="issuing"
      @issue="handleIssue"
    />
  </div>
</template>
