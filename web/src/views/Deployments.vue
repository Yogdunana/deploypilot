<script setup lang="ts">
import { ref, computed, inject, onMounted } from 'vue'
import {
  Search, Rocket,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Table from '@/components/ui/Table.vue'
import Pagination from '@/components/ui/Pagination.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as deploymentsApi from '@/api/modules/deployments'
import type { DeploymentRecord } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = inject<any>('toast')!
const { t } = useI18n()

// State
const deployments = ref<DeploymentRecord[]>([])
const loading = ref(true)
const searchQuery = ref('')
const statusFilter = ref<string | number>('')
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)

// Status filter options
const statusOptions = computed(() => [
  { label: t('deployments.allStatus'), value: '' },
  { label: t('deployments.success'), value: 'success' },
  { label: t('deployments.failed'), value: 'failed' },
  { label: t('deployments.deploying'), value: 'deploying' },
  { label: t('deployments.pending'), value: 'pending' },
  { label: t('deployments.building'), value: 'building' },
])

// Table columns
const columns = computed(() => [
  { key: 'app_name', label: t('deployments.appName') },
  { key: 'container_name', label: t('deployments.containerName') },
  { key: 'image', label: t('deployments.image') },
  { key: 'status', label: t('deployments.status') },
  { key: 'error_message', label: t('deployments.errorMessage') },
  { key: 'created_at', label: t('deployments.createdAt') },
])

// Filtered deployments
const filteredDeployments = computed(() => {
  let result = deployments.value
  if (searchQuery.value) {
    const q = searchQuery.value.toLowerCase()
    result = result.filter(
      (d) => d.app_name.toLowerCase().includes(q) || d.container_name.toLowerCase().includes(q)
    )
  }
  if (statusFilter.value) {
    result = result.filter((d) => d.status.toLowerCase() === String(statusFilter.value).toLowerCase())
  }
  return result
})

// Paginated deployments
const paginatedDeployments = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return filteredDeployments.value.slice(start, start + pageSize.value)
})

const paginatedTotal = computed(() => filteredDeployments.value.length)

// Fetch deployments
async function fetchDeployments() {
  loading.value = true
  try {
    const res = await deploymentsApi.list(undefined, undefined, { page: 1, page_size: 1000 })
    if (res.data.status === 'success') {
      deployments.value = res.data.data
      total.value = res.data.pagination?.total || deployments.value.length
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('deployments.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Click row
function handleRowClick(row: DeploymentRecord) {
  toast(t('deployments.detailInDev', { id: row.id }))
}

onMounted(fetchDeployments)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('deployments.title')" />

    <!-- Search & Filters -->
    <div class="flex items-center gap-4">
      <div class="relative w-72">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
        <Input
          v-model="searchQuery"
          :placeholder="t('deployments.searchPlaceholder')"
          class="pl-9"
        />
      </div>
      <Select
        v-model="statusFilter"
        :options="statusOptions"
        :placeholder="t('deployments.allStatus')"
        class="w-40"
      />
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading" class="rounded-lg border border-border bg-card">
      <div v-for="i in 5" :key="i" class="flex items-center gap-4 px-4 py-3 border-b border-border last:border-0">
        <Skeleton class="h-4 w-28" />
        <Skeleton class="h-4 w-32" />
        <Skeleton class="h-4 w-48" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-40" />
        <Skeleton class="h-4 w-28" />
      </div>
    </div>

    <!-- Table -->
    <Table
      v-else-if="paginatedDeployments.length > 0"
      :columns="columns"
      :data="paginatedDeployments"
    >
      <template #cell-app_name="{ row }">
        <button
          class="text-sm font-medium text-foreground hover:text-primary transition-colors cursor-pointer"
          @click="handleRowClick(row as DeploymentRecord)"
        >
          {{ row.app_name }}
        </button>
      </template>
      <template #cell-container_name="{ row }">
        <span class="text-sm text-muted-foreground font-mono">{{ row.container_name || '-' }}</span>
      </template>
      <template #cell-image="{ row }">
        <span class="text-sm text-muted-foreground truncate max-w-[200px] inline-block" :title="row.image">
          {{ row.image || '-' }}
        </span>
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :status="row.status" />
      </template>
      <template #cell-error_message="{ row }">
        <span
          v-if="row.error_message"
          class="text-sm text-destructive truncate max-w-[200px] inline-block"
          :title="row.error_message"
        >
          {{ row.error_message }}
        </span>
        <span v-else class="text-sm text-muted-foreground">-</span>
      </template>
      <template #cell-created_at="{ row }">
        <RelativeTime :date="row.created_at" />
      </template>
    </Table>

    <!-- Empty state -->
    <EmptyState
      v-else
      :icon="Rocket"
      :title="t('deployments.noDeployments')"
      :description="t('deployments.noDeploymentsDesc')"
    />

    <!-- Pagination -->
    <div v-if="paginatedTotal > pageSize" class="flex justify-end">
      <Pagination
        v-model:page="page"
        :page-size="pageSize"
        :total="paginatedTotal"
      />
    </div>
  </div>
</template>
