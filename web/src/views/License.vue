<script setup lang="ts">
import { ref, inject, onMounted, computed } from 'vue'
import { RefreshCw, Key, ShieldCheck, Lock, Check, Server, AppWindow, Users, Package, Ban } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Table from '@/components/ui/ResponsiveTable.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Input from '@/components/ui/Input.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Switch from '@/components/ui/Switch.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Progress from '@/components/ui/Progress.vue'
import * as licenseApi from '@/api/modules/license'
import type { LicenseInfo } from '@/types/models'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'

const { toast } = inject<any>('toast')!
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
const activateForm = ref({
  license_key: '',
  use_type: 'non-commercial',
  agree_terms: false,
})

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
const issueForm = ref({
  tenant: '',
  tier: 'community',
  use_type: 'non-commercial',
  duration_days: 365,
})

// 状态样式
function getStatusVariant(status: string): 'success' | 'destructive' | 'warning' | 'default' | 'secondary' {
  const s = (status || '').toLowerCase()
  switch (s) {
    case 'active':
      return 'success'
    case 'expired':
      return 'destructive'
    case 'revoked':
      return 'destructive'
    default:
      return 'secondary'
  }
}

function getStatusLabel(status: string): string {
  const s = (status || '').toLowerCase()
  const map: Record<string, string> = {
    active: t('license.active'),
    expired: t('license.expired'),
    revoked: t('license.revoked'),
  }
  return map[s] || s
}

// 套餐标签
function getTierLabel(tier: string): string {
  const t2 = (tier || '').toLowerCase()
  const map: Record<string, string> = {
    community: t('license.community'),
    team: t('license.team'),
    pro: t('license.pro'),
    enterprise: t('license.enterprise'),
  }
  return map[t2] || tier
}

// 用途类型标签
function getUseTypeLabel(useType: string): string {
  const u = (useType || '').toLowerCase()
  const map: Record<string, string> = {
    'non-commercial': t('license.nonCommercial'),
    commercial: t('license.commercial'),
  }
  return map[u] || useType
}

// 计算过期天数
function getDaysUntilExpiry(expiresAt: string | null): number {
  if (!expiresAt) return 0
  const now = new Date()
  const expiry = new Date(expiresAt)
  const diff = expiry.getTime() - now.getTime()
  return Math.ceil(diff / (1000 * 60 * 60 * 24))
}

// 格式化过期时间显示
function formatExpiry(expiresAt: string | null): { text: string; urgent: boolean } {
  if (!expiresAt) return { text: t('license.neverExpires'), urgent: false }
  const days = getDaysUntilExpiry(expiresAt)
  if (days < 0) return { text: t('license.expiredAgo', { days: Math.abs(days) }), urgent: true }
  if (days <= 7) return { text: t('license.daysRemaining', { days }), urgent: true }
  return { text: t('license.daysRemaining', { days }), urgent: false }
}

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
  activateForm.value = { license_key: '', use_type: 'non-commercial', agree_terms: false }
  activateDialogOpen.value = true
}

