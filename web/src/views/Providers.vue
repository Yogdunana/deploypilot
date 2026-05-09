<script setup lang="ts">
import { ref, computed, inject, onMounted, watch } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, Server } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Select from '@/components/ui/Select.vue'
import Switch from '@/components/ui/Switch.vue'
import Table from '@/components/ui/Table.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import Tabs from '@/components/ui/Tabs.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as providersApi from '@/api/modules/providers'
import type { Provider } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = inject<any>('toast')!
const { t } = useI18n()

// State
const providers = ref<Provider[]>([])
const loading = ref(true)
const activeTab = ref('')

// Dialog
const dialogOpen = ref(false)
const dialogTitle = ref(t('providers.createTitle'))
const editingId = ref<number | null>(null)
const formName = ref('')
const formType = ref('')
const formConfig = ref('')
const formEnabled = ref(true)
const configError = ref('')
const submitting = ref(false)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingItem = ref<Provider | null>(null)
const deleting = ref(false)

// Type options
const typeOptions = [
  { label: 'Docker', value: 'docker' },
  { label: 'SSH', value: 'ssh' },
  { label: '1Panel', value: '1panel' },
]

// Tab items
const tabItems = computed(() => [
  { key: '', label: t('providers.all') },
  { key: 'docker', label: 'Docker' },
  { key: 'ssh', label: 'SSH' },
  { key: '1panel', label: '1Panel' },
])

// Table columns
const columns = computed(() => [
  { key: 'name', label: t('providers.name') },
  { key: 'type', label: t('providers.type') },
  { key: 'enabled', label: t('providers.enabledStatus') },
  { key: 'created_at', label: t('providers.createdAt') },
  { key: 'actions', label: t('providers.actions'), width: '80px' },
])

// Type badge mapping
function getTypeBadge(type: string) {
  const map: Record<string, { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning'; label: string }> = {
    docker: { variant: 'success', label: 'Docker' },
    ssh: { variant: 'default', label: 'SSH' },
    '1panel': { variant: 'warning', label: '1Panel' },
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
    configError.value = t('providers.jsonInvalid')
    return false
  }
}

// Fetch providers
async function fetchProviders() {
  loading.value = true
  try {
    const res = await providersApi.list(activeTab.value || undefined)
    if (res.data.status === 'success') {
      providers.value = res.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('providers.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Watch tab change
watch(activeTab, () => {
  fetchProviders()
})

// Open create dialog
function openCreateDialog() {
  editingId.value = null
  dialogTitle.value = t('providers.createTitle')
  formName.value = ''
  formType.value = ''
  formConfig.value = ''
  formEnabled.value = true
  configError.value = ''
  dialogOpen.value = true
}

// Open edit dialog
function openEditDialog(item: Provider) {
  editingId.value = item.id
  dialogTitle.value = t('providers.editTitle')
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
    toast(t('providers.nameTypeRequired'), 'destructive')
    return
  }
  if (!validateConfig(formConfig.value)) {
    toast(t('providers.correctJson'), 'destructive')
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
      await providersApi.update(editingId.value, data)
      toast(t('providers.updated'), 'success')
    } else {
      await providersApi.create(data)
      toast(t('providers.created'), 'success')
    }
    dialogOpen.value = false
    fetchProviders()
  } catch (err: any) {
    toast(err.response?.data?.message || t('common.operationFailed'), 'destructive')
  } finally {
    submitting.value = false
  }
}

// Toggle enabled
async function handleToggleEnabled(item: Provider) {
  try {
    await providersApi.update(item.id, { enabled: !item.enabled })
    item.enabled = !item.enabled
    toast(t('providers.toggled', { name: item.name, status: item.enabled ? t('providers.enabled') : t('providers.disabled') }), 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || t('providers.toggleFailed'), 'destructive')
  }
}

// Delete
function openDeleteDialog(item: Provider) {
  deletingItem.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingItem.value) return
  deleting.value = true
  try {
    await providersApi.deleteProvider(deletingItem.value.id)
    toast(t('providers.deleted', { name: deletingItem.value.name }), 'success')
    fetchProviders()
  } catch (err: any) {
    toast(err.response?.data?.message || t('providers.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
    deletingItem.value = null
  }
}

function getDropdownItems(item: Provider) {
  return [
    { label: t('providers.edit'), icon: Pencil, action: () => openEditDialog(item) },
    { label: t('providers.delete'), icon: Trash2, danger: true, action: () => openDeleteDialog(item) },
  ]
}

onMounted(fetchProviders)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('providers.title')">
      <template #actions>
        <Button @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          {{ t('providers.addProvider') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Type filter tabs -->
    <Tabs v-model="activeTab" :tabs="tabItems" />

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
      v-else-if="providers.length > 0"
      :columns="columns"
      :data="providers"
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
        <Switch :model-value="row.enabled" @update:model-value="handleToggleEnabled(row as Provider)" />
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-actions="{ row }">
        <DropdownMenu :items="getDropdownItems(row as Provider)">
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
      :icon="Server"
      :title="t('providers.noProviders')"
      :description="t('providers.noProvidersDesc')"
      :action-text="t('providers.addProvider')"
      @action="openCreateDialog"
    />

    <!-- Create/Edit Dialog -->
    <Dialog
      v-model:open="dialogOpen"
      :title="dialogTitle"
      :description="t('providers.configDesc')"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('providers.name') }}</label>
          <Input v-model="formName" :placeholder="t('providers.namePlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('providers.type') }}</label>
          <Select v-model="formType" :options="typeOptions" :placeholder="t('providers.typePlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('providers.config') }}</label>
          <Textarea
            v-model="formConfig"
            placeholder='{"host": "192.168.1.1", "port": "22"}'
            :rows="5"
            class="font-mono text-xs"
            @input="validateConfig(formConfig)"
          />
          <p v-if="configError" class="text-xs text-destructive">{{ configError }}</p>
        </div>
        <div class="flex items-center gap-2">
          <Switch v-model="formEnabled" />
          <label class="text-sm text-foreground">{{ t('providers.enabled') }}</label>
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
      :title="t('providers.deleteConfirm')"
      :description="t('providers.deleteConfirmDesc', { name: deletingItem?.name || '' })"
      :confirm-text="t('providers.delete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
