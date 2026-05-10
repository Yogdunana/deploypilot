<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { RefreshCw, HeartPulse, Wrench } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Table from '@/components/ui/Table.vue'
import Progress from '@/components/ui/Progress.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as appsApi from '@/api/modules/apps'
import * as monitorApi from '@/api/modules/monitor'
import type { App, ContainerMetrics } from '@/types/models'
import { useI18n } from 'vue-i18n'

const { toast } = useToast()
const { t } = useI18n()

const loading = ref(true)
const apps = ref<App[]>([])
const containerMetricsMap = ref<Record<string, ContainerMetrics>>({})

// 自愈对话框
const healDialogOpen = ref(false)
const healingName = ref('')
const healing = ref(false)

// 健康检查中
const checkingName = ref<string | null>(null)

// 表格列
const columns = computed(() => [
  { key: 'container_name', label: t('monitorContainers.name') },
  { key: 'app_name', label: t('monitorContainers.appName') },
  { key: 'cpu_usage', label: t('monitorContainers.cpu') },
  { key: 'memory_usage', label: t('monitorContainers.memory') },
  { key: 'status', label: t('monitorContainers.status') },
  { key: 'actions', label: t('monitorContainers.actions'), width: '180px' },
])

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

// 获取容器指标数据（用于表格展示）
const tableData = computed(() => apps.value.map((app) => {
  const metrics = containerMetricsMap.value[app.container_name || app.name]
  return {
    container_name: app.container_name || app.name,
    app_name: app.name,
    cpu_usage: metrics?.cpu_usage ?? 0,
    memory_usage: metrics?.memory_usage ?? 0,
    memory_limit: metrics?.memory_limit ?? 0,
    status: metrics?.status || app.status,
  }
}))

// 获取应用列表
async function fetchApps() {
  loading.value = true
  try {
    const res = await appsApi.list()
    if (res.data.status === 'success') {
      apps.value = res.data.data || []
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('monitorContainers.fetchAppsFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// 获取所有容器的指标
async function fetchAllMetrics() {
  for (const app of apps.value) {
    const name = app.container_name || app.name
    try {
      const res = await monitorApi.getContainerMetrics(name)
      if (res.data.status === 'success') {
        containerMetricsMap.value[name] = res.data.data
      }
    } catch {
      // 单个容器指标获取失败不影响其他
    }
  }
}

// 健康检查
async function handleCheck(name: string) {
  checkingName.value = name
  try {
    const res = await monitorApi.check(name)
    if (res.data.status === 'success') {
      const result = res.data.data
      if (result.healthy) {
        toast(t('monitorContainers.healthCheckPassed', { name }), 'success')
      } else {
        toast(t('monitorContainers.healthCheckFailed', { name, message: result.message }), 'destructive')
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('monitorContainers.healthCheckFailedGeneric'), 'destructive')
  } finally {
    checkingName.value = null
  }
}

// 打开自愈对话框
function openHealDialog(name: string) {
  healingName.value = name
  healDialogOpen.value = true
}

// 确认自愈
async function confirmHeal() {
  healing.value = true
  try {
    await monitorApi.heal(healingName.value)
    toast(t('monitorContainers.healTriggered', { name: healingName.value }), 'success')
    // 刷新指标
    fetchAllMetrics()
  } catch (err: any) {
    toast(err.response?.data?.message || t('monitorContainers.healFailed'), 'destructive')
  } finally {
    healing.value = false
  }
}

// 刷新
async function handleRefresh() {
  await fetchApps()
  await fetchAllMetrics()
  toast(t('monitorContainers.dataRefreshed'), 'success')
}

onMounted(async () => {
  await fetchApps()
  await fetchAllMetrics()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader :title="t('monitorContainers.title')" :description="t('monitorContainers.description')">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="handleRefresh">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          {{ t('monitorContainers.refresh') }}
        </Button>
      </template>
    </PageHeader>

    <!-- 容器列表 -->
    <Table
      :columns="columns"
      :data="tableData"
      :loading="loading"
    >
      <template #cell-container_name="{ row }">
        <span class="font-mono text-sm text-foreground">{{ row.container_name }}</span>
      </template>
      <template #cell-app_name="{ row }">
        <span class="text-sm text-foreground">{{ row.app_name }}</span>
      </template>
      <template #cell-cpu_usage="{ row }">
        <div class="flex items-center gap-2 min-w-[120px]">
          <Progress
            :value="row.cpu_usage"
            :variant="getVariant(row.cpu_usage)"
            class="flex-1"
          />
          <span class="text-xs text-muted-foreground w-12 text-right">
            {{ row.cpu_usage.toFixed(1) }}%
          </span>
        </div>
      </template>
      <template #cell-memory_usage="{ row }">
        <div class="flex items-center gap-2 min-w-[120px]">
          <Progress
            :value="row.memory_usage"
            :variant="getVariant(row.memory_usage)"
            class="flex-1"
          />
          <span class="text-xs text-muted-foreground w-12 text-right">
            {{ row.memory_usage.toFixed(1) }}%
          </span>
        </div>
      </template>
      <template #cell-status="{ row }">
        <StatusBadge :status="row.status" />
      </template>
      <template #cell-actions="{ row }">
        <div class="flex items-center gap-1">
          <Button
            variant="ghost"
            size="sm"
            class="h-7 text-xs"
            :loading="checkingName === row.container_name"
            @click="handleCheck(row.container_name)"
          >
            <template #icon><HeartPulse class="w-3.5 h-3.5" /></template>
            {{ t('monitorContainers.healthCheck') }}
          </Button>
          <Button
            variant="ghost"
            size="sm"
            class="h-7 text-xs"
            @click="openHealDialog(row.container_name)"
          >
            <template #icon><Wrench class="w-3.5 h-3.5" /></template>
            {{ t('monitorContainers.heal') }}
          </Button>
        </div>
      </template>
    </Table>

    <!-- 自愈确认对话框 -->
    <AlertDialog
      v-model:open="healDialogOpen"
      :title="t('monitorContainers.healConfirm')"
      :description="t('monitorContainers.healConfirmDesc', { name: healingName })"
      :confirm-text="t('monitorContainers.confirmHeal')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmHeal"
    />
  </div>
</template>
