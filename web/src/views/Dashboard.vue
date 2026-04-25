<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, computed } from 'vue'
import { useRouter } from 'vue-router'
import * as appsApi from '@/api/modules/apps'
import * as serversApi from '@/api/modules/servers'
import * as deploymentsApi from '@/api/modules/deployments'
import * as monitorApi from '@/api/modules/monitor'
import type { App, Server, DeploymentRecord, SystemMetrics, Alert } from '@/types/models'

import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import Progress from '@/components/ui/Progress.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Table from '@/components/ui/Table.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'

import {
  Layers,
  CirclePlay,
  Server as ServerIcon,
  Rocket,
  Plus,
  ServerCog,
  LayoutTemplate,
  ShieldAlert,
  Cpu,
  MemoryStick,
  HardDrive,
  ArrowRight,
} from 'lucide-vue-next'

const router = useRouter()

// Data state
const apps = ref<App[]>([])
const servers = ref<Server[]>([])
const deployments = ref<DeploymentRecord[]>([])
const recentDeployments = ref<DeploymentRecord[]>([])
const systemMetrics = ref<SystemMetrics | null>(null)
const alerts = ref<Alert[]>([])

// Loading state
const loading = ref(true)
const metricsLoading = ref(true)
const alertsLoading = ref(true)
const error = ref('')

// Computed metrics
const totalApps = computed(() => apps.value.length)
const runningApps = computed(() => apps.value.filter(a => a.status === 'running').length)
const totalServers = computed(() => servers.value.length)
const recentDeployCount = computed(() => recentDeployments.value.length)

// CPU variant
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

// Alert severity variant
function alertVariant(level: string): 'destructive' | 'warning' | 'secondary' {
  const l = (level || '').toLowerCase()
  if (l === 'critical' || l === 'error' || l === 'high') return 'destructive'
  if (l === 'warning' || l === 'medium') return 'warning'
  return 'secondary'
}

// Format bytes
function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

// Table columns
const deploymentColumns = [
  { key: 'app_name', label: '应用名' },
  { key: 'status', label: '状态', width: '120px' },
  { key: 'created_at', label: '时间', width: '160px' },
]

// Auto refresh timer
let metricsTimer: ReturnType<typeof setInterval> | null = null

async function loadDashboardData() {
  loading.value = true
  error.value = ''
  try {
    const [appsRes, serversRes, deploymentsRes] = await Promise.allSettled([
      appsApi.list(),
      serversApi.list(),
      deploymentsApi.list(undefined, undefined, { page: 1, page_size: 10 }),
    ])

    if (appsRes.status === 'fulfilled') {
      apps.value = appsRes.value.data.data || []
    }
    if (serversRes.status === 'fulfilled') {
      servers.value = serversRes.value.data.data || []
    }
    if (deploymentsRes.status === 'fulfilled') {
      const items = deploymentsRes.value.data.data || []
      deployments.value = items
      recentDeployments.value = items.slice(0, 10)
    }
  } catch (err: any) {
    error.value = '加载仪表盘数据失败，请刷新重试'
  } finally {
    loading.value = false
  }
}

async function loadMetrics() {
  metricsLoading.value = true
  try {
    const res = await monitorApi.getSystemMetrics()
    systemMetrics.value = res.data.data
  } catch {
    // silently fail for metrics
  } finally {
    metricsLoading.value = false
  }
}

async function loadAlerts() {
  alertsLoading.value = true
  try {
    const res = await monitorApi.listAlerts({ page: 1, page_size: 10 })
    alerts.value = (res.data.data || []).filter(
      (a: Alert) => !a.resolved
    )
  } catch {
    // silently fail for alerts
  } finally {
    alertsLoading.value = false
  }
}

function goToDeployment(row: DeploymentRecord) {
  router.push(`/deployments/${row.id}`)
}

function goToCreateApp() {
  router.push('/apps/create')
}

