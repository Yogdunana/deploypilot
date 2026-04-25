<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { Plus, MoreHorizontal, Pencil, Trash2, Eye, EyeOff, KeyRound } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Select from '@/components/ui/Select.vue'
import Table from '@/components/ui/Table.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as credentialsApi from '@/api/modules/credentials'
import type { Credential } from '@/types/models'

const { toast } = inject<any>('toast')!

// State
const credentials = ref<Credential[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// Dialog
const dialogOpen = ref(false)
const dialogTitle = ref('创建凭证')
const editingId = ref<number | null>(null)
const formName = ref('')
const formType = ref('')
const formValue = ref('')
const showValue = ref(false)
const submitting = ref(false)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingItem = ref<Credential | null>(null)
const deleting = ref(false)

// Type options
const typeOptions = [
  { label: 'SSH 密钥', value: 'ssh' },
  { label: 'API Key', value: 'api_key' },
  { label: 'Token', value: 'token' },
]

// Table columns
const columns = [
  { key: 'name', label: '名称' },
  { key: 'type', label: '类型' },
  { key: 'created_at', label: '创建时间' },
  { key: 'actions', label: '操作', width: '80px' },
]

// Type badge mapping
function getTypeBadge(type: string) {
  const map: Record<string, { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning'; label: string }> = {
    ssh: { variant: 'default', label: 'SSH' },
    api_key: { variant: 'success', label: 'API Key' },
    token: { variant: 'warning', label: 'Token' },
  }
  return map[type] || { variant: 'secondary' as const, label: type }
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
    toast(err.response?.data?.message || '获取凭证列表失败', 'destructive')
  } finally {
    loading.value = false
  }
}

// Open create dialog
function openCreateDialog() {
  editingId.value = null
  dialogTitle.value = '创建凭证'
  formName.value = ''
  formType.value = ''
  formValue.value = ''
  showValue.value = false
  dialogOpen.value = true
}

// Open edit dialog
function openEditDialog(item: Credential) {
  editingId.value = item.id
  dialogTitle.value = '编辑凭证'
  formName.value = item.name
  formType.value = item.type
  formValue.value = ''
  showValue.value = false
  dialogOpen.value = true
}

// Submit form
async function handleSubmit() {
  if (!formName.value || !formType.value) {
    toast('请填写名称和类型', 'destructive')
    return
  }
  submitting.value = true
  try {
    const data = { name: formName.value, type: formType.value, value: formValue.value }
    if (editingId.value) {
      await credentialsApi.update(editingId.value, data)
      toast('凭证已更新', 'success')
    } else {
      await credentialsApi.create(data)
      toast('凭证已创建', 'success')
    }
    dialogOpen.value = false
    fetchCredentials()
  } catch (err: any) {
    toast(err.response?.data?.message || '操作失败', 'destructive')
  } finally {
    submitting.value = false
  }
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
    toast(`凭证「${deletingItem.value.name}」已删除`, 'success')
    fetchCredentials()
  } catch (err: any) {
    toast(err.response?.data?.message || '删除失败', 'destructive')
  } finally {
    deleting.value = false
    deletingItem.value = null
  }
}

function getDropdownItems(item: Credential) {
  return [
    { label: '编辑', icon: Pencil, action: () => openEditDialog(item) },
    { label: '删除', icon: Trash2, danger: true, action: () => openDeleteDialog(item) },
  ]
}

onMounted(fetchCredentials)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader title="凭证">
      <template #actions>
        <Button @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          创建凭证
        </Button>
      </template>
    </PageHeader>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-32" />
        <Skeleton class="h-4 w-20" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-8 ml-auto" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="credentials.length > 0"
      :columns="columns"
      :data="credentials"
    >
      <template #cell-name="{ row }">
        <span class="text-sm font-medium text-foreground">{{ row.name }}</span>
      </template>
      <template #cell-type="{ row }">
        <Badge :variant="getTypeBadge(row.type).variant">
          {{ getTypeBadge(row.type).label }}
        </Badge>
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-actions="{ row }">
        <DropdownMenu :items="getDropdownItems(row as Credential)">
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
      :icon="KeyRound"
      title="暂无凭证"
      description="点击上方按钮创建你的第一个凭证"
      action-text="创建凭证"
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
    <Dialog
      v-model:open="dialogOpen"
      :title="dialogTitle"
      description="配置凭证信息"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">名称</label>
          <Input v-model="formName" placeholder="输入凭证名称" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">类型</label>
          <Select v-model="formType" :options="typeOptions" placeholder="选择凭证类型" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">值</label>
          <div class="relative">
            <Textarea
              v-model="formValue"
              :type="showValue ? 'text' : 'password'"
              placeholder="输入凭证值"
              :rows="3"
              class="pr-10"
            />
            <button
              type="button"
              class="absolute right-3 top-3 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
              @click="showValue = !showValue"
            >
              <EyeOff v-if="showValue" class="w-4 h-4" />
              <Eye v-else class="w-4 h-4" />
            </button>
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="dialogOpen = false">取消</Button>
          <Button :loading="submitting" @click="handleSubmit">
            {{ editingId ? '保存' : '创建' }}
          </Button>
        </div>
      </div>
    </Dialog>

    <!-- Delete AlertDialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      title="删除凭证"
      :description="`确定要删除凭证「${deletingItem?.name}」吗？此操作不可撤销。`"
      confirm-text="删除"
      cancel-text="取消"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
