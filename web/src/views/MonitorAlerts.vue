<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { RefreshCw } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Badge from '@/components/ui/Badge.vue'
import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { usePolling } from '@/composables/usePolling'
import * as monitorApi from '@/api/modules/monitor'
import type { Alert } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = inject<any>('toast')!
const { t } = useI18n()

// 轮询告警列表，每 30 秒刷新
const { data: alerts, loading, refresh } = usePolling<Alert[]>({
  fetchFn: async () => {
    const res = await monitorApi.listAlerts()
    if (res.data.status === 'success') {
      return res.data.data || []
    }
    return []
  },
  interval: 30000,
  autoStart: true,
})

// 告警严重程度样式映射
function getLevelBadge(level: string) {
  const l = (level || '').toLowerCase()
  switch (l) {
    case 'critical':
      return { variant: 'destructive' as const, label: t('monitorAlerts.critical') }
    case 'warning':
      return { variant: 'warning' as const, label: t('monitorAlerts.warning') }
    case 'info':
      return { variant: 'default' as const, label: t('monitorAlerts.info') }
    default:
      return { variant: 'secondary' as const, label: l }
  }
}

// 告警严重程度左边框颜色
function getLevelBorderColor(level: string): string {
  const l = (level || '').toLowerCase()
  switch (l) {
    case 'critical':
      return 'border-l-destructive'
    case 'warning':
      return 'border-l-warning'
    case 'info':
      return 'border-l-primary'
    default:
      return 'border-l-border'
  }
}

// 手动刷新
async function handleRefresh() {
  try {
    await refresh()
    toast(t('monitorAlerts.dataRefreshed'), 'success')
  } catch {
    toast(t('monitorAlerts.refreshFailed'), 'destructive')
  }
}
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader :title="t('monitorAlerts.title')" :description="t('monitorAlerts.description')">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="handleRefresh">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          {{ t('monitorAlerts.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <!-- 加载状态 -->
    <div v-if="loading && !alerts.value" class="space-y-3">
      <Skeleton v-for="i in 3" :key="i" class="h-24" />
    </div>

    <!-- 空状态 -->
    <EmptyState
      v-else-if="alerts.value && alerts.value.length === 0"
      :title="t('monitorAlerts.noAlerts')"
      :description="t('monitorAlerts.noAlertsDesc')"
    />

    <!-- 告警列表 -->
    <div v-else-if="alerts.value && alerts.value.length > 0" class="space-y-3">
      <Card
        v-for="alert in alerts.value"
        :key="alert.id"
        class="border-l-4"
        :class="getLevelBorderColor(alert.level)"
      >
        <div class="flex items-start justify-between gap-4">
          <div class="flex-1 min-w-0 space-y-1">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-foreground">
{{ t('monitorAlerts.alertRule') }} #{{ alert.rule_id }}
              </span>
              <Badge :variant="getLevelBadge(alert.level).variant">
                {{ getLevelBadge(alert.level).label }}
              </Badge>
            </div>
            <p class="text-sm text-muted-foreground">
              {{ alert.message }}
            </p>
            <p class="text-xs text-muted-foreground">
{{ t('monitorAlerts.triggerTime') }}: <RelativeTime :date="alert.created_at" />
            </p>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
