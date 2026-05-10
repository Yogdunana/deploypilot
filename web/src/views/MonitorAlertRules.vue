<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { RefreshCw } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Table from '@/components/ui/Table.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { usePolling } from '@/composables/usePolling'
import * as monitorApi from '@/api/modules/monitor'
import type { AlertRule } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = useToast()
const { t } = useI18n()

// 轮询告警规则列表
const { data: rulesRef, loading, refresh } = usePolling<AlertRule[]>({
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

// Unwrap ref for template
const rules = computed(() => rulesRef.value)

// 表格列
const columns = computed(() => [
  { key: 'name', label: t('monitorAlertRules.name') },
  { key: 'type', label: t('monitorAlertRules.metric') },
  { key: 'condition', label: t('monitorAlertRules.condition') },
  { key: 'threshold', label: t('monitorAlertRules.threshold') },
  { key: 'level', label: t('monitorAlertRules.severity') },
  { key: 'enabled', label: t('monitorAlertRules.enabled') },
])

// 格式化条件
function formatCondition(condition: Record<string, any>): string {
  if (!condition) return '-'
  const op = condition.operator || condition.op || ''
  const value = condition.value ?? ''
  const duration = condition.duration ? ` ${t('monitorAlertRules.duration')} ${condition.duration}s` : ''
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
      return { variant: 'destructive' as const, label: t('monitorAlertRules.critical') }
    case 'warning':
      return { variant: 'warning' as const, label: t('monitorAlertRules.warning') }
    case 'info':
      return { variant: 'default' as const, label: t('monitorAlertRules.info') }
    default:
      return { variant: 'secondary' as const, label: l || '-' }
  }
}

// 手动刷新
async function handleRefresh() {
  try {
    await refresh()
    toast(t('monitorAlertRules.dataRefreshed'), 'success')
  } catch {
    toast(t('monitorAlertRules.refreshFailed'), 'destructive')
  }
}
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader :title="t('monitorAlertRules.title')" :description="t('monitorAlertRules.description')">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="handleRefresh">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          {{ t('monitorAlertRules.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <!-- 规则表格 -->
    <Table
      :columns="columns"
      :data="rules || []"
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
          {{ row.enabled ? t('monitorAlertRules.enabledStatus') : t('monitorAlertRules.disabledStatus') }}
        </Badge>
      </template>
    </Table>
  </div>
</template>
