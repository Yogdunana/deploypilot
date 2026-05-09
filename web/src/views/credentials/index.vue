<script setup lang="ts">
import { ref, computed, inject, onMounted } from 'vue'
import { Plus, KeyRound, RefreshCw } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Tabs from '@/components/ui/Tabs.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Pagination from '@/components/ui/Pagination.vue'
import * as credentialsApi from '@/api/modules/credentials'
import * as auditApi from '@/api/modules/audit'
import type { Credential } from '@/types/models'
import { useI18n } from 'vue-i18n'

import CredentialTable from './CredentialTable.vue'
import CredentialDialog from './CredentialDialog.vue'
import RotateDialog from './RotateDialog.vue'
import AuditLogTab from './AuditLogTab.vue'

const { toast } = inject<any>('toast')!
const { t } = useI18n()

// State
const credentials = ref<Credential[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// Dialog
const dialogOpen = ref(false)
const dialogTitle = ref(t('credentials.createTitle'))
const editingId = ref<number | null>(null)
const formName = ref('')
const formType = ref('')
const formValue = ref('')
const formExpiresInDays = ref(0)
const showValue = ref(false)
const submitting = ref(false)

// Rotate dialog
const rotateDialogOpen = ref(false)
const rotatingItem = ref<Credential | null>(null)
const rotateValue = ref('')
const rotateSubmitting = ref(false)

// Detail dialog
const detailDialogOpen = ref(false)
const detailItem = ref<Credential | null>(null)

// Detail audit
const detailActiveTab = ref('info')
const detailAuditLogs = ref<any[]>([])
const detailAuditLoading = ref(false)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingItem = ref<Credential | null>(null)
const deleting = ref(false)

// Type badge mapping
function getTypeBadge(type: string) {
  const map: Record<string, { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning'; label: string }> = {
    ssh: { variant: 'default', label: 'SSH' },
    api_key: { variant: 'success', label: 'API Key' },
    token: { variant: 'warning', label: 'Token' },
  }
  return map[type] || { variant: 'secondary' as const, label: type }
}

// Expiry status helpers
function getExpiryStatus(item: Credential): { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive'; label: string } {
  if (item.is_expired) {
    return { variant: 'destructive', label: t('credentials.expired') }
  }
  if (item.days_until_expiry !== undefined && item.days_until_expiry !== -1 && item.days_until_expiry <= 7) {
    return { variant: 'warning', label: t('credentials.expiringSoon') }
  }
  if (item.days_until_expiry === -1 || !item.expires_at) {
    return { variant: 'success', label: t('credentials.neverExpires') }
  }
  return { variant: 'success', label: t('credentials.valid') }
}

// Fetch credentials
async function fetchCredentials() {
  loading.value = true
  try {
    const res = await credentialsApi.list(undefined, { page: page.value, page_size: pageSize.value })
    if (res.data.status === 'success') {
      credentials.value = res.data.data
      total.value = res.data.pagination?.total || 0
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('credentials.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Open create dialog
function openCreateDialog() {
  editingId.value = null
  dialogTitle.value = t('credentials.createTitle')
  formName.value = ''
  formType.value = ''
  formValue.value = ''
  formExpiresInDays.value = 0
  showValue.value = false
  dialogOpen.value = true
}

// Open edit dialog
function openEditDialog(item: Credential) {
  editingId.value = item.id
  dialogTitle.value = t('credentials.editTitle')
  formName.value = item.name
  formType.value = item.type
  formValue.value = ''
  showValue.value = false
  dialogOpen.value = true
}

// Submit form
async function handleSubmit() {
  if (!formName.value || !formType.value) {
    toast(t('credentials.nameTypeRequired'), 'destructive')
    return
  }
  submitting.value = true
  try {
    const data: any = { name: formName.value, type: formType.value, value: formValue.value }
    if (!editingId.value && formExpiresInDays.value > 0) {
      data.expires_in_days = formExpiresInDays.value
    }
    if (editingId.value) {
      await credentialsApi.update(editingId.value, data)
      toast(t('credentials.updated'), 'success')
    } else {
      await credentialsApi.create(data)
      toast(t('credentials.created'), 'success')
    }
    dialogOpen.value = false
    fetchCredentials()
  } catch (err: any) {
    toast(err.response?.data?.message || t('common.operationFailed'), 'destructive')
  } finally {
    submitting.value = false
  }
}

// Open rotate dialog
function openRotateDialog(item: Credential) {
  rotatingItem.value = item
  rotateValue.value = ''
  rotateDialogOpen.value = true
}

// Submit rotate
async function handleRotate() {
  if (!rotatingItem.value || !rotateValue.value) {
    toast(t('credentials.nameTypeRequired'), 'destructive')
    return
  }
  rotateSubmitting.value = true
  try {
    await credentialsApi.rotate(rotatingItem.value.id, { value: rotateValue.value })
    toast(t('credentials.rotateSuccess', { name: rotatingItem.value.name }), 'success')
    rotateDialogOpen.value = false
    fetchCredentials()
  } catch (err: any) {
    toast(err.response?.data?.message || t('credentials.rotateFailed'), 'destructive')
  } finally {
    rotateSubmitting.value = false
  }
}

// Open detail dialog
function openDetailDialog(item: Credential) {
  detailItem.value = item
  detailActiveTab.value = 'info'
  detailDialogOpen.value = true
  fetchDetailAudit(item)
}

// Fetch detail audit logs
async function fetchDetailAudit(item: any) {
  detailAuditLoading.value = true
  try {
    const res = await auditApi.listLogs({ resource_type: 'credential', page: 1, page_size: 10 })
    if (res.data.status === 'success') {
      detailAuditLogs.value = res.data.data.filter(
        (log: any) => log.detail?.includes(item.name) || String(log.resource_id) === String(item.id)
      )
    }
  } catch { /* ignore */ }
  finally { detailAuditLoading.value = false }
}

// Delete
function openDeleteDialog(item: Credential) {
  deletingItem.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingItem.value) return
  deleting.value = true
  try {
    await credentialsApi.deleteCredential(deletingItem.value.id)
    toast(t('credentials.deleted', { name: deletingItem.value.name }), 'success')
    fetchCredentials()
  } catch (err: any) {
    toast(err.response?.data?.message || t('credentials.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
    deletingItem.value = null
  }
}

onMounted(fetchCredentials)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('credentials.title')">
      <template #actions>
        <Button @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          {{ t('credentials.addCredential') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-32" />
        <Skeleton class="h-4 w-20" />
        <Skeleton class="h-4 w-24" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-8 ml-auto" />
      </div>
    </div>

    <!-- Table -->
    <CredentialTable
      v-else-if="credentials.length > 0"
      :credentials="credentials"
      :loading="loading"
      @edit="openEditDialog"
      @rotate="openRotateDialog"
      @delete="openDeleteDialog"
      @detail="openDetailDialog"
    />

    <!-- Empty state -->
    <EmptyState
      v-else
      :icon="KeyRound"
      :title="t('credentials.noCredentials')"
      :description="t('credentials.noCredentialsDesc')"
      :action-text="t('credentials.addCredential')"
      @action="openCreateDialog"
    />

    <!-- Pagination -->
    <div v-if="total > pageSize" class="flex justify-end">
      <Pagination
        v-model:page="page"
        :page-size="pageSize"
        :total="total"
        @update:page="fetchCredentials"
      />
    </div>

    <!-- Create/Edit Dialog -->
    <CredentialDialog
      :open="dialogOpen"
      :title="dialogTitle"
      :editing-id="editingId"
      :form-name="formName"
      :form-type="formType"
      :form-value="formValue"
      :form-expires-in-days="formExpiresInDays"
      :show-value="showValue"
      :submitting="submitting"
      @update:open="dialogOpen = $event"
      @update:form-name="formName = $event"
      @update:form-type="formType = $event"
      @update:form-value="formValue = $event"
      @update:form-expires-in-days="formExpiresInDays = $event"
      @update:show-value="showValue = $event"
      @submit="handleSubmit"
    />

    <!-- Rotate Dialog -->
    <RotateDialog
      :open="rotateDialogOpen"
      :rotating-item="rotatingItem"
      :rotate-value="rotateValue"
      :rotate-submitting="rotateSubmitting"
      @update:open="rotateDialogOpen = $event"
      @update:rotate-value="rotateValue = $event"
      @submit="handleRotate"
    />

    <!-- Detail Dialog -->
    <Dialog
      v-model:open="detailDialogOpen"
      :title="detailItem?.name || ''"
      :description="t('credentials.configDesc')"
    >
      <div v-if="detailItem" class="space-y-4">
        <Tabs v-model="detailActiveTab" :tabs="[{ key: 'info', label: t('credentials.detailInfo') }, { key: 'audit', label: t('credentials.auditHistory') }]" />
        <div v-if="detailActiveTab === 'info'">
          <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div class="space-y-1">
              <p class="text-sm text-muted-foreground">{{ t('credentials.name') }}</p>
              <p class="text-sm font-medium">{{ detailItem.name }}</p>
            </div>
            <div class="space-y-1">
              <p class="text-sm text-muted-foreground">{{ t('credentials.type') }}</p>
              <Badge :variant="getTypeBadge(detailItem.type).variant">
                {{ getTypeBadge(detailItem.type).label }}
              </Badge>
            </div>
            <div class="space-y-1">
              <p class="text-sm text-muted-foreground">{{ t('credentials.expiryStatus') }}</p>
              <Badge :variant="getExpiryStatus(detailItem).variant">
                {{ getExpiryStatus(detailItem).label }}
              </Badge>
            </div>
            <div class="space-y-1">
              <p class="text-sm text-muted-foreground">{{ t('credentials.expiresAt') }}</p>
              <p class="text-sm font-medium">
                {{ detailItem.expires_at ? new Date(detailItem.expires_at).toLocaleString() : t('credentials.neverExpires') }}
              </p>
            </div>
            <div class="space-y-1">
              <p class="text-sm text-muted-foreground">{{ t('credentials.lastRotated') }}</p>
              <p class="text-sm font-medium">
                {{ detailItem.last_rotated ? new Date(detailItem.last_rotated).toLocaleString() : '-' }}
              </p>
            </div>
            <div class="space-y-1">
              <p class="text-sm text-muted-foreground">{{ t('credentials.rotationDays') }}</p>
              <p class="text-sm font-medium">
                {{ detailItem.rotation_days || 0 === 0 ? t('credentials.neverExpires') : detailItem.rotation_days + ' ' + t('common.days') }}
              </p>
            </div>
            <div class="space-y-1">
              <p class="text-sm text-muted-foreground">{{ t('credentials.createdAt') }}</p>
              <p class="text-sm font-medium">{{ new Date(detailItem.created_at).toLocaleString() }}</p>
            </div>
          </div>
          <div class="flex justify-end gap-2 pt-2">
            <Button variant="outline" @click="detailDialogOpen = false">{{ t('common.close') }}</Button>
            <Button @click="detailDialogOpen = false; openRotateDialog(detailItem)">
              <template #icon><RefreshCw class="w-4 h-4" /></template>
              {{ t('credentials.rotate') }}
            </Button>
          </div>
        </div>
        <div v-else-if="detailActiveTab === 'audit'">
          <AuditLogTab
            :audit-logs="detailAuditLogs"
            :loading="detailAuditLoading"
          />
        </div>
      </div>
    </Dialog>

    <!-- Delete AlertDialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      :title="t('credentials.deleteConfirm')"
      :description="t('credentials.deleteConfirmDesc', { name: deletingItem?.name || '' })"
      :confirm-text="t('credentials.delete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
