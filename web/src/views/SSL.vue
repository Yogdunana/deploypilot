<script setup lang="ts">
import { ref, inject, onMounted, computed } from 'vue'
import { Plus, RefreshCw, RotateCcw, Trash2 } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Table from '@/components/ui/Table.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Switch from '@/components/ui/Switch.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as sslApi from '@/api/modules/ssl'
import type { SSLCertificate } from '@/types/models'

const { toast } = inject<any>('toast')!

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
const providerOptions = [
  { label: 'Cloudflare', value: 'cloudflare' },
  { label: '阿里云', value: 'aliyun' },
  { label: '腾讯云', value: 'tencent' },
]

// 表格列
const columns = [
  { key: 'domain', label: '域名' },
  { key: 'provider', label: 'Provider' },
  { key: 'status', label: '状态' },
  { key: 'expires_at', label: '过期时间' },
  { key: 'auto_renew', label: '自动续期' },
  { key: 'actions', label: '操作', width: '160px' },
]

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
    active: '有效',
    expired: '已过期',
    pending: '申请中',
    renewing: '续期中',
    failed: '失败',
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
  if (days < 0) return { text: `已过期 ${Math.abs(days)} 天`, urgent: true }
  if (days === 0) return { text: '今天过期', urgent: true }
  if (days <= 15) return { text: `还有 ${days} 天过期`, urgent: true }
  if (days <= 30) return { text: `还有 ${days} 天过期`, urgent: false }
  return { text: `还有 ${days} 天过期`, urgent: false }
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
    toast(err.response?.data?.message || '获取证书列表失败', 'destructive')
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
    toast('请输入域名', 'destructive')
    return
  }
  if (!createForm.value.email.trim()) {
    toast('请输入邮箱', 'destructive')
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
    toast('证书申请已提交', 'success')
    createDialogOpen.value = false
    fetchCertificates()
  } catch (err: any) {
    toast(err.response?.data?.message || '申请证书失败', 'destructive')
  } finally {
    creating.value = false
  }
}

// 续期
async function handleRenew(cert: SSLCertificate) {
  renewingId.value = cert.id
  try {
    await sslApi.renewCertificate(cert.id)
    toast(`证书「${cert.domain}」续期已触发`, 'success')
    fetchCertificates()
  } catch (err: any) {
    toast(err.response?.data?.message || '续期失败', 'destructive')
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
    toast('证书已删除', 'success')
    deleteDialogOpen.value = false
    fetchCertificates()
  } catch (err: any) {
    toast(err.response?.data?.message || '删除证书失败', 'destructive')
  } finally {
    deleting.value = false
  }
}

// 刷新
async function handleRefresh() {
  await fetchCertificates()
  toast('数据已刷新', 'success')
}

onMounted(() => {
  fetchCertificates()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader title="SSL 证书" description="管理域名 SSL 证书的申请、续期和删除">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="handleRefresh">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          刷新
        </Button>
        <Button size="sm" @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          申请证书
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
            class="h-7 text-xs"
            :loading="renewingId === row.id"
            :disabled="row.status === 'expired'"
            @click="handleRenew(row as SSLCertificate)"
          >
            <template #icon><RotateCcw class="w-3.5 h-3.5" /></template>
            续期
          </Button>
          <Button
            variant="ghost"
            size="sm"
            class="h-7 text-xs text-muted-foreground hover:text-destructive"
            @click="openDeleteDialog(row as SSLCertificate)"
          >
            <template #icon><Trash2 class="w-3.5 h-3.5" /></template>
            删除
          </Button>
        </div>
      </template>
    </Table>

    <!-- 创建证书对话框 -->
    <Dialog
      v-model:open="createDialogOpen"
      title="申请 SSL 证书"
      description="填写域名和邮箱信息申请新的 SSL 证书"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">域名</label>
          <Input
            v-model="createForm.domain"
            placeholder="例如: example.com"
          />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">邮箱</label>
          <Input
            v-model="createForm.email"
            type="email"
            placeholder="例如: admin@example.com"
          />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">Provider</label>
          <Select
            v-model="createForm.provider"
            :options="providerOptions"
            placeholder="选择证书提供商"
          />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="createDialogOpen = false">取消</Button>
          <Button :loading="creating" @click="handleCreate">申请证书</Button>
        </div>
      </div>
    </Dialog>

    <!-- 删除确认对话框 -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      title="删除证书"
      :description="`确定要删除域名「${selectedCert?.domain}」的 SSL 证书吗？此操作不可撤销。`"
      confirm-text="确认删除"
      cancel-text="取消"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
