<script setup lang="ts">
import { ref, computed, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Search, Plus, MoreHorizontal, Eye, Rocket, FileText,
  RotateCcw, Trash2, Package,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Tabs from '@/components/ui/Tabs.vue'
import Table from '@/components/ui/Table.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as appsApi from '@/api/modules/apps'
import type { App } from '@/types/models'

const router = useRouter()
const { toast } = inject<any>('toast')!

// State
const apps = ref<App[]>([])
const loading = ref(true)
const searchQuery = ref('')
const activeTab = ref('all')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingApp = ref<App | null>(null)
const deleting = ref(false)

// Status tabs
const statusTabs = [
  { key: 'all', label: '全部' },
  { key: 'running', label: '运行中' },
  { key: 'stopped', label: '已停止' },
  { key: 'failed', label: '失败' },
  { key: 'deploying', label: '部署中' },
]

// Table columns
const columns = [
  { key: 'name', label: '名称' },
  { key: 'tech_stack', label: '技术栈' },
  { key: 'status', label: '状态' },
  { key: 'domain', label: '域名' },
  { key: 'server', label: '服务器' },
  { key: 'created_at', label: '创建时间' },
  { key: 'actions', label: '操作', width: '80px' },
]

// Filtered apps
const filteredApps = computed(() => {
  let result = apps.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter((app) => app.name.toLowerCase().includes(q))
  }
  if (activeTab.value !== 'all') {
    result = result.filter((app) => app.status.toLowerCase() === activeTab.value)
  }
  return result
})

// Paginated apps
const paginatedApps = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredApps.value.slice(start, start + pageSize.value)
})

const paginatedTotal = computed(() => filteredApps.value.length)

// Fetch apps
async function fetchApps() {
  loading.value = true
  try {
    const res = await appsApi.list({ page: 1, page_size: 1000 })
    if (res.data.status === 'success') {
      apps.value = res.data.data
      total.value = res.data.pagination?.total || apps.value.length
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '获取应用列表失败', 'destructive')
  } finally {
    loading.value = false
  }
}

// Deploy
async function handleDeploy(app: App) {
  try {
    await appsApi.deploy(app.id)
    toast(`应用 "${app.name}" 部署已触发`, 'success')
    fetchApps()
  } catch (err: any) {
    toast(err.response?.data?.message || '部署失败', 'destructive')
  }
}

// Delete
function openDeleteDialog(app: App) {
  deletingApp.value = app
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingApp.value) return
  deleting.value = true
  try {
    await appsApi.deleteApp(deletingApp.value.id)
    toast(`应用 "${deletingApp.value.name}" 已删除`, 'success')
    fetchApps()
  } catch (err: any) {
    toast(err.response?.data?.message || '删除失败', 'destructive')
  } finally {
    deleting.value = false
    deletingApp.value = null
  }
}

// Get dropdown items for an app
function getAppActions(app: App) {
  return [
    { label: '详情', icon: Eye, action: () => router.push(`/apps/${app.id}`) },
    { label: '部署', icon: Rocket, action: () => handleDeploy(app) },
    { label: '日志', icon: FileText, action: () => router.push(`/apps/${app.id}`) },
    { label: '回滚', icon: RotateCcw, action: () => toast('回滚功能请在应用详情页操作') },
    { label: '删除', icon: Trash2, danger: true, action: () => openDeleteDialog(app) },
  ]
}

function getTechStackVariant(stack: string): 'default' | 'secondary' | 'outline' | 'success' | 'warning' | 'destructive' {
  const s = stack.toLowerCase()
  if (s.includes('node') || s.includes('react') || s.includes('vue') || s.includes('next')) return 'success'
  if (s.includes('python') || s.includes('django') || s.includes('flask')) return 'warning'
  if (s.includes('go') || s.includes('rust') || s.includes('java')) return 'destructive'
  if (s.includes('php') || s.includes('laravel')) return 'outline'
  return 'secondary'
}

onMounted(fetchApps)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader title="应用">
      <template #actions>
        <Button @click="router.push('/apps/create')">
          <template #icon><Plus class="w-4 h-4" /></template>
          创建应用
        </Button>
      </template>
    </PageHeader>

    <!-- Search & Filters -->
    <div class="flex items-center gap-4">
      <div class="relative w-72">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          v-model="searchQuery"
          placeholder="搜索应用名称..."
          class="pl-9"
        />
      </div>
      <Tabs v-model="activeTab" :tabs="statusTabs" />
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-32" />
        <Skeleton class="h-4 w-20" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-40" />
        <Skeleton class="h-4 w-24" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-8 ml-auto" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="paginatedApps.length > 0"
      :columns="columns"
      :data="paginatedApps"
    >
      <template #cell-name="{ row }">
        <button
          class="text-sm font-medium text-foreground hover:text-primary transition-colors cursor-pointer"
          @click="router.push(`/apps/${row.id}`)"
        >
          {{ row.name }}
        </button>
      </template>
      <template #cell-tech_stack="{ row }">
        <Badge v-if="row.tech_stack" :variant="getTechStackVariant(row.tech_stack)">
          {{ row.tech_stack }}
        </Badge>
        <span v-else class="text-sm text-muted-foreground">-</span>
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :status="row.status" />
      </template>
      <template #cell-domain="{ row }">
        <span class="text-sm text-muted-foreground">{{ row.domain || '-' }}</span>
      </template>
      <template #cell-server="{ row }">
        <span class="text-sm text-muted-foreground">{{ row.server_id || '-' }}</span>
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-actions="{ row }">
        <DropdownMenu :items="getAppActions(row as App)">
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
      :icon="Package"
      title="暂无应用"
      description="点击上方按钮创建你的第一个应用"
      action-text="创建应用"
      @action="router.push('/apps/create')"
    />

    <!-- Pagination -->
    <div v-if="paginatedTotal > pageSize" class="flex justify-end">
      <Pagination
        v-model:page="page"
        :page-size="pageSize"
        :total="paginatedTotal"
      />
    </div>

    <!-- Delete confirmation dialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      title="删除应用"
      :description="`确定要删除应用「${deletingApp?.name}」吗？此操作不可撤销。`"
      confirm-text="删除"
      cancel-text="取消"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