// 激活许可证
async function handleActivate() {
  activating.value = true
  try {
    const data: { license_key?: string; use_type?: string; agree_terms?: boolean } = {}
    if (activateForm.value.license_key.trim()) {
      data.license_key = activateForm.value.license_key.trim()
    } else {
      data.use_type = activateForm.value.use_type
      data.agree_terms = activateForm.value.agree_terms
    }
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
  issueForm.value = { tenant: '', tier: 'community', use_type: 'non-commercial', duration_days: 365 }
  issueDialogOpen.value = true
}

// 签发许可证
async function handleIssue() {
  if (!issueForm.value.tenant.trim()) {
    toast(t('common.nameRequired'), 'destructive')
    return
  }
  issuing.value = true
  try {
    await licenseApi.issueLicense(issueForm.value)
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

// 已签发许可证表格列
const issuedColumns = computed(() => [
  { key: 'tenant', label: t('license.tenant'), mobile: true },
  { key: 'tier', label: t('license.tier'), mobile: true },
  { key: 'use_type', label: t('license.useType'), mobile: true },
  { key: 'status', label: t('license.status'), mobile: true },
  { key: 'issued_at', label: t('license.issuedAt') },
  { key: 'actions', label: t('common.actions'), width: '120px' },
])

// 功能列表 (所有可能的功能)
const allFeatures = [
  'basic_deploy', 'ssl', 'dns', 'monitor', 'ci_cd',
  'cluster', 'registry', 'plugin', 'api_key', 'audit',
  'batch_ops', 'oauth2', 'webhook', 'template', 'backup',
]

onMounted(() => {
  fetchLicenseStatus()
  if (isAdmin.value) {
    fetchIssuedLicenses()
  }
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
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

    <!-- 加载状态 -->
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

    <!-- 无许可证 -->
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

    <!-- 许可证信息 -->
    <template v-else>
      <!-- 许可证状态卡片 -->
      <Card>
        <template #header>
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2">
              <ShieldCheck class="w-5 h-5 text-primary" />
              <h3 class="text-sm font-medium text-foreground">{{ t('license.status') }}</h3>
            </div>
            <div class="flex items-center gap-2">
              <Badge :variant="getStatusVariant(licenseInfo.status)">
                {{ getStatusLabel(licenseInfo.status) }}
              </Badge>
              <Badge variant="outline">{{ getTierLabel(licenseInfo.tier) }}</Badge>
              <Badge variant="secondary">{{ getUseTypeLabel(licenseInfo.use_type) }}</Badge>
            </div>
          </div>
        </template>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <div class="space-y-1">
            <p class="text-xs text-muted-foreground">{{ t('license.tier') }}</p>
            <p class="text-sm font-medium text-foreground">{{ getTierLabel(licenseInfo.tier) }}</p>
          </div>
          <div class="space-y-1">
            <p class="text-xs text-muted-foreground">{{ t('license.useType') }}</p>
            <p class="text-sm font-medium text-foreground">{{ getUseTypeLabel(licenseInfo.use_type) }}</p>
          </div>
          <div class="space-y-1">
            <p class="text-xs text-muted-foreground">{{ t('license.expiresAt') }}</p>
            <p
              class="text-sm font-medium"
              :class="formatExpiry(licenseInfo.expires_at).urgent ? 'text-destructive' : 'text-foreground'"
            >
              {{ formatExpiry(licenseInfo.expires_at).text }}
            </p>
          </div>
          <div class="space-y-1">
            <p class="text-xs text-muted-foreground">{{ t('license.machineId') }}</p>
            <p class="text-xs font-mono text-muted-foreground break-all">{{ licenseInfo.machine_id || '-' }}</p>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-4 pt-4 border-t border-border">
          <Button variant="outline" size="sm" @click="openDeactivateDialog">
            <template #icon><Ban class="w-4 h-4" /></template>
            <span class="hidden sm:inline">{{ t('license.deactivate') }}</span>
          </Button>
        </div>
      </Card>

      <!-- 资源使用 -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <Users class="w-5 h-5 text-primary" />
            <h3 class="text-sm font-medium text-foreground">{{ t('license.resourceUsage') }}</h3>
          </div>
        </template>
        <div class="space-y-4">
          <!-- 服务器 -->
          <div class="space-y-2">
            <div class="flex items-center justify-between text-sm">
              <div class="flex items-center gap-2">
                <Server class="w-4 h-4 text-muted-foreground" />
                <span class="text-foreground">{{ t('license.servers') }}</span>
              </div>
              <span class="text-muted-foreground">
                {{ licenseInfo.limits?.max_servers === -1 ? t('license.unlimited') : licenseInfo.limits?.max_servers }}
              </span>
            </div>
            <Progress
              v-if="licenseInfo.limits?.max_servers !== -1"
              :value="100"
              variant="default"
            />
          </div>
          <!-- 应用 -->
          <div class="space-y-2">
            <div class="flex items-center justify-between text-sm">
              <div class="flex items-center gap-2">
                <AppWindow class="w-4 h-4 text-muted-foreground" />
                <span class="text-foreground">{{ t('license.apps') }}</span>
              </div>
              <span class="text-muted-foreground">
                {{ licenseInfo.limits?.max_apps === -1 ? t('license.unlimited') : licenseInfo.limits?.max_apps }}
              </span>
            </div>
            <Progress
              v-if="licenseInfo.limits?.max_apps !== -1"
              :value="100"
              variant="default"
            />
          </div>
          <!-- 用户 -->
          <div class="space-y-2">
            <div class="flex items-center justify-between text-sm">
              <div class="flex items-center gap-2">
                <Users class="w-4 h-4 text-muted-foreground" />
                <span class="text-foreground">{{ t('license.users') }}</span>
              </div>
              <span class="text-muted-foreground">
                {{ licenseInfo.limits?.max_users === -1 ? t('license.unlimited') : licenseInfo.limits?.max_users }}
              </span>
            </div>
            <Progress
              v-if="licenseInfo.limits?.max_users !== -1"
              :value="100"
              variant="default"
            />
          </div>
        </div>
      </Card>

      <!-- 功能列表 -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <Package class="w-5 h-5 text-primary" />
            <h3 class="text-sm font-medium text-foreground">{{ t('license.features') }}</h3>
          </div>
        </template>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          <div
            v-for="feature in allFeatures"
            :key="feature"
            class="flex items-center gap-2 text-sm"
          >
            <Check v-if="licenseInfo.features?.includes(feature)" class="w-4 h-4 text-success shrink-0" />
            <Lock v-else class="w-4 h-4 text-muted-foreground shrink-0" />
            <span :class="licenseInfo.features?.includes(feature) ? 'text-foreground' : 'text-muted-foreground'">
              {{ feature }}
            </span>
          </div>
        </div>
      </Card>

      <!-- 增购功能 -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <Package class="w-5 h-5 text-primary" />
            <h3 class="text-sm font-medium text-foreground">{{ t('license.addons') }}</h3>
          </div>
        </template>
        <div v-if="licenseInfo.addons && licenseInfo.addons.length > 0" class="space-y-3">
          <div
            v-for="(addon, index) in licenseInfo.addons"
            :key="index"
            class="flex items-center justify-between py-2 border-b border-border last:border-0"
          >
            <div class="flex items-center gap-2">
              <Badge variant="secondary">{{ addon.key }}</Badge>
              <span class="text-sm text-foreground">x{{ addon.amount }}</span>
            </div>
            <span class="text-xs text-muted-foreground">{{ addon.expires_at || t('license.neverExpires') }}</span>
          </div>
        </div>
        <p v-else class="text-sm text-muted-foreground text-center py-4">{{ t('license.noAddons') }}</p>
      </Card>

      <!-- 管理员: 已签发许可证 -->
      <Card v-if="isAdmin">
        <template #header>
          <div class="flex items-center gap-2">
            <ShieldCheck class="w-5 h-5 text-primary" />
            <h3 class="text-sm font-medium text-foreground">{{ t('license.issuedLicenses') }}</h3>
          </div>
        </template>
        <Table
          :columns="issuedColumns"
          :data="issuedLicenses"
          :loading="issuedLoading"
        >
          <template #cell-tenant="{ row }">
            <span class="text-sm font-medium text-foreground">{{ row.tenant || row.tenant_name || '-' }}</span>
          </template>
          <template #cell-tier="{ row }">
            <Badge variant="outline">{{ getTierLabel(row.tier) }}</Badge>
          </template>
          <template #cell-use_type="{ row }">
            <Badge variant="secondary">{{ getUseTypeLabel(row.use_type) }}</Badge>
          </template>
          <template #cell-status="{ row }">
            <Badge :variant="getStatusVariant(row.status)">
              {{ getStatusLabel(row.status) }}
            </Badge>
          </template>
          <template #cell-issued_at="{ row }">
            <span class="text-sm text-muted-foreground">{{ row.issued_at || '-' }}</span>
          </template>
          <template #cell-actions="{ row }">
            <Button
              variant="ghost"
              size="sm"
              class="h-8 sm:h-7 text-xs text-muted-foreground hover:text-destructive min-w-[2.5rem]"
              :disabled="row.status === 'revoked'"
              @click="openRevokeDialog(row)"
            >
              <template #icon><Ban class="w-3.5 h-3.5" /></template>
              <span class="hidden sm:inline">{{ t('license.revoke') }}</span>
            </Button>
          </template>
        </Table>
      </Card>
    </template>

    <!-- 激活许可证对话框 -->
    <Dialog
      v-model:open="activateDialogOpen"
      :title="t('license.activateTitle')"
      :description="t('license.activateDesc')"
    >
      <div class="space-y-4">
        <!-- 许可证密钥输入 -->
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('license.licenseKey') }}</label>
          <Input
            v-model="activateForm.license_key"
            :placeholder="t('license.licenseKeyPlaceholder')"
          />
        </div>

        <!-- 分隔线 -->
        <div class="relative">
          <div class="absolute inset-0 flex items-center">
            <span class="w-full border-t border-border" />
          </div>
          <div class="relative flex justify-center text-xs">
            <span class="bg-card px-2 text-muted-foreground">{{ t('license.orCommunity') }}</span>
          </div>
        </div>

        <!-- Community 许可证 -->
        <div class="rounded-lg border border-border p-4 space-y-3">
          <h4 class="text-sm font-medium text-foreground">{{ t('license.communityTitle') }}</h4>
          <p class="text-xs text-muted-foreground">{{ t('license.communityDesc') }}</p>

          <!-- 法律声明 -->
          <div class="rounded-md bg-accent/50 p-3 space-y-2">
            <h5 class="text-xs font-medium text-foreground">{{ t('license.legalNotice') }}</h5>
            <p class="text-xs text-muted-foreground leading-relaxed">{{ t('license.legalText') }}</p>
          </div>

          <label class="flex items-start gap-2 cursor-pointer">
            <Switch v-model="activateForm.agree_terms" class="mt-0.5" />
            <span class="text-xs text-foreground">{{ t('license.agreeTerms') }}</span>
          </label>
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="activateDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button
            :loading="activating"
            :disabled="!activateForm.license_key.trim() && !activateForm.agree_terms"
            @click="handleActivate"
          >
            <template #icon><Key class="w-4 h-4" /></template>
            {{ activating ? t('license.activating') : t('license.activate') }}
          </Button>
        </div>
      </div>
    </Dialog>

    <!-- 停用确认对话框 -->
    <AlertDialog
      v-model:open="deactivateDialogOpen"
      :title="t('license.deactivateTitle')"
      :description="t('license.deactivateDesc')"
      :confirm-text="t('common.confirm')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDeactivate"
    />

    <!-- 撤销确认对话框 -->
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

    <!-- 签发许可证对话框 -->
    <Dialog
      v-model:open="issueDialogOpen"
      :title="t('license.issueLicense')"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('license.tenant') }}</label>
          <Input
            v-model="issueForm.tenant"
            :placeholder="t('common.namePlaceholder')"
          />
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-2">
            <label class="text-sm font-medium text-foreground">{{ t('license.tier') }}</label>
            <Input
              v-model="issueForm.tier"
              placeholder="community"
            />
          </div>
          <div class="space-y-2">
            <label class="text-sm font-medium text-foreground">{{ t('license.useType') }}</label>
            <Input
              v-model="issueForm.use_type"
              placeholder="non-commercial"
            />
          </div>
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('common.days', { count: '' }) }}</label>
          <Input
            v-model.number="issueForm.duration_days"
            type="number"
            placeholder="365"
          />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="issueDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :loading="issuing" @click="handleIssue">
            {{ t('license.issueLicense') }}
          </Button>
        </div>
      </div>
    </Dialog>
  </div>
</template>
