<script setup lang="ts">
import { computed, inject } from 'vue'
import { RefreshCw, Cpu, MemoryStick, HardDrive, ArrowUp, ArrowDown, Clock } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Progress from '@/components/ui/Progress.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import { usePolling } from '@/composables/usePolling'
import * as monitorApi from '@/api/modules/monitor'
import type { SystemMetrics } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = inject<any>('toast')!
const { t } = useI18n()

// 轮询系统指标，每 30 秒刷新
const { data: metrics, loading, error, refresh } = usePolling<SystemMetrics>({
  fetchFn: async () => {
    const res = await monitorApi.getSystemMetrics()
    if (res.data.status === 'success') {
      return res.data.data
    }
    throw new Error(t('monitor.loadMetricsFailed'))
  },
  interval: 30000,
  autoStart: true,
})

// 根据使用率获取进度条颜色
function getVariant(usage: number): 'success' | 'warning' | 'destructive' {
  if (usage < 60) return 'success'
  if (usage <= 80) return 'warning'
  return 'destructive'
}

// 格式化字节数
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

// 格式化网络流量
function formatNetwork(bytes: number): string {
  if (bytes < 1024) return `${bytes.toFixed(0)} B/s`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB/s`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB/s`
}

// 格式化运行时间
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days} ${t('monitor.days')} ${hours} ${t('monitor.hours')} ${minutes} ${t('monitor.minutes')}`
  if (hours > 0) return `${hours} ${t('monitor.hours')} ${minutes} ${t('monitor.minutes')}`
  return `${minutes} ${t('monitor.minutes')}`
}

// CPU 指标
const cpuMetrics = computed(() => {
  if (!metrics.value) return { usage: 0, label: '0%', detail: '-' }
  return {
    usage: metrics.value.cpu_usage,
    label: `${metrics.value.cpu_usage.toFixed(1)}%`,
    detail: t('monitor.cpuUsage'),
  }
})

// 内存指标
const memoryMetrics = computed(() => {
  if (!metrics.value) return { usage: 0, label: '0%', detail: '- / -' }
  return {
    usage: metrics.value.memory_usage,
    label: `${metrics.value.memory_usage.toFixed(1)}%`,
    detail: `${formatBytes(metrics.value.memory_used)} / ${formatBytes(metrics.value.memory_total)}`,
  }
})

// 磁盘指标
const diskMetrics = computed(() => {
  if (!metrics.value) return { usage: 0, label: '0%', detail: '- / -' }
  return {
    usage: metrics.value.disk_usage,
    label: `${metrics.value.disk_usage.toFixed(1)}%`,
    detail: `${formatBytes(metrics.value.disk_used)} / ${formatBytes(metrics.value.disk_total)}`,
  }
})

// 手动刷新
async function handleRefresh() {
  try {
    await refresh()
    toast(t('monitor.dataRefreshed'), 'success')
  } catch {
    toast(t('monitor.refreshFailed'), 'destructive')
  }
}
</script>

<template>
  <div class="p-6 space-y-6">
    <!-- 页面头部 -->
    <PageHeader :title="t('monitor.title')" :description="t('monitor.description')">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="handleRefresh">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          {{ t('monitor.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <!-- 加载状态 -->
    <div v-if="loading && !metrics" class="space-y-4">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Skeleton v-for="i in 3" :key="i" class="h-36" />
      </div>
      <Skeleton class="h-32" />
    </div>

    <!-- 错误状态 -->
    <Card v-else-if="error" class="p-8">
      <div class="flex flex-col items-center justify-center text-center">
        <p class="text-sm text-destructive">{{ t('monitor.loadMetricsFailed') }}</p>
        <p class="mt-1 text-xs text-muted-foreground">{{ error.message }}</p>
      </div>
    </Card>

    <template v-else-if="metrics">
      <!-- 三个指标卡片 -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <!-- CPU -->
        <Card>
          <template #header>
            <div class="flex items-center gap-2">
              <Cpu class="w-4 h-4 text-muted-foreground" />
              <h3 class="text-sm font-medium text-foreground">{{ t('monitor.cpuUsage') }}</h3>
            </div>
          </template>
          <div class="space-y-3">
            <div class="flex items-baseline gap-2">
              <span class="text-3xl font-bold" :class="{
                'text-success': cpuMetrics.usage < 60,
                'text-warning': cpuMetrics.usage >= 60 && cpuMetrics.usage <= 80,
                'text-destructive': cpuMetrics.usage > 80,
              }">
                {{ cpuMetrics.label }}
              </span>
            </div>
            <Progress :value="cpuMetrics.usage" :variant="getVariant(cpuMetrics.usage)" />
            <p class="text-xs text-muted-foreground">{{ cpuMetrics.detail }}</p>
          </div>
        </Card>

        <!-- 内存 -->
        <Card>
          <template #header>
            <div class="flex items-center gap-2">
              <MemoryStick class="w-4 h-4 text-muted-foreground" />
              <h3 class="text-sm font-medium text-foreground">{{ t('monitor.memoryUsage') }}</h3>
            </div>
          </template>
          <div class="space-y-3">
            <div class="flex items-baseline gap-2">
              <span class="text-3xl font-bold" :class="{
                'text-success': memoryMetrics.usage < 60,
                'text-warning': memoryMetrics.usage >= 60 && memoryMetrics.usage <= 80,
                'text-destructive': memoryMetrics.usage > 80,
              }">
                {{ memoryMetrics.label }}
              </span>
            </div>
            <Progress :value="memoryMetrics.usage" :variant="getVariant(memoryMetrics.usage)" />
            <p class="text-xs text-muted-foreground">{{ memoryMetrics.detail }}</p>
          </div>
        </Card>

        <!-- 磁盘 -->
        <Card>
          <template #header>
            <div class="flex items-center gap-2">
              <HardDrive class="w-4 h-4 text-muted-foreground" />
              <h3 class="text-sm font-medium text-foreground">{{ t('monitor.diskUsage') }}</h3>
            </div>
          </template>
          <div class="space-y-3">
            <div class="flex items-baseline gap-2">
              <span class="text-3xl font-bold" :class="{
                'text-success': diskMetrics.usage < 60,
                'text-warning': diskMetrics.usage >= 60 && diskMetrics.usage <= 80,
                'text-destructive': diskMetrics.usage > 80,
              }">
                {{ diskMetrics.label }}
              </span>
            </div>
            <Progress :value="diskMetrics.usage" :variant="getVariant(diskMetrics.usage)" />
            <p class="text-xs text-muted-foreground">{{ diskMetrics.detail }}</p>
          </div>
        </Card>
      </div>

      <!-- 网络和运行时间 -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <!-- 网络入站 -->
        <Card>
          <template #header>
            <div class="flex items-center gap-2">
              <ArrowDown class="w-4 h-4 text-success" />
              <h3 class="text-sm font-medium text-foreground">{{ t('monitor.networkIn') }}</h3>
            </div>
          </template>
          <p class="text-2xl font-bold text-foreground">
            {{ formatNetwork(metrics.value!.network_in) }}
          </p>
        </Card>

        <!-- 网络出站 -->
        <Card>
          <template #header>
            <div class="flex items-center gap-2">
              <ArrowUp class="w-4 h-4 text-primary" />
              <h3 class="text-sm font-medium text-foreground">{{ t('monitor.networkOut') }}</h3>
            </div>
          </template>
          <p class="text-2xl font-bold text-foreground">
            {{ formatNetwork(metrics.value!.network_out) }}
          </p>
        </Card>

        <!-- 运行时间 -->
        <Card>
          <template #header>
            <div class="flex items-center gap-2">
              <Clock class="w-4 h-4 text-muted-foreground" />
              <h3 class="text-sm font-medium text-foreground">{{ t('monitor.systemUptime') }}</h3>
            </div>
          </template>
          <p class="text-2xl font-bold text-foreground">
            {{ formatUptime(metrics.value!.uptime) }}
          </p>
        </Card>
      </div>
    </template>
  </div>
</template>

