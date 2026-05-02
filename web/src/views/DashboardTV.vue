<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useMonitorStore } from '@/stores/monitor'
import MonitorHeader from '@/components/monitor/MonitorHeader.vue'
import ServerStatusGrid from '@/components/monitor/ServerStatusGrid.vue'
import DeploymentFeed from '@/components/monitor/DeploymentFeed.vue'
import AlertSummaryCard from '@/components/monitor/AlertSummaryCard.vue'
import GaugeChart from '@/components/charts/GaugeChart.vue'
import PieChart from '@/components/charts/PieChart.vue'

const store = useMonitorStore()

const isDark = ref(true)
const isFullscreen = ref(false)
const autoRotate = ref(true)
const currentView = ref(0)
const views = ['overview', 'charts', 'servers']
let rotateTimer: ReturnType<typeof setInterval> | null = null
let ws: WebSocket | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null

const uptimeDistData = ref<Array<{ name: string; value: number; color?: string }>>([])

function connectWS() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  ws = new WebSocket(`${protocol}//${window.location.host}/ws/monitor`)
  ws.onopen = () => { store.connected = true }
  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'monitor_check' && msg.data?.results) {
        store.monitors = msg.data.results
        updateChartData()
      } else if (msg.type === 'monitor_update' && msg.data) {
        const idx = store.monitors.findIndex((m: any) => m.id === msg.data.id)
        if (idx >= 0) store.monitors[idx] = { ...store.monitors[idx], ...msg.data }
        else store.monitors.push(msg.data)
      } else if (msg.type === 'full_sync') {
        store.monitors = msg.data || []
        updateChartData()
      } else if (msg.type === 'heartbeat_timeout') {
        store.fetchHeartbeats()
      }
    } catch { /* ignore */ }
  }
  ws.onclose = () => {
    store.connected = false
    setTimeout(connectWS, 5000)
  }
  ws.onerror = () => { ws?.close() }
}

function updateChartData() {
  const up = store.monitors.filter((m: any) => m.status === 'up').length
  const down = store.monitors.filter((m: any) => m.status === 'down').length
  const pending = store.monitors.length - up - down
  uptimeDistData.value = [
    { name: 'Up', value: up, color: '#10b981' },
    { name: 'Down', value: down, color: '#ef4444' },
    { name: 'Pending', value: pending, color: '#f59e0b' },
  ]
}

function toggleFullscreen() {
  if (!document.fullscreenElement) {
    document.documentElement.requestFullscreen()
    isFullscreen.value = true
  } else {
    document.exitFullscreen()
    isFullscreen.value = false
  }
}

function toggleTheme() {
  isDark.value = !isDark.value
}

function startAutoRotate() {
  if (rotateTimer) clearInterval(rotateTimer)
  if (autoRotate.value) {
    rotateTimer = setInterval(() => {
      currentView.value = (currentView.value + 1) % views.length
    }, 30000)
  }
}

watch(autoRotate, startAutoRotate)

onMounted(async () => {
  await Promise.all([
    store.fetchMonitors(),
    store.fetchHeartbeats(),
    store.fetchSystemMetrics(),
    store.fetchAlertStats('24h'),
  ])
  updateChartData()
  connectWS()
  pollTimer = setInterval(() => {
    store.fetchMonitors()
    store.fetchAlertStats('24h')
  }, 30000)
  startAutoRotate()
})

onUnmounted(() => {
  ws?.close()
  if (pollTimer) clearInterval(pollTimer)
  if (rotateTimer) clearInterval(rotateTimer)
})

document.addEventListener('fullscreenchange', () => {
  isFullscreen.value = !!document.fullscreenElement
})
</script>

