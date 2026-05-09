<script setup lang="ts">
import { ref, computed, inject, onMounted } from 'vue'
import { Search, FileText, Calendar } from 'lucide-vue-next'
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
import { useI18n } from 'vue-i18n'

const { toast } = inject<any>('toast')!
const { t } = useI18n()

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
const filterStartDate = ref('')
const filterEndDate = ref('')

// Action options
const actionOptions = computed(() => [
  { label: t('audit.allActions'), value: '' },
  { label: t('audit.create'), value: 'create' },
  { label: t('audit.update'), value: 'update' },
  { label: t('audit.delete'), value: 'delete' },
  { label: t('audit.login'), value: 'login' },
  { label: t('audit.deploy'), value: 'deploy' },
])

// Resource type options
const resourceTypeOptions = computed(() => [
  { label: t('audit.allResources'), value: '' },
  { label: t('audit.app'), value: 'app' },
  { label: t('audit.server'), value: 'server' },
  { label: t('audit.credential'), value: 'credential' },
  { label: t('audit.dns'), value: 'dns' },
  { label: t('audit.userType'), value: 'user' },
  { label: t('audit.template'), value: 'template' },
  { label: t('audit.provider'), value: 'provider' },
  { label: t('audit.notification'), value: 'notification' },
])

// Table columns
const columns = computed(() => [
  { key: 'created_at', label: t('audit.time') },
  { key: 'username', label: t('audit.user') },
  { key: 'action', label: t('audit.action') },
  { key: 'resource_type', label: t('audit.resourceType') },
  { key: 'resource_id', label: t('audit.resourceId') },
  { key: 'detail', label: t('audit.detail') },
  { key: 'ip_address', label: t('audit.ipAddress') },
])

// Fetch logs
async function fetchLogs() {
  loading.value = true
  try {
    const params: auditApi.AuditLogParams = {
      page: page.value,
      page_size: pageSize.value,
    }
    if (filterUsername.value) params.username = filterUsername.value
    if (filterAction.value) params.action = filterAction.value
    if (filterResourceType.value) params.resource_type = filterResourceType.value
    if (filterStartDate.value) params.start_date = filterStartDate.value
    if (filterEndDate.value) params.end_date = filterEndDate.value
    const res = await auditApi.listLogs(params)
    if (res.data.status === 'success') {
      logs.value = res.data.data
      total.value = res.data.pagination?.total || 0
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('audit.fetchFailed'), 'destructive')
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
    <PageHeader :title="t('audit.title')" />

    <!-- Filters -->
    <div class="flex items-center gap-3 flex-wrap">
      <div class="relative w-56">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          v-model="filterUsername"
          :placeholder="t('audit.searchPlaceholder')"
          class="pl-9"
          @keyup.enter="applyFilters"
        />
      </div>
      <Select
        v-model="filterAction"
        :options="actionOptions"
        :placeholder="t('audit.actionType')"
        class="w-36"
        @update:model-value="applyFilters"
      />
      <Select
        v-model="filterResourceType"
        :options="resourceTypeOptions"
        :placeholder="t('audit.resourceType')"
        class="w-36"
        @update:model-value="applyFilters"
      />
      <div class="relative w-40">
        <Calendar class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
        <Input
          v-model="filterStartDate"
          type="date"
          :placeholder="t('audit.startDate')"
          class="pl-9"
          @change="applyFilters"
        />
      </div>
      <div class="relative w-40">
        <Calendar class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground pointer-events-none" />
        <Input
          v-model="filterEndDate"
          type="date"
          :placeholder="t('audit.endDate')"
          class="pl-9"
          @change="applyFilters"
        />
      </div>
      <Button variant="outline" size="sm" @click="applyFilters">{{ t('audit.search') }}</Button>
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
      :title="t('audit.noLogs')"
      :description="t('audit.noLogsDesc')"
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
