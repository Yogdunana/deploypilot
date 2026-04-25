<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { Info, HeartPulse, RefreshCw, CheckCircle, XCircle } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Card from '@/components/ui/Card.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Button from '@/components/ui/Button.vue'
import * as systemApi from '@/api/modules/system'

const { toast } = inject<any>('toast')!

// State
const loading = ref(true)
const versionInfo = ref<{ version: string; build_time: string; git_commit: string } | null>(null)
const healthInfo = ref<{ status: string; uptime: number; components: Record<string, string> } | null>(null)
const updateInfo = ref<{ current: string; latest: string; has_update: boolean } | null>(null)
const checkingUpdate = ref(false)

// Fetch all system info
async function fetchSystemInfo() {
  loading.value = true
  try {
    const [versionRes, healthRes] = await Promise.all([
      systemApi.getVersion(),
      systemApi.getHealth(),
    ])
    if (versionRes.data.status === 'success') {
      versionInfo.value = versionRes.data.data
    }
    if (healthRes.data.status === 'success') {
      healthInfo.value = healthRes.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '获取系统信息失败', 'destructive')
  } finally {
    loading.value = false
  }
}

// Check for updates
async function handleCheckUpdate() {
  checkingUpdate.value = true
  try {
    const res = await systemApi.checkUpdate()
    if (res.data.status === 'success') {
      updateInfo.value = res.data.data
      if (res.data.data.has_update) {
        toast(`发现新版本 ${res.data.data.latest}`, 'warning')
      } else {
        toast('当前已是最新版本', 'success')
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '检查更新失败', 'destructive')
  } finally {
    checkingUpdate.value = false
  }
}

// Format uptime
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days} 天 ${hours} 小时 ${minutes} 分钟`
  if (hours > 0) return `${hours} 小时 ${minutes} 分钟`
  return `${minutes} 分钟`
}

onMounted(fetchSystemInfo)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader title="系统信息" />

    <!-- Loading skeleton -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <div v-for="i in 3" :key="i" class="rounded-lg border border-border bg-card p-6 space-y-4">
        <Skeleton class="h-5 w-24" />
        <div class="space-y-3">
          <Skeleton class="h-4 w-full" />
          <Skeleton class="h-4 w-3/4" />
          <Skeleton class="h-4 w-1/2" />
        </div>
      </div>
    </div>

    <!-- Cards grid -->
    <div v-else class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <!-- Version Info -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <Info class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">版本信息</h3>
          </div>
        </template>
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">版本号</span>
            <code class="text-sm font-mono text-foreground">{{ versionInfo?.version || '-' }}</code>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">构建时间</span>
            <span class="text-sm text-foreground">{{ versionInfo?.build_time || '-' }}</span>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">Git Commit</span>
            <code class="text-xs font-mono text-foreground truncate max-w-[140px]">{{ versionInfo?.git_commit || '-' }}</code>
          </div>
        </div>
      </Card>

      <!-- Health Status -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <HeartPulse class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">健康状态</h3>
          </div>
        </template>
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">系统状态</span>
            <div class="flex items-center gap-1.5">
              <CheckCircle v-if="healthInfo?.status === 'healthy'" class="w-4 h-4 text-success" />
              <XCircle v-else class="w-4 h-4 text-destructive" />
              <span class="text-sm text-foreground">{{ healthInfo?.status === 'healthy' ? '正常' : '异常' }}</span>
            </div>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">运行时间</span>
            <span class="text-sm text-foreground">{{ healthInfo ? formatUptime(healthInfo.uptime) : '-' }}</span>
          </div>
          <div v-if="healthInfo?.components" class="space-y-2">
            <div
              v-for="(status, name) in healthInfo.components"
              :key="name"
              class="flex items-center justify-between"
            >
              <span class="text-sm text-muted-foreground capitalize">{{ name }}</span>
              <div class="flex items-center gap-1.5">
                <CheckCircle v-if="status === 'healthy'" class="w-3.5 h-3.5 text-success" />
                <XCircle v-else class="w-3.5 h-3.5 text-destructive" />
                <span class="text-xs text-foreground">{{ status === 'healthy' ? '正常' : '异常' }}</span>
              </div>
            </div>
          </div>
        </div>
      </Card>

      <!-- Update Check -->
      <Card>
        <template #header>
          <div class="flex items-center gap-2">
            <RefreshCw class="w-4 h-4 text-muted-foreground" />
            <h3 class="text-sm font-semibold text-foreground">更新检查</h3>
          </div>
        </template>
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">当前版本</span>
            <code class="text-sm font-mono text-foreground">{{ versionInfo?.version || '-' }}</code>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">最新版本</span>
            <code class="text-sm font-mono text-foreground">{{ updateInfo?.latest || '-' }}</code>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-sm text-muted-foreground">状态</span>
            <span v-if="updateInfo" class="text-sm" :class="updateInfo.has_update ? 'text-warning' : 'text-success'">
              {{ updateInfo.has_update ? '有新版本可用' : '已是最新' }}
            </span>
            <span v-else class="text-sm text-muted-foreground">未检查</span>
          </div>
          <Button
            variant="outline"
            size="sm"
            class="w-full mt-2"
            :loading="checkingUpdate"
            @click="handleCheckUpdate"
          >
            <template #icon><RefreshCw class="w-3.5 h-3.5" /></template>
            检查更新
          </Button>
        </div>
      </Card>
    </div>
  </div>
</template>