<template>
  <div class="min-h-screen transition-colors duration-300" :class="isDark ? 'bg-gray-950 text-white' : 'bg-gray-50 text-gray-900'">
    <MonitorHeader
      :connected="store.connected"
      :monitor-count="store.monitors.length"
      :dark="isDark"
      @toggle-fullscreen="toggleFullscreen"
      @toggle-theme="toggleTheme"
    />

    <!-- Status Banner -->
    <div
      class="px-6 py-2.5 text-center text-sm font-medium transition-colors"
      :class="store.overallStatus === 'operational'
        ? (isDark ? 'bg-green-900/30 text-green-400' : 'bg-green-100 text-green-700')
        : store.overallStatus === 'degraded'
          ? (isDark ? 'bg-red-900/30 text-red-400' : 'bg-red-100 text-red-700')
          : (isDark ? 'bg-gray-800 text-gray-400' : 'bg-gray-200 text-gray-600')"
    >
      {{ store.overallStatus === 'operational' ? 'All Systems Operational' : store.overallStatus === 'degraded' ? `${store.downCount} Service(s) Down` : 'Loading...' }}
    </div>

    <!-- View: Overview -->
    <div v-show="currentView === 0" class="p-6 space-y-6">
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="rounded-xl p-4" :class="isDark ? 'bg-gray-900 border border-gray-800' : 'bg-white border border-gray-200 shadow-sm'">
          <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">Total Monitors</p>
          <p class="text-2xl font-bold mt-1">{{ store.monitors.length }}</p>
        </div>
        <div class="rounded-xl p-4" :class="isDark ? 'bg-gray-900 border border-gray-800' : 'bg-white border border-gray-200 shadow-sm'">
          <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">Avg Uptime</p>
          <p class="text-2xl font-bold mt-1" :class="store.avgUptime >= 99 ? 'text-green-400' : store.avgUptime >= 95 ? 'text-yellow-400' : 'text-red-400'">
            {{ store.avgUptime.toFixed(2) }}%
          </p>
        </div>
        <div class="rounded-xl p-4" :class="isDark ? 'bg-gray-900 border border-gray-800' : 'bg-white border border-gray-200 shadow-sm'">
          <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">Heartbeats</p>
          <p class="text-2xl font-bold mt-1">{{ store.heartbeats.length }}</p>
        </div>
        <div class="rounded-xl p-4" :class="isDark ? 'bg-gray-900 border border-gray-800' : 'bg-white border border-gray-200 shadow-sm'">
          <p class="text-xs" :class="isDark ? 'text-gray-500' : 'text-gray-400'">Avg Latency</p>
          <p class="text-2xl font-bold mt-1">
            {{ store.monitors.length > 0 ? (store.monitors.reduce((s: number, m: any) => s + m.avg_latency, 0) / store.monitors.length).toFixed(0) : 0 }}ms
          </p>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        <div
          v-for="m in store.monitors"
          :key="m.id"
          class="rounded-xl p-4 transition-colors"
          :class="isDark ? 'bg-gray-900 border border-gray-800 hover:border-gray-700' : 'bg-white border border-gray-200 shadow-sm hover:shadow'"
        >
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-2">
              <span class="w-2.5 h-2.5 rounded-full" :class="m.status === 'up' ? 'bg-green-400' : 'bg-red-400'"></span>
              <span class="font-medium text-sm truncate">{{ m.name }}</span>
            </div>
            <span class="text-xs px-1.5 py-0.5 rounded" :class="isDark ? 'bg-gray-800 text-gray-400' : 'bg-gray-100 text-gray-500'">{{ m.type.toUpperCase() }}</span>
          </div>
          <p class="text-xs font-mono truncate mb-3" :class="isDark ? 'text-gray-500' : 'text-gray-400'">{{ m.target }}</p>
          <div class="space-y-2">
            <div class="flex justify-between text-xs">
              <span :class="isDark ? 'text-gray-500' : 'text-gray-400'">Uptime</span>
              <span class="font-medium" :class="m.uptime >= 99 ? 'text-green-400' : m.uptime >= 95 ? 'text-yellow-400' : 'text-red-400'">
                {{ m.uptime.toFixed(2) }}%
              </span>
            </div>
            <div class="h-1 rounded-full overflow-hidden" :class="isDark ? 'bg-gray-800' : 'bg-gray-200'">
              <div
                class="h-full rounded-full transition-all"
                :class="m.uptime >= 99 ? 'bg-green-500' : m.uptime >= 95 ? 'bg-yellow-500' : 'bg-red-500'"
                :style="{ width: `${Math.min(m.uptime, 100)}%` }"
              ></div>
            </div>
            <div class="flex justify-between text-xs">
              <span :class="isDark ? 'text-gray-500' : 'text-gray-400'">Latency</span>
              <span :class="isDark ? 'text-gray-300' : 'text-gray-600'">{{ m.avg_latency < 1000 ? `${m.avg_latency.toFixed(0)}ms` : `${(m.avg_latency / 1000).toFixed(2)}s` }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <AlertSummaryCard />
        <DeploymentFeed />
      </div>
    </div>

    <!-- View: Charts -->
    <div v-show="currentView === 1" class="p-6 space-y-6">
      <div class="rounded-xl p-4" :class="isDark ? 'bg-gray-900 border border-gray-800' : 'bg-white border border-gray-200 shadow-sm'">
        <PieChart title="Monitor Status" :data="uptimeDistData" :dark="isDark" height="300px" />
      </div>
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-4">
        <div
          v-for="m in store.monitors.slice(0, 6)"
          :key="m.id"
          class="rounded-xl p-3"
          :class="isDark ? 'bg-gray-900 border border-gray-800' : 'bg-white border border-gray-200 shadow-sm'"
        >
          <GaugeChart :value="m.uptime" :dark="isDark" :title="m.name" height="160px" />
        </div>
      </div>
    </div>

    <!-- View: Servers -->
    <div v-show="currentView === 2" class="p-6 space-y-6">
      <ServerStatusGrid />
    </div>

    <!-- View Navigation Dots -->
    <div class="fixed bottom-10 left-1/2 -translate-x-1/2 flex gap-2 z-10">
      <button
        v-for="(v, i) in views"
        :key="v"
        class="w-2 h-2 rounded-full transition-all"
        :class="currentView === i ? 'bg-blue-500 w-6' : isDark ? 'bg-gray-700 hover:bg-gray-600' : 'bg-gray-300 hover:bg-gray-400'"
        @click="currentView = i; autoRotate = false"
      ></button>
    </div>

    <!-- Footer -->
    <div class="fixed bottom-0 left-0 right-0 px-6 py-1.5 text-center text-[10px] border-t transition-colors"
      :class="isDark ? 'bg-gray-950 border-gray-800 text-gray-600' : 'bg-gray-50 border-gray-200 text-gray-400'">
      DeployPilot Dashboard TV · Auto-refresh every 30s · View {{ currentView + 1 }}/{{ views.length }}
    </div>
  </div>
</template>
