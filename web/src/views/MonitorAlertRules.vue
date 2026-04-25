<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Table from '@/components/ui/Table.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { usePolling } from '@/composables/usePolling'
import * as monitorApi from '@/api/modules/monitor'
import type { AlertRule } from '@/types/models'

const { toast } = inject<any>('toast')!

// 轮询告警规则列表
const { data: rules, loading, refresh } = usePolling<AlertRule[]>({
  fetchFn: async () => {
    const res = await monitorApi.listAlertRules()
    if (res.data.status === 'success') {
      return res.data.data || []
    }
    return []
  },
  interval: 60000,
  autoStart: true,
})

// 表格列
const columns = [
  { key: 'name', label: '规则名' },
  { key: 'type', label: '指标类型' },
  { key: 'condition', label: '条件' },
  { key: 'threshold', label: '阈值' },
  { key: 'level', label: '严重程度' },
  { key: 'enabled', label: '启用状态' },
]

// 格式化条件
function formatCondition(condition: Record<string, any>): string {
  if (!condition) return '-'
  const op = condition.operator || condition.op || ''
  const value = condition.value ?? ''
  const duration = condition.duration ? ` 持续 ${condition.duration}s` : ''
  return `${op} ${value}${duration}`
}

// 获取阈值显示
function getThreshold(condition: Record<string, any>): string {
  if (!condition) return '-'
  return String(condition.value ?? '-')
}

// 严重程度样式
function getLevelBadge(level: string) {
  const l = (level || '').toLowerCase()
  switch (l) {
    case 'critical':
      return { variant: 'destructive' as const, label: '严重' }
    case 'warning':
      return { variant: 'warning' as const, label: '警告' }
    case 'info':
      return { variant: 'default' as const, label: '信息' }
    default:
      return { variant: 'secondary' as const, label: l || '-' }
  }
}

// 手动刷新
async function handleRefresh() {
  try {
    await refresh()
    toast('数据已刷新', 'success')
  } catch {
    toast('刷新失败', 'destructive')
  }
}
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader title="告警规则" description="系统告警规则配置（只读）">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="handleRefresh">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          刷新
        </Button>
      </template>
    </PageHeader>

    <!-- 规则表格 -->
    <Table
      :columns="columns"
      :data="rules.value || []"
      :loading="loading"
    >
      <template #cell-name="{ row }">
        <span class="text-sm font-medium text-foreground">{{ row.name }}</span>
      </template>
      <template #cell-type="{ row }">
        <Badge variant="secondary">{{ row.type || '-' }}</Badge>
      </template>
      <template #cell-condition="{ row }">
        <span class="text-sm text-muted-foreground">
          {{ formatCondition(row.condition) }}
        </span>
      </template>
      <template #cell-threshold="{ row }">
        <span class="text-sm text-foreground font-mono">
          {{ getThreshold(row.condition) }}
        </span>
      </template>
      <template #cell-level="{ row }">
        <Badge :variant="getLevelBadge(row.level).variant">
          {{ getLevelBadge(row.level).label }}
        </Badge>
      </template>
      <template #cell-enabled="{ row }">
        <Badge :variant="row.enabled ? 'success' : 'destructive'">
          {{ row.enabled ? '已启用' : '已禁用' }}
        </Badge>
      </template>
    </Table>
  </div>
</template>