function goToCreateServer() {
  router.push('/servers/create')
}

function goToTemplates() {
  router.push('/templates')
}

onMounted(async () => {
  await loadDashboardData()
  loadMetrics()
  loadAlerts()

  // Auto-refresh metrics every 30 seconds
  metricsTimer = setInterval(() => {
    loadMetrics()
    loadAlerts()
  }, 30000)
})

onBeforeUnmount(() => {
  if (metricsTimer) {
    clearInterval(metricsTimer)
    metricsTimer = null
  }
})
</script>

<template>
  <div class="p-6 space-y-6">
    <!-- Page Header -->
    <PageHeader title="仪表盘" description="系统概览与快速操作">
      <template #actions>
        <Button variant="outline" size="sm" @click="loadDashboardData">
          刷新
        </Button>
      </template>
    </PageHeader>

    <!-- Error Banner -->
    <div v-if="error" class="rounded-lg border border-destructive/50 bg-destructive/5 p-4 text-sm text-destructive">
      {{ error }}
    </div>

    <!-- Metrics Cards Row -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      <!-- Total Apps -->
      <div
        class="rounded-lg border border-border bg-card p-5 shadow-sm hover:shadow-md transition-shadow"
        style="border-left: 3px solid hsl(var(--primary))"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-[13px] text-muted-foreground">应用总数</p>
            <p v-if="loading" class="mt-1">
              <Skeleton class="h-8 w-16" variant="text" />
            </p>
            <p v-else class="text-[28px] font-bold text-foreground leading-tight mt-1">
              {{ totalApps }}
            </p>
          </div>
          <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-primary/10">
            <Layers class="w-5 h-5 text-primary" />
          </div>
        </div>
      </div>

      <!-- Running Apps -->
      <div
        class="rounded-lg border border-border bg-card p-5 shadow-sm hover:shadow-md transition-shadow"
        style="border-left: 3px solid hsl(var(--success))"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-[13px] text-muted-foreground">运行中应用</p>
            <p v-if="loading" class="mt-1">
              <Skeleton class="h-8 w-16" variant="text" />
            </p>
            <p v-else class="text-[28px] font-bold text-success leading-tight mt-1">
              {{ runningApps }}
            </p>
          </div>
          <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-success/10">
            <CirclePlay class="w-5 h-5 text-success" />
          </div>
        </div>
      </div>

      <!-- Total Servers -->
      <div
        class="rounded-lg border border-border bg-card p-5 shadow-sm hover:shadow-md transition-shadow"
        style="border-left: 3px solid hsl(var(--warning))"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-[13px] text-muted-foreground">服务器总数</p>
            <p v-if="loading" class="mt-1">
              <Skeleton class="h-8 w-16" variant="text" />
            </p>
            <p v-else class="text-[28px] font-bold text-foreground leading-tight mt-1">
              {{ totalServers }}
            </p>
          </div>
          <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-warning/10">
            <ServerIcon class="w-5 h-5 text-warning" />
          </div>
        </div>
      </div>

      <!-- Recent Deployments -->
      <div
        class="rounded-lg border border-border bg-card p-5 shadow-sm hover:shadow-md transition-shadow"
        style="border-left: 3px solid hsl(221, 83%, 53%)"
      >
        <div class="flex items-center justify-between">
          <div>
            <p class="text-[13px] text-muted-foreground">最近部署</p>
            <p v-if="loading" class="mt-1">
              <Skeleton class="h-8 w-16" variant="text" />
            </p>
            <p v-else class="text-[28px] font-bold text-foreground leading-tight mt-1">
              {{ recentDeployCount }}
            </p>
          </div>
          <div class="flex items-center justify-center w-10 h-10 rounded-lg bg-blue-500/10">
            <Rocket class="w-5 h-5 text-blue-500" />
          </div>
        </div>
      </div>
    </div>

    <!-- Two Column Layout: System Resources + Active Alerts -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Left: System Resources (2/3) -->
      <div class="lg:col-span-2">
        <Card>
          <template #header>
            <div class="flex items-center gap-2">
              <Cpu class="w-4 h-4 text-muted-foreground" />
              <h2 class="text-base font-semibold text-foreground">系统资源</h2>
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
                  <span class="text-sm font-medium text-foreground">CPU 使用率</span>
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
                  <span class="text-sm font-medium text-foreground">内存使用率</span>
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
                  <span class="text-sm font-medium text-foreground">磁盘使用率</span>
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
            <p class="text-sm text-muted-foreground text-center py-4">暂无系统资源数据</p>
          </div>
        </Card>
      </div>

      <!-- Right: Active Alerts (1/3) -->
      <div>
        <Card class="h-full">
          <template #header>
            <div class="flex items-center gap-2">
              <ShieldAlert class="w-4 h-4 text-muted-foreground" />
              <h2 class="text-base font-semibold text-foreground">活跃告警</h2>
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
              title="系统运行正常"
              description="当前没有活跃的告警"
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
                  规则 #{{ alert.rule_id }}
                </span>
                <RelativeTime :date="alert.created_at" class="text-xs" />
              </div>
            </div>
          </div>
        </Card>
      </div>
    </div>

    <!-- Recent Deployments Table -->
    <Card>
      <template #header>
        <div class="flex items-center justify-between">
          <h2 class="text-base font-semibold text-foreground">最近部署</h2>
          <Button variant="ghost" size="sm" @click="router.push('/deployments')">
            查看全部
            <template #icon>
              <ArrowRight class="w-4 h-4" />
            </template>
          </Button>
        </div>
      </template>

      <Table
        :columns="deploymentColumns"
        :data="recentDeployments"
        :loading="loading"
      >
        <template #cell-app_name="{ row }">
          <span class="font-medium text-foreground">{{ row.app_name }}</span>
        </template>
        <template #cell-status="{ row }">
          <StatusBadge :status="row.status" />
        </template>
        <template #cell-created_at="{ row }">
          <RelativeTime :date="row.created_at" />
        </template>
      </Table>
    </Card>

    <!-- Quick Actions -->
    <Card>
      <template #header>
        <h2 class="text-base font-semibold text-foreground">快速操作</h2>
      </template>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <Button
          variant="outline"
          class="h-auto py-4 justify-start gap-3"
          @click="goToCreateApp"
        >
          <div class="flex items-center justify-center w-9 h-9 rounded-lg bg-primary/10 shrink-0">
            <Plus class="w-4 h-4 text-primary" />
          </div>
          <div class="text-left">
            <p class="text-sm font-medium text-foreground">创建应用</p>
            <p class="text-xs text-muted-foreground">部署一个新的应用</p>
          </div>
        </Button>

        <Button
          variant="outline"
          class="h-auto py-4 justify-start gap-3"
          @click="goToCreateServer"
        >
          <div class="flex items-center justify-center w-9 h-9 rounded-lg bg-warning/10 shrink-0">
            <ServerCog class="w-4 h-4 text-warning" />
          </div>
          <div class="text-left">
            <p class="text-sm font-medium text-foreground">添加服务器</p>
            <p class="text-xs text-muted-foreground">注册新的部署服务器</p>
          </div>
        </Button>

        <Button
          variant="outline"
          class="h-auto py-4 justify-start gap-3"
          @click="goToTemplates"
        >
          <div class="flex items-center justify-center w-9 h-9 rounded-lg bg-success/10 shrink-0">
            <LayoutTemplate class="w-4 h-4 text-success" />
          </div>
          <div class="text-left">
            <p class="text-sm font-medium text-foreground">浏览模板</p>
            <p class="text-xs text-muted-foreground">从模板快速创建应用</p>
          </div>
        </Button>
      </div>
    </Card>
  </div>
</template>
