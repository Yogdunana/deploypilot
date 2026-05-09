<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { Plus, MoreHorizontal, Pencil, Trash2, Search, Globe } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Select from '@/components/ui/Select.vue'
import Table from '@/components/ui/ResponsiveTable.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as dnsApi from '@/api/modules/dns'
import type { DNSRecord } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = useToast()
const { t } = useI18n()

// State
const records = ref<DNSRecord[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const domainFilter = ref('')

// Dialog
const dialogOpen = ref(false)
const dialogTitle = ref(t('dns.createTitle'))
const editingId = ref<number | null>(null)
const formDomain = ref('')
const formSubdomain = ref('')
const formType = ref('')
const formValue = ref('')
const submitting = ref(false)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingItem = ref<DNSRecord | null>(null)
const deleting = ref(false)

// Type options
const typeOptions = [
  { label: 'A', value: 'A' },
  { label: 'CNAME', value: 'CNAME' },
  { label: 'TXT', value: 'TXT' },
  { label: 'MX', value: 'MX' },
]

// Table columns
const columns = computed(() => [
  { key: 'domain', label: t('dns.domain'), mobile: true },
  { key: 'subdomain', label: t('dns.subdomain'), mobile: true },
  { key: 'type', label: t('dns.type'), mobile: true },
  { key: 'value', label: t('dns.value'), mobile: true },
  { key: 'created_at', label: t('dns.createdAt') },
  { key: 'actions', label: t('dns.actions'), width: '80px' },
])

// Type badge mapping
function getTypeBadge(type: string) {
  const map: Record<string, { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning'; label: string }> = {
    A: { variant: 'default', label: 'A' },
    CNAME: { variant: 'success', label: 'CNAME' },
    TXT: { variant: 'warning', label: 'TXT' },
    MX: { variant: 'outline', label: 'MX' },
  }
  return map[type] || { variant: 'secondary' as const, label: type }
}

// Fetch records
async function fetchRecords() {
  loading.value = true
  try {
    const res = await dnsApi.listRecords(domainFilter.value, { page: page.value, page_size: pageSize.value })
    if (res.data.status === 'success') {
      records.value = res.data.data
      total.value = res.data.pagination?.total || 0
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('dns.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Search
function handleSearch() {
  page.value = 1
  fetchRecords()
}

// Open create dialog
function openCreateDialog() {
  editingId.value = null
  dialogTitle.value = t('dns.createTitle')
  formDomain.value = ''
  formSubdomain.value = ''
  formType.value = ''
  formValue.value = ''
  dialogOpen.value = true
}

// Open edit dialog
function openEditDialog(item: DNSRecord) {
  editingId.value = item.id
  dialogTitle.value = t('dns.editTitle')
  formDomain.value = item.domain
  formSubdomain.value = item.subdomain
  formType.value = item.type
  formValue.value = item.value
  dialogOpen.value = true
}

// Submit form
async function handleSubmit() {
  if (!formDomain.value || !formType.value || !formValue.value) {
    toast(t('dns.domainRequired'), 'destructive')
    return
  }
  submitting.value = true
  try {
    const data = { domain: formDomain.value, subdomain: formSubdomain.value, type: formType.value, value: formValue.value }
    if (editingId.value) {
      await dnsApi.updateRecord(editingId.value, data)
      toast(t('dns.updated'), 'success')
    } else {
      await dnsApi.createRecord(data)
      toast(t('dns.created'), 'success')
    }
    dialogOpen.value = false
    fetchRecords()
  } catch (err: any) {
    toast(err.response?.data?.message || t('common.operationFailed'), 'destructive')
  } finally {
    submitting.value = false
  }
}

// Delete
function openDeleteDialog(item: DNSRecord) {
  deletingItem.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingItem.value) return
  deleting.value = true
  try {
    await dnsApi.deleteRecord(deletingItem.value.id)
    toast(t('dns.deleted', { record: `${deletingItem.value.subdomain}.${deletingItem.value.domain}` }), 'success')
    fetchRecords()
  } catch (err: any) {
    toast(err.response?.data?.message || t('dns.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
    deletingItem.value = null
  }
}

function getDropdownItems(item: DNSRecord) {
  return [
    { label: t('dns.edit'), icon: Pencil, action: () => openEditDialog(item) },
    { label: t('dns.delete'), icon: Trash2, danger: true, action: () => openDeleteDialog(item) },
  ]
}

onMounted(fetchRecords)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('dns.title')">
      <template #actions>
        <Button @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          {{ t('dns.addRecord') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Filter -->
    <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 sm:gap-4">
      <div class="relative w-full sm:w-72">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          v-model="domainFilter"
          :placeholder="t('dns.searchPlaceholder')"
          class="pl-9"
          @keyup.enter="handleSearch"
        />
      </div>
      <Button variant="outline" size="sm" @click="handleSearch">{{ t('dns.search') }}</Button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-32" />
        <Skeleton class="h-4 w-24" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-36" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-8 ml-auto" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="records.length > 0"
      :columns="columns"
      :data="records"
    >
      <template #cell-domain="{ row }">
        <span class="text-sm font-medium text-foreground">{{ row.domain }}</span>
      </template>
      <template #cell-subdomain="{ row }">
        <span class="text-sm text-muted-foreground font-mono">{{ row.subdomain || '-' }}</span>
      </template>
      <template #cell-type="{ row }">
        <Badge :variant="getTypeBadge(row.type).variant">
          {{ getTypeBadge(row.type).label }}
        </Badge>
      </template>
      <template #cell-value="{ row }">
        <span class="text-sm text-muted-foreground font-mono truncate max-w-[120px] sm:max-w-[200px] inline-block">{{ row.value }}</span>
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-actions="{ row }">
        <DropdownMenu :items="getDropdownItems(row as DNSRecord)">
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
      :icon="Globe"
      :title="t('dns.noRecords')"
      :description="t('dns.noRecordsDesc')"
      :action-text="t('dns.addRecord')"
      @action="openCreateDialog"
    />

    <!-- Pagination -->
    <div v-if="total > pageSize" class="flex justify-end">
      <Pagination
        v-model:page="page"
        :page-size="pageSize"
        :total="total"
        @update:page="fetchRecords"
      />
    </div>

    <!-- Create/Edit Dialog -->
    <Dialog
      v-model:open="dialogOpen"
      :title="dialogTitle"
      :description="t('dns.configDesc')"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('dns.domain') }}</label>
          <Input v-model="formDomain" :placeholder="t('dns.domainPlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('dns.subdomain') }}</label>
          <Input v-model="formSubdomain" :placeholder="t('dns.subdomainPlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('dns.type') }}</label>
          <Select v-model="formType" :options="typeOptions" :placeholder="t('dns.typePlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('dns.value') }}</label>
          <Input v-model="formValue" :placeholder="t('dns.valuePlaceholder')" />
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
      :title="t('dns.deleteConfirm')"
      :description="t('dns.deleteConfirmDesc', { record: `${deletingItem?.subdomain || ''}.${deletingItem?.domain || ''}` })"
      :confirm-text="t('dns.delete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
