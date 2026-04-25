<script setup lang="ts">
import { ref, computed, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Search, Plus, MoreHorizontal, Eye, Zap, Terminal,
  Cpu, Pencil, Trash2, Server as ServerIcon,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Table from '@/components/ui/Table.vue'
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as serversApi from '@/api/modules/servers'
import type { Server } from '@/types/models'

const router = useRouter()
const { toast } = inject<any>('toast')!

// State
const servers = ref<Server[]>([])
const loading = ref(true)
const searchQuery = ref('')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingServer = ref<Server | null>(null)
const deleting = ref(false)

// Testing connection
const testingId = ref<number | null>(null)

// Table columns
const columns = [
  { key: 'name', label: '名称' },
  { key: 'host_port', label: '主机:端口' },
  { key: 'status', label: '状态' },
  { key: 'tags', label: '标签' },
  { key: 'created_at', label: '创建时间' },
  { key: 'actions', label: '操作', width: '80px' },
]

// Filtered servers
const filteredServers = computed(() => {
  if (!searchQuery.value) return servers.value
  const q = searchQuery.value.toLowerCase()
  return servers.value.filter(
    (s) => s.name.toLowerCase().includes(q) || s.host.toLowerCase().includes(q)
  )
})

// Paginated servers
const paginatedServers = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredServers.value.slice(start, start + pageSize.value)
})

const paginatedTotal = computed(() => filteredServers.value.length)

// Map server status to StatusBadge status
function mapServerStatus(status: string): string {
  const s = status.toLowerCase()
  if (s === 'reachable' || s === 'online' || s === 'connected') return 'success'
  if (s === 'unreachable' || s === 'offline') return 'destructive'
  return 'secondary'
}

// Fetch servers
async function fetchServers() {
  loading.value = true
  try {
    const res = await serversApi.list({ page: 1, page_size: 1000 })
    if (res.data.status === 'success') {
      servers.value = res.data.data
      total.value = res.data.pagination?.total || servers.value.length
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '获取服务器列表失败', 'destructive')
  } finally {
    loading.value = false
  }
}

// Test connection
async function handleTestConnection(server: Server) {
  testingId.value = server.id
  try {
    const res = await serversApi.test(server.id)
    if (res.data.status === 'success' && res.data.data.success) {
      toast(`服务器 "${server.name}" 连接成功`, 'success')
    } else {
      toast(res.data.data?.message || `服务器 "${server.name}" 连接失败`, 'destructive')
    }
  } catch (err: any) {
    toast(err.response?.data?.message || `服务器 "${server.name}" 连接测试失败`, 'destructive')
  } finally {
    testingId.value = null
  }
}

// Detect environment
async function handleDetect(server: Server) {
  try {
    await serversApi.detect(server.id, { host: server.host, port: server.port })
    toast(`服务器 "${server.name}" 环境检测已触发`, 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || '环境检测失败', 'destructive')
  }
}

// Delete
function openDeleteDialog(server: Server) {
  deletingServer.value = server
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingServer.value) return
  deleting.value = true
  try {
    await serversApi.deleteServer(deletingServer.value.id)
    toast(`服务器 "${deletingServer.value.name}" 已删除`, 'success')
    fetchServers()
  } catch (err: any) {
    toast(err.response?.data?.message || '删除失败', 'destructive')
  } finally {
    deleting.value = false
    deletingServer.value = null
  }
}

// Get dropdown items for a server
function getServerActions(server: Server) {
  return [
    { label: '详情', icon: Eye, action: () => router.push(`/servers/${server.id}`) },
    { label: '测试连接', icon: Zap, action: () => handleTestConnection(server) },
    { label: '终端', icon: Terminal, action: () => router.push(`/servers/${server.id}/terminal`) },
    { label: '环境检测', icon: Cpu, action: () => handleDetect(server) },
    { label: '编辑', icon: Pencil, action: () => router.push(`/servers/${server.id}`) },
    { label: '删除', icon: Trash2, danger: true, action: () => openDeleteDialog(server) },
  ]
}

onMounted(fetchServers)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader title="服务器">
      <template #actions>
        <Button @click="router.push('/servers/create')">
          <template #icon><Plus class="w-4 h-4" /></template>
          添加服务器
        </Button>
      </template>
    </PageHeader>

    <!-- Search -->
    <div class="flex items-center gap-4">
      <div class="relative w-72">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          v-model="searchQuery"
          placeholder="搜索服务器名称或主机..."
          class="pl-9"
        />
      </div>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-36" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-24" />
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-8 ml-auto" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="paginatedServers.length > 0"
      :columns="columns"
      :data="paginatedServers"
    >
      <template #cell-name="{ row }">
        <button
          class="text-sm font-medium text-foreground hover:text-primary transition-colors cursor-pointer"
          @click="router.push(`/servers/${row.id}`)"
        >
          {{ row.name }}
        </button>
      </template>
      <template #cell-host_port="{ row }">
        <span class="text-sm text-muted-foreground font-mono">{{ row.host }}:{{ row.port }}</span>
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :status="mapServerStatus(row.status)" />
      </template>
      <template #cell-tags="{ row }">
        <div v-if="row.tags && row.tags.length > 0" class="flex items-center gap-1 flex-wrap">
          <Badge v-for="tag in row.tags.slice(0, 3)" :key="tag" variant="outline" class="text-xs">
            {{ tag }}
          </Badge>
          <Badge v-if="row.tags.length > 3" variant="secondary" class="text-xs">
            +{{ row.tags.length - 3 }}
          </Badge>
        </div>
        <span v-else class="text-sm text-muted-foreground">-</span>
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-actions="{ row }">
        <DropdownMenu :items="getServerActions(row as Server)">
          <template #trigger>
            <Button variant="ghost" size="icon" :loading="testingId === row.id">
              <MoreHorizontal v-if="testingId !== row.id" class="w-4 h-4" />
            </Button>
          </template>
        </DropdownMenu>
      </template>
    </Table>

    <!-- Empty state -->
    <EmptyState
      v-else
      :icon="ServerIcon"
      title="暂无服务器"
      description="点击上方按钮添加你的第一台服务器"
      action-text="添加服务器"
      @action="router.push('/servers/create')"
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
      title="删除服务器"
      :description="`确定要删除服务器「${deletingServer?.name}」吗？此操作不可撤销。`"
      confirm-text="删除"
      cancel-text="取消"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
