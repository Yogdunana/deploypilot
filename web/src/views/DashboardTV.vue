<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import * as uptimeApi from '@/api/modules/uptime'
import type { UptimeMonitor } from '@/types/monitor'

const monitors = ref<UptimeMonitor[]>([])
const connected = ref(false)
let ws: WebSocket | null = null
let pollTimer: ReturnType<typeof setInterval> | null = null

const overallStatus = computed(() => {
  if (monitors.value.length === 0) return 'loading'
  return monitors.value.every((m) => m.status === 'up') ? 'operational' : 'degraded'
})

const downCount = computed(() => monitors.value.filter((m) => m.status === 'down').length)

function connectWS() {
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  ws = new WebSocket(`${protocol}//${window.location.host}/ws/monitor`)

  ws.onopen = () => {
    connected.value = true
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      if (msg.type === 'monitor_check' && msg.data?.results) {
        monitors.value = msg.data.results
      } else if (msg.type === 'monitor_update' && msg.data) {
        const idx = monitors.value.findIndex((m) => m.id === msg.data.id)
        if (idx >= 0) {
          monitors.value[idx] = { ...monitors.value[idx], ...msg.data }
        } else {
          monitors.value.push(msg.data)
        }
      } else if (msg.type === 'full_sync') {
        monitors.value = msg.data || []
      }
    } catch {
      /* ignore parse errors */
    }
  }

  ws.onclose = () => {
    connected.value = false
    setTimeout(connectWS, 5000)
  }

  ws.onerror = () => {
    ws?.close()
  }
}

async function fetchMonitors() {
  try {
    const res = await uptimeApi.listMonitors()
    if (res.data.status === 'success') {
      monitors.value = res.data.data
    }
  } catch {
    /* handled silently */
  }
}

function formatUptime(value: number): string {
  return `${value.toFixed(2)}%`
}

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms.toFixed(0)}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

onMounted(() => {
  fetchMonitors()
  connectWS()
  pollTimer = setInterval(fetchMonitors, 30000)
})

onUnmounted(() => {
  ws?.close()
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<template>
  <div class="min-h-screen bg-gray-950 text-white">
    <!-- Header -->
    <div class="border-b border-gray-800 px-6 py-4">
      <div class="flex items-center justify-between">
        <h1 class="text-xl font-bold">DeployPilot Monitor</h1>
        <div class="flex items-center gap-3">
          <span class="text-xs px-2 py-1 rounded" :class="connected ? 'bg-green-900 text-green-300' : 'bg-red-900 text-red-300'">
            {{ connected ? 'LIVE' : 'RECONNECTING' }}
          </span>
          <span class="text-xs text-gray-500">
            {{ monitors.length }} monitors
          </span>
        </div>
      </div>
    </div>

    <!-- Status Banner -->
    <div
      class="px-6 py-3 text-center text-sm font-medium"
      :class="overallStatus === 'operational' ? 'bg-green-900/30 text-green-400' : overallStatus === 'degraded' ? 'bg-red-900/30 text-red-400' : 'bg-gray-800 text-gray-400'"
    >
      {{ overallStatus === 'operational' ? 'All Systems Operational' : overallStatus === 'degraded' ? `${downCount} Service(s) Down` : 'Loading...' }}
    </div>

    <!-- Monitor Grid -->
    <div class="p-6 grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      <div
        v-for="m in monitors"
        :key="m.id"
        class="bg-gray-900 border border-gray-800 rounded-xl p-4"
      >
        <div class="flex items-center justify-between mb-3">
          <div class="flex items-center gap-2">
            <span
              class="w-2.5 h-2.5 rounded-full"
              :class="m.status === 'up' ? 'bg-green-400' : 'bg-red-400'"
            ></span>
            <span class="font-medium text-sm truncate">{{ m.name }}</span>
          </div>
          <span class="text-xs px-1.5 py-0.5 rounded bg-gray-800 text-gray-400">{{ m.type.toUpperCase() }}</span>
        </div>

        <p class="text-xs text-gray-500 font-mono truncate mb-3">{{ m.target }}</p>

        <div class="space-y-2">
          <div class="flex justify-between text-xs">
            <span class="text-gray-500">Uptime</span>
            <span class="font-medium" :class="m.uptime >= 99 ? 'text-green-400' : m.uptime >= 95 ? 'text-yellow-400' : 'text-red-400'">
              {{ formatUptime(m.uptime) }}
            </span>
          </div>
          <div class="h-1 bg-gray-800 rounded-full overflow-hidden">
            <div
              class="h-full rounded-full transition-all"
              :class="m.uptime >= 99 ? 'bg-green-500' : m.uptime >= 95 ? 'bg-yellow-500' : 'bg-red-500'"
              :style="{ width: `${Math.min(m.uptime, 100)}%` }"
            ></div>
          </div>
          <div class="flex justify-between text-xs">
            <span class="text-gray-500">Latency</span>
            <span class="text-gray-300">{{ formatLatency(m.avg_latency) }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Empty state -->
    <div v-if="monitors.length === 0" class="flex items-center justify-center h-64 text-gray-600">
      No monitors configured
    </div>

    <!-- Footer -->
    <div class="fixed bottom-0 left-0 right-0 bg-gray-950 border-t border-gray-800 px-6 py-2 text-center text-xs text-gray-600">
      DeployPilot Dashboard TV &middot; Auto-refresh enabled
    </div>
  </div>
</template>
