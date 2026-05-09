<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { Plus, MoreHorizontal, Trash2, Copy, Key } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Table from '@/components/ui/ResponsiveTable.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as apikeysApi from '@/api/modules/apikeys'
import type { ApiKey } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = useToast()
const { t } = useI18n()

// State
const apiKeys = ref<ApiKey[]>([])
const loading = ref(true)

// Create dialog
const createDialogOpen = ref(false)
const formName = ref('')
const formExpiresInDays = ref(0)
const submitting = ref(false)

// Created key dialog
const createdKeyDialogOpen = ref(false)
const createdRawKey = ref('')
const createdKeyName = ref('')

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingItem = ref<ApiKey | null>(null)
const deleting = ref(false)

// Table columns
const columns = computed(() => [
  { key: 'name', label: t('apiKeys.name'), mobile: true },
  { key: 'key_prefix', label: t('apiKeys.keyPrefix'), mobile: true },
  { key: 'scopes', label: t('apiKeys.scopes'), mobile: true },
  { key: 'expires_at', label: t('apiKeys.expiresAt'), mobile: true },
  { key: 'created_at', label: t('apiKeys.createdAt') },
  { key: 'actions', label: t('apiKeys.actions'), width: '80px' },
])

// Expiry status helper
function getExpiryStatus(item: ApiKey): { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive'; label: string } {
  if (!item.expires_at) {
    return { variant: 'secondary', label: t('apiKeys.neverExpires') }
  }
  const now = new Date()
  const expiresAt = new Date(item.expires_at)
  if (now > expiresAt) {
    return { variant: 'destructive', label: t('apiKeys.expired') }
  }
  const daysLeft = Math.ceil((expiresAt.getTime() - now.getTime()) / (1000 * 60 * 60 * 24))
  if (daysLeft <= 7) {
    return { variant: 'warning', label: t('apiKeys.expiringSoon') }
  }
  return { variant: 'success', label: t('apiKeys.valid') }
}

// Normalize scopes to array
function normalizeScopes(scopes: string[] | string): string[] {
  if (Array.isArray(scopes)) return scopes
  if (!scopes) return []
  return scopes.split(',').map((s) => s.trim()).filter(Boolean)
}

