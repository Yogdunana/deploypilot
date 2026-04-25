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
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()
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
const columns = computed(() => [
  { key: 'name', label: t('servers.name') },
  { key: 'host_port', label: t('servers.hostPort') },
  { key: 'status', label: t('servers.status') },
  { key: 'tags', label: t('servers.tags') },
  { key: 'created_at', label: t('servers.createdAt') },
  { key: 'actions', label: t('servers.actions'), width: '80px' },
])

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
    toast(err.response?.data?.message || t('servers.fetchFailed'), 'destructive')
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
      toast(t('servers.connectionSuccess', { name: server.name }), 'success')
    } else {
      toast(res.data.data?.message || t('servers.connectionFailed', { name: server.name }), 'destructive')
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('servers.connectionTestFailed', { name: server.name }), 'destructive')
  } finally {
    testingId.value = null
  }
}

// Detect environment
async function handleDetect(server: Server) {
  try {
    await serversApi.detect(server.id, { host: server.host, port: server.port })
    toast(t('servers.detectTriggered', { name: server.name }), 'success')
  } catch (err: any) {
    toast(err.response?.data?.message || t('servers.detectFailed'), 'destructive')
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
    toast(t('servers.deleted', { name: deletingServer.value.name }), 'success')
    fetchServers()
  } catch (err: any) {
    toast(err.response?.data?.message || t('servers.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
    deletingServer.value = null
  }
}

// Get dropdown items for a server
function getServerActions(server: Server) {
  return [
    { label: t('servers.detail'), icon: Eye, action: () => router.push(`/servers/${server.id}`) },
    { label: t('servers.testConnection'), icon: Zap, action: () => handleTestConnection(server) },
    { label: t('servers.terminal'), icon: Terminal, action: () => router.push(`/servers/${server.id}/terminal`) },
    { label: t('servers.detect'), icon: Cpu, action: () => handleDetect(server) },
    { label: t('servers.edit'), icon: Pencil, action: () => router.push(`/servers/${server.id}`) },
    { label: t('servers.delete'), icon: Trash2, danger: true, action: () => openDeleteDialog(server) },
  ]
}

onMounted(fetchServers)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('servers.title')">
      <template #actions>
        <Button @click="router.push('/servers/create')">
          <template #icon><Plus class="w-4 h-4" /></template>
          {{ t('servers.addServer') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Search -->
    <div class="flex items-center gap-4">
      <div class="relative w-72">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          v-model="searchQuery"
          :placeholder="t('servers.searchPlaceholder')"
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
      :title="t('servers.noServers')"
      :description="t('servers.noServersDesc')"
      :action-text="t('servers.addServer')"
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
      :title="t('servers.deleteConfirm')"
      :description="t('servers.deleteConfirmDesc', { name: deletingServer?.name || '' })"
      :confirm-text="t('servers.delete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
