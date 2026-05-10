<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useToast } from '@/composables/useToast'
import { Plus, RefreshCw, RotateCcw, Trash2 } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Table from '@/components/ui/ResponsiveTable.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Switch from '@/components/ui/Switch.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as sslApi from '@/api/modules/ssl'
import type { SSLCertificate } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = useToast()
const { t } = useI18n()

const loading = ref(true)
const certificates = ref<SSLCertificate[]>([])

// 创建对话框
const createDialogOpen = ref(false)
const creating = ref(false)
const createForm = ref({
  domain: '',
  email: '',
  provider: 'cloudflare',
})

// 删除对话框
const deleteDialogOpen = ref(false)
const deleting = ref(false)
const selectedCert = ref<SSLCertificate | null>(null)

// 续期中
const renewingId = ref<number | null>(null)

// Provider 选项
const providerOptions = computed(() => [
  { label: 'Cloudflare', value: 'cloudflare' },
  { label: t('ssl.aliyun'), value: 'aliyun' },
  { label: t('ssl.tencent'), value: 'tencent' },
])

// 表格列
const columns = computed(() => [
  { key: 'domain', label: t('ssl.domain'), mobile: true },
  { key: 'provider', label: t('ssl.provider'), mobile: true },
  { key: 'status', label: t('ssl.status'), mobile: true },
  { key: 'expires_at', label: t('ssl.expiresAt'), mobile: true },
  { key: 'auto_renew', label: t('ssl.autoRenew') },
  { key: 'actions', label: t('ssl.actions'), width: '160px' },
])

// 证书状态样式
function getStatusVariant(status: string): 'success' | 'destructive' | 'warning' | 'default' | 'secondary' {
  const s = (status || '').toLowerCase()
  switch (s) {
    case 'active':
      return 'success'
    case 'expired':
    case 'failed':
      return 'destructive'
    case 'pending':
    case 'renewing':
      return 'warning'
    default:
      return 'secondary'
  }
}

function getStatusLabel(status: string): string {
  const s = (status || '').toLowerCase()
  const map: Record<string, string> = {
    active: t('ssl.active'),
    expired: t('ssl.expired'),
    pending: t('ssl.pending'),
    renewing: t('ssl.renewing'),
    failed: t('ssl.failed'),
  }
  return map[s] || s
}

// 计算过期天数
function getDaysUntilExpiry(expiresAt: string): number {
  if (!expiresAt) return 0
  const now = new Date()
  const expiry = new Date(expiresAt)
  const diff = expiry.getTime() - now.getTime()
  return Math.ceil(diff / (1000 * 60 * 60 * 24))
}

// 格式化过期时间显示
function formatExpiry(expiresAt: string): { text: string; urgent: boolean } {
  const days = getDaysUntilExpiry(expiresAt)
  if (days < 0) return { text: `${t('ssl.expiredDays', { days: Math.abs(days) })}`, urgent: true }
  if (days === 0) return { text: t('ssl.expiresToday'), urgent: true }
  if (days <= 15) return { text: t('ssl.expiresInDays', { days }), urgent: true }
  if (days <= 30) return { text: t('ssl.expiresInDays', { days }), urgent: false }
  return { text: t('ssl.expiresInDays', { days }), urgent: false }
}