// Fetch keys
async function fetchKeys() {
  loading.value = true
  try {
    const res = await apikeysApi.list()
    if (res.data.status === 'success') {
      apiKeys.value = res.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('apiKeys.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Open create dialog
function openCreateDialog() {
  formName.value = ''
  formExpiresInDays.value = 0
  createDialogOpen.value = true
}

// Submit create
async function handleCreate() {
  if (!formName.value) {
    toast(t('apiKeys.nameRequired'), 'destructive')
    return
  }
  submitting.value = true
  try {
    const data: any = { name: formName.value }
    if (formExpiresInDays.value > 0) {
      data.expires_in_days = formExpiresInDays.value
    }
    const res = await apikeysApi.create(data)
    if (res.data.status === 'success') {
      createdRawKey.value = res.data.data.key
      createdKeyName.value = res.data.data.name
      createDialogOpen.value = false
      createdKeyDialogOpen.value = true
      fetchKeys()
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('apiKeys.createFailed'), 'destructive')
  } finally {
    submitting.value = false
  }
}

// Copy to clipboard
async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast(t('apiKeys.copied'), 'success')
  } catch {
    toast(t('apiKeys.copyFailed'), 'destructive')
  }
}

// Delete
function openDeleteDialog(item: ApiKey) {
  deletingItem.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingItem.value) return
  deleting.value = true
  try {
    await apikeysApi.remove(deletingItem.value.id)
    toast(t('apiKeys.deleted', { name: deletingItem.value.name }), 'success')
    fetchKeys()
  } catch (err: any) {
    toast(err.response?.data?.message || t('apiKeys.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
    deletingItem.value = null
  }
}

// Dropdown items
function getDropdownItems(item: ApiKey) {
  return [
    { label: t('apiKeys.copyPrefix'), icon: Copy, action: () => copyToClipboard(item.key_prefix) },
    { label: t('apiKeys.delete'), icon: Trash2, danger: true, action: () => openDeleteDialog(item) },
  ]
}

onMounted(fetchKeys)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('apiKeys.title')">
      <template #actions>
        <Button @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          {{ t('apiKeys.create') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-32" />
        <Skeleton class="h-4 w-24" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-24" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-8 ml-auto" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="apiKeys.length > 0"
      :columns="columns"
      :data="apiKeys"
    >
      <template #cell-name="{ row }">
        <span class="text-sm font-medium text-foreground">{{ row.name }}</span>
      </template>
      <template #cell-key_prefix="{ row }">
        <code class="text-xs font-mono bg-muted px-1.5 py-0.5 rounded text-foreground">{{ row.key_prefix }}...</code>
      </template>
      <template #cell-scopes="{ row }">
        <div class="flex flex-wrap gap-1">
          <Badge
            v-for="scope in normalizeScopes(row.scopes)"
            :key="scope"
            variant="outline"
          >
            {{ scope }}
          </Badge>
        </div>
      </template>
      <template #cell-expires_at="{ row }">
        <Badge :variant="getExpiryStatus(row as ApiKey).variant">
          {{ getExpiryStatus(row as ApiKey).label }}
        </Badge>
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-actions="{ row }">
        <DropdownMenu :items="getDropdownItems(row as ApiKey)">
          <template #trigger>
            <Button variant="ghost" size="icon">
              <MoreHorizontal class="w-4 h-4" />
            </Button>
          </template>
        </DropdownMenu>
      </template>
    </Table>

    <!-- Empty state -->
    <EmptyState
      v-else
      :icon="Key"
      :title="t('apiKeys.noKeys')"
      :description="t('apiKeys.noKeysDesc')"
      :action-text="t('apiKeys.create')"
      @action="openCreateDialog"
    />

    <!-- Create Dialog -->
    <Dialog
      v-model:open="createDialogOpen"
      :title="t('apiKeys.createTitle')"
      :description="t('apiKeys.createDesc')"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('apiKeys.name') }}</label>
          <Input v-model="formName" :placeholder="t('apiKeys.namePlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('apiKeys.expiresInDays') }}</label>
          <Input
            v-model.number="formExpiresInDays"
            type="number"
            :min="0"
            :placeholder="t('apiKeys.expiresInDaysPlaceholder')"
          />
          <p class="text-xs text-muted-foreground">{{ t('apiKeys.expiresInDaysHint') }}</p>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="createDialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :loading="submitting" @click="handleCreate">
            {{ t('common.createText') }}
          </Button>
        </div>
      </div>
    </Dialog>

    <!-- Created Key Dialog -->
    <Dialog
      v-model:open="createdKeyDialogOpen"
      :title="t('apiKeys.keyCreatedTitle')"
      :description="t('apiKeys.keyCreatedDesc')"
    >
      <div class="space-y-4">
        <div class="rounded-lg bg-destructive/10 border border-destructive/20 p-3 text-sm text-destructive">
          {{ t('apiKeys.keyWarning') }}
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('apiKeys.keyName') }}</label>
          <p class="text-sm text-muted-foreground">{{ createdKeyName }}</p>
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('apiKeys.rawKey') }}</label>
          <div class="flex items-center gap-2">
            <code class="flex-1 text-xs font-mono bg-muted px-3 py-2 rounded break-all block text-foreground">{{ createdRawKey }}</code>
            <Button variant="outline" size="icon" @click="copyToClipboard(createdRawKey)">
              <template #icon><Copy class="w-4 h-4" /></template>
            </Button>
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button @click="createdKeyDialogOpen = false">{{ t('common.close') }}</Button>
        </div>
      </div>
    </Dialog>

    <!-- Delete AlertDialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      :title="t('apiKeys.deleteConfirm')"
      :description="t('apiKeys.deleteConfirmDesc', { name: deletingItem?.name || '' })"
      :confirm-text="t('apiKeys.delete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
