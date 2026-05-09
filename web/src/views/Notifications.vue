<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { Plus, MoreHorizontal, Pencil, Trash2, Bell } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Select from '@/components/ui/Select.vue'
import Switch from '@/components/ui/Switch.vue'
import Table from '@/components/ui/ResponsiveTable.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as notificationsApi from '@/api/modules/notifications'
import type { Notification } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = useToast()
const { t } = useI18n()

// State
const notifications = ref<Notification[]>([])
const loading = ref(true)

// Dialog
const dialogOpen = ref(false)
const dialogTitle = ref(t('notifications.createTitle'))
const editingId = ref<number | null>(null)
const formName = ref('')
const formType = ref('')
const formConfig = ref('')
const formEnabled = ref(true)
const configError = ref('')
const submitting = ref(false)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingItem = ref<Notification | null>(null)
const deleting = ref(false)

// Type options
const typeOptions = computed(() => [
  { label: t('notifications.dingtalk'), value: 'dingtalk' },
  { label: t('notifications.feishu'), value: 'feishu' },
  { label: 'Telegram', value: 'telegram' },
  { label: t('notifications.email'), value: 'email' },
])

// Table columns
const columns = computed(() => [
  { key: 'name', label: t('notifications.name'), mobile: true },
  { key: 'type', label: t('notifications.type'), mobile: true },
  { key: 'enabled', label: t('notifications.enabledStatus'), mobile: true },
  { key: 'created_at', label: t('notifications.createdAt') },
  { key: 'actions', label: t('notifications.actions'), width: '80px' },
])

// Type badge mapping
function getTypeBadge(type: string) {
  const map: Record<string, { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning'; label: string }> = {
    dingtalk: { variant: 'default', label: t('notifications.dingtalk') },
    feishu: { variant: 'success', label: t('notifications.feishu') },
    telegram: { variant: 'warning', label: 'Telegram' },
    email: { variant: 'outline', label: t('notifications.email') },
  }
  return map[type] || { variant: 'secondary' as const, label: type }
}

// Validate JSON
function validateConfig(val: string) {
  if (!val.trim()) {
    configError.value = ''
    return true
  }
  try {
    JSON.parse(val)
    configError.value = ''
    return true
  } catch {
    configError.value = t('notifications.jsonInvalid')
    return false
  }
}

// Fetch notifications
async function fetchNotifications() {
  loading.value = true
  try {
    const res = await notificationsApi.list()
    if (res.data.status === 'success') {
      notifications.value = res.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('notifications.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Open create dialog
function openCreateDialog() {
  editingId.value = null
  dialogTitle.value = t('notifications.createTitle')
  formName.value = ''
  formType.value = ''
  formConfig.value = ''
  formEnabled.value = true
  configError.value = ''
  dialogOpen.value = true
}

// Open edit dialog
function openEditDialog(item: Notification) {
  editingId.value = item.id
  dialogTitle.value = t('notifications.editTitle')
  formName.value = item.name
  formType.value = item.type
  formConfig.value = JSON.stringify(item.config, null, 2)
  formEnabled.value = item.enabled
  configError.value = ''
  dialogOpen.value = true
}

// Submit form
async function handleSubmit() {
  if (!formName.value || !formType.value) {
    toast(t('notifications.nameRequired'), 'destructive')
    return
  }
  if (!validateConfig(formConfig.value)) {
    toast(t('notifications.correctJson'), 'destructive')
    return
  }
  submitting.value = true
  try {
    let config: Record<string, string> = {}
    if (formConfig.value.trim()) {
      config = JSON.parse(formConfig.value)
    }
    const data = { name: formName.value, type: formType.value, config, enabled: formEnabled.value }
    if (editingId.value) {
      await notificationsApi.update(editingId.value, data)
      toast(t('notifications.updated'), 'success')
    } else {
      await notificationsApi.create(data)
      toast(t('notifications.created'), 'success')
    }
    dialogOpen.value = false
    fetchNotifications()
  } catch (err: any) {
    toast(err.response?.data?.message || t('common.operationFailed'), 'destructive')
  } finally {
    submitting.value = false
  }
}

// Toggle enabled
async function handleToggleEnabled(item: Notification) {
  try {
    await notificationsApi.update(item.id, { enabled: !item.enabled })
    item.enabled = !item.enabled
    toast(t('notifications.toggled', { name: item.name, status: item.enabled ? t('notifications.enabled') : t('notifications.disabled') }), 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || t('notifications.toggleFailed'), 'destructive')
  }
}

// Delete
function openDeleteDialog(item: Notification) {
  deletingItem.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingItem.value) return
  deleting.value = true
  try {
    await notificationsApi.deleteNotification(deletingItem.value.id)
    toast(t('notifications.deleted', { name: deletingItem.value.name }), 'success')
    fetchNotifications()
  } catch (err: any) {
    toast(err.response?.data?.message || t('notifications.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
    deletingItem.value = null
  }
}

function getDropdownItems(item: Notification) {
  return [
    { label: t('notifications.edit'), icon: Pencil, action: () => openEditDialog(item) },
    { label: t('notifications.delete'), icon: Trash2, danger: true, action: () => openDeleteDialog(item) },
  ]
}

onMounted(fetchNotifications)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('notifications.title')">
      <template #actions>
        <Button @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          {{ t('notifications.addProvider') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-32" />
        <Skeleton class="h-4 w-20" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-8 ml-auto" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="notifications.length > 0"
      :columns="columns"
      :data="notifications"
    >
      <template #cell-name="{ row }">
        <span class="text-sm font-medium text-foreground">{{ row.name }}</span>
      </template>
      <template #cell-type="{ row }">
        <Badge :variant="getTypeBadge(row.type).variant">
          {{ getTypeBadge(row.type).label }}
        </Badge>
      </template>
      <template #cell-enabled="{ row }">
        <Switch :model-value="row.enabled" @update:model-value="handleToggleEnabled(row as Notification)" />
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-actions="{ row }">
        <DropdownMenu :items="getDropdownItems(row as Notification)">
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
      :icon="Bell"
      :title="t('notifications.noProviders')"
      :description="t('notifications.noProvidersDesc')"
      :action-text="t('notifications.addProvider')"
      @action="openCreateDialog"
    />

    <!-- Create/Edit Dialog -->
    <Dialog
      v-model:open="dialogOpen"
      :title="dialogTitle"
      :description="t('notifications.configDesc')"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('notifications.name') }}</label>
          <Input v-model="formName" :placeholder="t('notifications.namePlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('notifications.type') }}</label>
          <Select v-model="formType" :options="typeOptions" :placeholder="t('notifications.typePlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('notifications.config') }}</label>
          <Textarea
            v-model="formConfig"
            placeholder='{"webhook": "https://oapi.dingtalk.com/robot/send?access_token=xxx"}'
            :rows="5"
            class="font-mono text-xs"
            @input="validateConfig(formConfig)"
          />
          <p v-if="configError" class="text-xs text-destructive">{{ configError }}</p>
        </div>
        <div class="flex items-center gap-2">
          <Switch v-model="formEnabled" />
          <label class="text-sm text-foreground">{{ t('notifications.enabled') }}</label>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="dialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :loading="submitting" @click="handleSubmit">
            {{ editingId ? t('common.saveText') : t('common.createText') }}
          </Button>
        </div>
      </div>
    </Dialog>

    <!-- Delete AlertDialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      :title="t('notifications.deleteConfirm')"
      :description="t('notifications.deleteConfirmDesc', { name: deletingItem?.name || '' })"
      :confirm-text="t('notifications.delete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
