<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import * as appsApi from '@/api/modules/apps'
import * as serversApi from '@/api/modules/servers'
import * as deploymentsApi from '@/api/modules/deployments'
import * as monitorApi from '@/api/modules/monitor'
import type { App, Server, DeploymentRecord, SystemMetrics, Alert } from '@/types/models'

import Card from '@/components/ui/Card.vue'
import Button from '@/components/ui/Button.vue'
import PageHeader from '@/components/common/PageHeader.vue'

import {
  Plus,
  ServerCog,
  LayoutTemplate,
} from 'lucide-vue-next'

import AppSummaryCard from './AppSummaryCard.vue'
import ServerSummaryCard from './ServerSummaryCard.vue'
import RecentDeployments from './RecentDeployments.vue'
import SystemMetricsPanel from './SystemMetricsPanel.vue'

const router = useRouter()
const { t } = useI18n()

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
    error.value = t('dashboard.loadFailed')
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
  <div class="space-y-6">
    <!-- Page Header -->
    <PageHeader :title="t('dashboard.title')" :description="t('dashboard.description')">
      <template #actions>
        <Button variant="outline" size="sm" @click="loadDashboardData">
          {{ t('dashboard.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Error Banner -->
    <div v-if="error" class="rounded-lg border border-destructive/50 bg-destructive/5 p-4 text-sm text-destructive">
      {{ error }}
    </div>

    <!-- Metrics Cards Row -->
    <AppSummaryCard
      :apps="apps"
      :servers="servers"
      :recent-deployments="recentDeployments"
      :loading="loading"
    />

    <!-- Two Column Layout: System Resources + Active Alerts -->
    <SystemMetricsPanel
      :system-metrics="systemMetrics"
      :metrics-loading="metricsLoading"
      :alerts="alerts"
      :alerts-loading="alertsLoading"
    />

    <!-- Recent Deployments Table -->
    <RecentDeployments
      :recent-deployments="recentDeployments"
      :loading="loading"
    />

    <!-- Quick Actions -->
    <Card>
      <template #header>
        <h2 class="text-base font-semibold text-foreground">{{ t('dashboard.quickActions') }}</h2>
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
            <p class="text-sm font-medium text-foreground">{{ t('dashboard.createApp') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('dashboard.createAppDesc') }}</p>
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
            <p class="text-sm font-medium text-foreground">{{ t('dashboard.addServer') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('dashboard.addServerDesc') }}</p>
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
            <p class="text-sm font-medium text-foreground">{{ t('dashboard.browseTemplates') }}</p>
            <p class="text-xs text-muted-foreground">{{ t('dashboard.browseTemplatesDesc') }}</p>
          </div>
        </Button>
      </div>
    </Card>
  </div>
</template>