// 获取证书列表
async function fetchCertificates() {
  loading.value = true
  try {
    const res = await sslApi.listCertificates()
    if (res.data.status === 'success') {
      certificates.value = res.data.data || []
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('ssl.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// 打开创建对话框
function openCreateDialog() {
  createForm.value = { domain: '', email: '', provider: 'cloudflare' }
  createDialogOpen.value = true
}

// 创建证书
async function handleCreate() {
  if (!createForm.value.domain.trim()) {
    toast(t('ssl.domainRequired'), 'destructive')
    return
  }
  if (!createForm.value.email.trim()) {
    toast(t('ssl.emailRequired'), 'destructive')
    return
  }

  creating.value = true
  try {
    await sslApi.requestCertificate({
      domain: createForm.value.domain.trim(),
      email: createForm.value.email.trim(),
      provider: createForm.value.provider,
      auto_renew: true,
    })
    toast(t('ssl.requestSubmitted'), 'success')
    createDialogOpen.value = false
    fetchCertificates()
  } catch (err: any) {
    toast(err.response?.data?.message || t('ssl.requestFailed'), 'destructive')
  } finally {
    creating.value = false
  }
}

// 续期
async function handleRenew(cert: SSLCertificate) {
  renewingId.value = cert.id
  try {
    await sslApi.renewCertificate(cert.id)
    toast(t('ssl.renewTriggered', { domain: cert.domain }), 'success')
    fetchCertificates()
  } catch (err: any) {
    toast(err.response?.data?.message || t('ssl.renewFailed'), 'destructive')
  } finally {
    renewingId.value = null
  }
}

// 打开删除对话框
function openDeleteDialog(cert: SSLCertificate) {
  selectedCert.value = cert
  deleteDialogOpen.value = true
}

// 确认删除
async function confirmDelete() {
  if (!selectedCert.value) return
  deleting.value = true
  try {
    await sslApi.deleteCertificate(selectedCert.value.id)
    toast(t('ssl.deleted'), 'success')
    deleteDialogOpen.value = false
    fetchCertificates()
  } catch (err: any) {
    toast(err.response?.data?.message || t('ssl.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
  }
}

// 刷新
async function handleRefresh() {
  await fetchCertificates()
}

onMounted(() => {
  fetchCertificates()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader :title="t('ssl.title')" :description="t('ssl.description')">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="handleRefresh">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          <span class="hidden sm:inline">{{ t('ssl.refresh') }}</span>
        </Button>
        <Button size="sm" @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          <span class="hidden sm:inline">{{ t('ssl.addCertificate') }}</span>
        </Button>
      </template>
    </PageHeader>

    <!-- 证书列表 -->
    <Table
      :columns="columns"
      :data="certificates"
      :loading="loading"
    >
      <template #cell-domain="{ row }">
        <span class="text-sm font-medium text-foreground">{{ row.domain }}</span>
      </template>
      <template #cell-provider="{ row }">
        <Badge variant="secondary">{{ row.provider || '-' }}</Badge>
      </template>
      <template #cell-status="{ row }">
        <Badge :variant="getStatusVariant(row.status)">
          {{ getStatusLabel(row.status) }}
        </Badge>
      </template>
      <template #cell-expires_at="{ row }">
        <span
          class="text-sm"
          :class="formatExpiry(row.expires_at).urgent ? 'text-destructive font-medium' : 'text-muted-foreground'"
        >
          {{ formatExpiry(row.expires_at).text }}
        </span>
      </template>
      <template #cell-auto_renew="{ row }">
        <Switch :model-value="row.auto_renew" disabled />
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            class="h-8 sm:h-7 text-xs min-w-[2.5rem]"
            :loading="renewingId === row.id"
            :disabled="row.status === 'expired'"
            @click="handleRenew(row as SSLCertificate)"
          >
            <template #icon><RotateCcw class="w-3.5 h-3.5" /></template>
            <span class="hidden sm:inline">{{ t('ssl.renew') }}</span>
          </Button>
          <Button
            variant="ghost"
            size="sm"
            class="h-8 sm:h-7 text-xs text-muted-foreground hover:text-destructive min-w-[2.5rem]"
            @click="openDeleteDialog(row as SSLCertificate)"
          >
            <template #icon><Trash2 class="w-3.5 h-3.5" /></template>
            <span class="hidden sm:inline">{{ t('ssl.delete') }}</span>
          </Button>
        </div>
      </template>
    </Table>

    <!-- 创建证书对话框 -->
    <Dialog
      v-model:open="createDialogOpen"
      :title="t('ssl.requestTitle')"
      :description="t('ssl.requestDesc')"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('ssl.domain') }}</label>
          <Input
            v-model="createForm.domain"
            :placeholder="t('ssl.domainPlaceholder')"
          />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('ssl.email') }}</label>
          <Input
            v-model="createForm.email"
            type="email"
            :placeholder="t('ssl.emailPlaceholder')"
          />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('ssl.provider') }}</label>
          <Select
            v-model="createForm.provider"
            :options="providerOptions"
            :placeholder="t('ssl.selectProvider')"
          />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :loading="creating" @click="handleCreate">{{ t('ssl.addCertificate') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- 删除确认对话框 -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      :title="t('ssl.deleteConfirm')"
      :description="t('ssl.deleteConfirmDesc', { domain: selectedCert?.domain || '' })"
      :confirm-text="t('ssl.confirmDelete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
