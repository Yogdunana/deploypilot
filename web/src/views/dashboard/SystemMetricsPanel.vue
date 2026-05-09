<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Cpu, MemoryStick, HardDrive, ShieldAlert } from 'lucide-vue-next'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Progress from '@/components/ui/Progress.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import type { SystemMetrics, Alert } from '@/types/models'

const { t } = useI18n()

const props = defineProps<{
  systemMetrics: SystemMetrics | null
  metricsLoading: boolean
  alerts: Alert[]
  alertsLoading: boolean
}>()

function cpuVariant(value: number): 'success' | 'warning' | 'destructive' {
  if (value < 60) return 'success'
  if (value < 85) return 'warning'
  return 'destructive'
}
function memoryVariant(value: number): 'success' | 'warning' | 'destructive' {
  if (value < 70) return 'success'
  if (value < 90) return 'warning'
  return 'destructive'
}
function diskVariant(value: number): 'success' | 'warning' | 'destructive' {
  if (value < 70) return 'success'
  if (value < 90) return 'warning'
  return 'destructive'
}

function alertVariant(level: string): 'destructive' | 'warning' | 'secondary' {
  const l = (level || '').toLowerCase()
  if (l === 'critical' || l === 'error' || l === 'high') return 'destructive'
  if (l === 'warning' || l === 'medium') return 'warning'
  return 'secondary'
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}
</script>

<template>
  <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
    <!-- Left: System Resources (2/3) -->
    <div class="lg:col-span-2">
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <Cpu class="w-4 h-4 text-muted-foreground" />
            <h2 class="text-base font-semibold text-foreground">{{ t('dashboard.systemResources') }}</h2>
          </div>
        </template>

        <div v-if="metricsLoading && !systemMetrics" class="space-y-5 py-2">
          <div class="space-y-2">
            <div class="flex justify-between">
              <Skeleton class="h-4 w-20" variant="text" />
              <Skeleton class="h-4 w-12" variant="text" />
            </div>
            <Skeleton class="h-2 w-full" variant="rectangular" />
          </div>
          <div class="space-y-2">
            <div class="flex justify-between">
              <Skeleton class="h-4 w-24" variant="text" />
              <Skeleton class="h-4 w-20" variant="text" />
            </div>
            <Skeleton class="h-2 w-full" variant="rectangular" />
          </div>
          <div class="space-y-2">
            <div class="flex justify-between">
              <Skeleton class="h-4 w-20" variant="text" />
              <Skeleton class="h-4 w-20" variant="text" />
            </div>
            <Skeleton class="h-2 w-full" variant="rectangular" />
          </div>
        </div>

        <div v-else-if="systemMetrics" class="space-y-5">
          <!-- CPU Usage -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <Cpu class="w-4 h-4 text-muted-foreground" />
                <span class="text-sm font-medium text-foreground">{{ t('dashboard.cpuUsage') }}</span>
              </div>
              <span class="text-sm font-semibold text-foreground">
                {{ systemMetrics.cpu_usage.toFixed(1) }}%
              </span>
            </div>
            <Progress :value="systemMetrics.cpu_usage" :variant="cpuVariant(systemMetrics.cpu_usage)" />
          </div>

          <!-- Memory Usage -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <MemoryStick class="w-4 h-4 text-muted-foreground" />
                <span class="text-sm font-medium text-foreground">{{ t('dashboard.memoryUsage') }}</span>
              </div>
              <span class="text-sm font-semibold text-foreground">
                {{ systemMetrics.memory_usage.toFixed(1) }}%
                <span class="text-muted-foreground font-normal">
                  ({{ formatBytes(systemMetrics.memory_used) }} / {{ formatBytes(systemMetrics.memory_total) }})
                </span>
              </span>
            </div>
            <Progress :value="systemMetrics.memory_usage" :variant="memoryVariant(systemMetrics.memory_usage)" />
          </div>

          <!-- Disk Usage -->
          <div>
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2">
                <HardDrive class="w-4 h-4 text-muted-foreground" />
                <span class="text-sm font-medium text-foreground">{{ t('dashboard.diskUsage') }}</span>
              </div>
              <span class="text-sm font-semibold text-foreground">
                {{ systemMetrics.disk_usage.toFixed(1) }}%
                <span class="text-muted-foreground font-normal">
                  ({{ formatBytes(systemMetrics.disk_used) }} / {{ formatBytes(systemMetrics.disk_total) }})
                </span>
              </span>
            </div>
            <Progress :value="systemMetrics.disk_usage" :variant="diskVariant(systemMetrics.disk_usage)" />
          </div>
        </div>

        <div v-else>
          <p class="text-sm text-muted-foreground text-center py-4">{{ t('dashboard.noSystemData') }}</p>
        </div>
      </Card>
    </div>

    <!-- Right: Active Alerts (1/3) -->
    <div>
      <Card class="h-full">
        <template #header>
          <div class="flex items-center gap-2">
            <ShieldAlert class="w-4 h-4 text-muted-foreground" />
            <h2 class="text-base font-semibold text-foreground">{{ t('dashboard.activeAlerts') }}</h2>
            <Badge v-if="alerts.length > 0" variant="destructive" class="ml-auto">
              {{ alerts.length }}
            </Badge>
          </div>
        </template>

        <div v-if="alertsLoading && alerts.length === 0" class="space-y-3 py-2">
          <div v-for="i in 3" :key="i" class="space-y-2">
            <Skeleton class="h-4 w-3/4" variant="text" />
            <Skeleton class="h-4 w-full" variant="text" />
          </div>
        </div>

        <div v-else-if="alerts.length === 0">
          <EmptyState
            :title="t('dashboard.noAlerts')"
            :description="t('dashboard.noAlertsDesc')"
          />
        </div>

        <div v-else class="space-y-3">
          <div
            v-for="alert in alerts"
            :key="alert.id"
            class="rounded-md border border-border bg-background/50 p-3 space-y-1.5"
          >
            <div class="flex items-center gap-2">
              <Badge :variant="alertVariant(alert.level)">
                {{ alert.level }}
              </Badge>
              <span class="text-sm font-medium text-foreground truncate">
                {{ alert.message }}
              </span>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-xs text-muted-foreground">
                {{ t('dashboard.rule') }} #{{ alert.rule_id }}
              </span>
              <RelativeTime :date="alert.created_at" class="text-xs" />
            </div>
          </div>
        </div>
      </Card>
    </div>
  </div>
</template>
