<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { Search, FileText } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Table from '@/components/ui/Table.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as auditApi from '@/api/modules/audit'
import type { AuditLog } from '@/types/models'

const { toast } = inject<any>('toast')!

// State
const logs = ref<AuditLog[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)

// Filters
const filterUsername = ref('')
const filterAction = ref('')
const filterResourceType = ref('')

// Action options
const actionOptions = [
  { label: '全部操作', value: '' },
  { label: '创建', value: 'create' },
  { label: '更新', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '登录', value: 'login' },
  { label: '部署', value: 'deploy' },
]

// Resource type options
const resourceTypeOptions = [
  { label: '全部资源', value: '' },
  { label: '应用', value: 'app' },
  { label: '服务器', value: 'server' },
  { label: '凭证', value: 'credential' },
  { label: 'DNS', value: 'dns' },
  { label: '用户', value: 'user' },
  { label: '模板', value: 'template' },
  { label: '提供商', value: 'provider' },
  { label: '通知', value: 'notification' },
]

// Table columns
const columns = [
  { key: 'created_at', label: '时间' },
  { key: 'username', label: '用户' },
  { key: 'action', label: '操作' },
  { key: 'resource_type', label: '资源类型' },
  { key: 'resource_id', label: '资源 ID' },
  { key: 'detail', label: '详情' },
  { key: 'ip_address', label: 'IP 地址' },
]

// Fetch logs
async function fetchLogs() {
  loading.value = true
  try {
    const params: auditApi.AuditLogParams = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (filterAction.value) params.action = filterAction.value
    if (filterResourceType.value) params.resource_type = filterResourceType.value
    const res = await auditApi.listLogs(params)
    if (res.data.status === 'success') {
      logs.value = res.data.data
      total.value = res.data.pagination?.total || 0
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '获取审计日志失败', 'destructive')
  } finally {
    loading.value = false
  }
}

// Apply filters
function applyFilters() {
  page.value = 1
  fetchLogs()
}

// Truncate detail text
function truncateDetail(text: string, maxLen: number = 50): string {
  if (!text) return '-'
  return text.length > maxLen ? text.slice(0, maxLen) + '...' : text
}

onMounted(fetchLogs)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader title="审计日志" />

    <!-- Filters -->
    <div class="flex items-center gap-3 flex-wrap">
      <div class="relative w-56">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          v-model="filterUsername"
          placeholder="用户名..."
          class="pl-9"
          @keyup.enter="applyFilters"
        />
      </div>
      <Select
        v-model="filterAction"
        :options="actionOptions"
        placeholder="操作类型"
        class="w-36"
        @update:model-value="applyFilters"
      />
      <Select
        v-model="filterResourceType"
        :options="resourceTypeOptions"
        placeholder="资源类型"
        class="w-36"
        @update:model-value="applyFilters"
      />
      <Button variant="outline" size="sm" @click="applyFilters">搜索</Button>
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 8" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-20" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-20" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-40" />
        <Skeleton class="h-4 w-28" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="logs.length > 0"
      :columns="columns"
      :data="logs"
    >
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
      <template #cell-username="{ row }">
        <span class="text-sm font-medium text-foreground">{{ row.username }}</span>
      </template>
      <template #cell-action="{ row }">
        <span class="text-sm text-muted-foreground">{{ row.action }}</span>
      </template>
      <template #cell-resource_type="{ row }">
        <span class="text-sm text-muted-foreground">{{ row.resource_type }}</span>
      </template>
      <template #cell-resource_id="{ row }">
        <code class="text-xs font-mono text-muted-foreground">{{ row.resource_id }}</code>
      </template>
      <template #cell-detail="{ row }">
        <span class="text-sm text-muted-foreground" :title="row.detail">{{ truncateDetail(row.detail) }}</span>
      </template>
      <template #cell-ip_address="{ row }">
        <code class="text-xs font-mono text-muted-foreground">{{ row.ip_address }}</code>
      </template>
    </Table>

    <!-- Empty state -->
    <EmptyState
      v-else
      :icon="FileText"
      title="暂无审计日志"
      description="暂无操作日志记录"
    />

    <!-- Pagination -->
    <div v-if="total > pageSize" class="flex justify-end">
      <Pagination
        v-model:page="page"
        :page-size="pageSize"
        :total="total"
        @update:page="fetchLogs"
      />
    </div>
  </div>
</template>
