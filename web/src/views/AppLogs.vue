<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Radio, History, Trash2 } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import LogViewer from '@/components/common/LogViewer.vue'
import type { LogEntry } from '@/components/common/LogViewer.vue'
import Button from '@/components/ui/Button.vue'
import Switch from '@/components/ui/Switch.vue'
import * as appsApi from '@/api/modules/apps'
import { useWebSocket } from '@/composables/useWebSocket'

const props = defineProps<{ id: string }>()
const router = useRouter()
const { toast } = inject<any>('toast')!

const appName = ref('')
const realtimeEnabled = ref(false)
const logs = ref<LogEntry[]>([])
const loadingHistory = ref(false)

// WebSocket 连接（不自动连接，由开关控制）
const { connected, send, disconnect, connect } = useWebSocket({
  path: `/ws/logs/${props.id}`,
  autoConnect: false,
  onMessage(data: any) {
    if (data.type === 'log') {
      logs.value.push({
        timestamp: data.timestamp || new Date().toISOString(),
        data: data.data || '',
      })
    }
  },
})

// 切换实时日志
function toggleRealtime() {
  if (realtimeEnabled.value) {
    connect()
  } else {
    disconnect()
  }
}

// 加载历史日志
async function loadHistory() {
  loadingHistory.value = true
  try {
    const res = await appsApi.getLogs(Number(props.id), 500)
    if (res.data.status === 'success') {
      const logText = res.data.data
      if (logText && typeof logText === 'string') {
        const lines = logText.split('\n').filter((line) => line.trim())
        const historyLogs: LogEntry[] = lines.map((line) => ({
          timestamp: new Date().toISOString(),
          data: line,
        }))
        logs.value = [...historyLogs, ...logs.value]
        toast(`已加载 ${historyLogs.length} 条历史日志`, 'success')
      }
    }
  } catch (err: any) {
    toast(err.response?.data?.message || '加载历史日志失败', 'destructive')
  } finally {
    loadingHistory.value = false
  }
}

// 清空日志
function clearLogs() {
  logs.value = []
}

// 获取应用信息
async function fetchApp() {
  try {
    const res = await appsApi.get(Number(props.id))
    if (res.data.status === 'success') {
      appName.value = res.data.data.name
    }
  } catch {
    // 静默处理
  }
}

onMounted(() => {
  fetchApp()
})
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- 页面头部 -->
    <PageHeader>
      <template #default>
        <div class="flex items-center gap-3">
          <Button variant="ghost" size="icon" @click="router.push('/apps')">
            <ArrowLeft class="w-4 h-4" />
          </Button>
          <div>
            <div class="flex items-center gap-2">
              <h1 class="text-xl font-semibold text-foreground">
                应用日志 - {{ appName || '加载中...' }}
              </h1>
              <span
                class="inline-block w-2 h-2 rounded-full"
                :class="connected ? 'bg-green-500 shadow-[0_0_6px_rgba(34,197,94,0.5)]' : 'bg-red-500'"
              />
            </div>
            <p class="mt-0.5 text-sm text-muted-foreground">
              {{ connected ? '实时日志已连接' : '实时日志未连接' }}
            </p>
          </div>
        </div>
      </template>
      <template #actions>
        <!-- 实时日志开关 -->
        <div class="flex items-center gap-2">
          <Radio class="w-4 h-4 text-muted-foreground" />
          <span class="text-sm text-muted-foreground">实时日志</span>
          <Switch v-model="realtimeEnabled" @update:model-value="toggleRealtime" />
        </div>
      </template>
    </PageHeader>

    <!-- 工具栏 -->
    <div class="flex items-center gap-2">
      <Button variant="outline" size="sm" :loading="loadingHistory" @click="loadHistory">
        <template #icon><History class="w-4 h-4" /></template>
        加载历史日志
      </Button>
      <Button variant="outline" size="sm" @click="clearLogs">
        <template #icon><Trash2 class="w-4 h-4" /></template>
        清空
      </Button>
    </div>

    <!-- 日志查看器 -->
    <div class="h-[calc(100vh-220px)] min-h-[400px]">
      <LogViewer
        :logs="logs"
        :connected="connected"
        :auto-scroll="true"
        @clear="clearLogs"
      />
    </div>
  </div>
</template>
