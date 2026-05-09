<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { RefreshCw, Activity } from 'lucide-vue-next'
import * as uptimeApi from '@/api/modules/uptime'
import type { StatusPageData } from '@/types/monitor'

const data = ref<StatusPageData | null>(null)
const loading = ref(true)
const error = ref(false)
let refreshTimer: ReturnType<typeof setInterval> | null = null

async function fetchData() {
  try {
    const res = await uptimeApi.getStatusPage()
    if (res.data.status === 'success') {
      data.value = res.data.data
      error.value = false
    }
  } catch {
    error.value = true
  } finally {
    loading.value = false
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
  fetchData()
  refreshTimer = setInterval(fetchData, 60000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
  <div class="min-h-screen bg-gray-50">
    <div class="max-w-3xl mx-auto px-4 py-12">
      <!-- Header -->
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-gray-900">System Status</h1>
        <p class="mt-2 text-gray-500">Real-time service availability</p>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="text-center py-12 text-gray-500">Loading status...</div>

      <!-- Error -->
      <div v-else-if="error" class="text-center py-12">
        <p class="text-red-500 mb-4">Failed to load status</p>
        <button class="px-4 py-2 text-sm border rounded-md hover:bg-gray-100" @click="fetchData">
          <RefreshCw class="w-4 h-4 inline mr-1" /> Retry
        </button>
      </div>

      <!-- Status Content -->
      <div v-else-if="data">
        <!-- Overall Status Banner -->
        <div
          class="rounded-xl p-6 mb-8 text-center"
          :class="data.up_monitors === data.total_monitors ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'"
        >
          <div class="flex items-center justify-center gap-2 mb-2">
            <span
              class="w-3 h-3 rounded-full"
              :class="data.up_monitors === data.total_monitors ? 'bg-green-500' : 'bg-red-500'"
            ></span>
            <span
              class="text-xl font-semibold"
              :class="data.up_monitors === data.total_monitors ? 'text-green-700' : 'text-red-700'"
            >
              {{ data.up_monitors === data.total_monitors ? 'All Systems Operational' : `${data.total_monitors - data.up_monitors} Service(s) Down` }}
            </span>
          </div>
          <p class="text-sm text-gray-500">
            Overall uptime: {{ formatUptime(data.overall_uptime) }}
          </p>
        </div>

        <!-- Monitor List -->
        <div class="space-y-3">
          <div
            v-for="m in data.monitors"
            :key="m.id"
            class="bg-white rounded-lg border p-4"
          >
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-3">
                <span
                  class="w-2.5 h-2.5 rounded-full"
                  :class="m.status === 'up' ? 'bg-green-500' : 'bg-red-500'"
                ></span>
                <div>
                  <p class="font-medium text-gray-900">{{ m.name }}</p>
                  <p class="text-xs text-gray-400">{{ m.type.toUpperCase() }}</p>
                </div>
              </div>
              <div class="text-right">
                <p class="text-sm font-medium" :class="m.status === 'up' ? 'text-green-600' : 'text-red-600'">
                  {{ m.status === 'up' ? 'Operational' : 'Down' }}
                </p>
                <p class="text-xs text-gray-400">{{ formatLatency(m.avg_latency) }}</p>
              </div>
            </div>
            <div class="mt-3">
              <div class="flex justify-between text-xs text-gray-400 mb-1">
                <span>Uptime</span>
                <span>{{ formatUptime(m.uptime) }}</span>
              </div>
              <div class="h-1.5 bg-gray-100 rounded-full overflow-hidden">
                <div
                  class="h-full rounded-full transition-all"
                  :class="m.uptime >= 99 ? 'bg-green-500' : m.uptime >= 95 ? 'bg-yellow-500' : 'bg-red-500'"
                  :style="{ width: `${m.uptime}%` }"
                ></div>
              </div>
            </div>
          </div>
        </div>

        <!-- Footer -->
        <div class="mt-8 text-center text-xs text-gray-400">
          <p>Auto-refreshes every 60 seconds</p>
          <button class="mt-1 text-blue-500 hover:underline" @click="fetchData">Refresh now</button>
        </div>
      </div>
    </div>
  </div>
</template>
