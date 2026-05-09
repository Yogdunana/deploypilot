<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import * as serverApi from '@/api/modules/servers'
import * as monitorApi from '@/api/modules/monitor'
import GaugeChart from '@/components/charts/GaugeChart.vue'
import PageHeader from '@/components/common/PageHeader.vue'

const route = useRoute()
const serverId = route.params.id as string

const server = ref<any>(null)
const metrics = ref<any>(null)
const loading = ref(true)
let refreshTimer: ReturnType<typeof setInterval> | null = null

async function fetchData() {
  loading.value = true
  try {
    const [serverRes, metricsRes] = await Promise.all([
      serverApi.list(),
      monitorApi.getSystemMetrics(),
    ])
    if (serverRes.data.status === 'success') {
      server.value = (serverRes.data.data || []).find((s: any) => s.id === serverId)
    }
    if (metricsRes.data.status === 'success') {
      metrics.value = metricsRes.data.data
    }
  } catch { /* silent */ } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchData()
  refreshTimer = setInterval(fetchData, 15000)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
})
</script>

<template>
  <div class="space-y-6">
    <PageHeader :title="server?.name || 'Server Detail'" :description="server?.host" />
    <div v-if="loading" class="text-center py-12 text-gray-500">Loading...</div>
    <div v-else-if="metrics" class="space-y-6">
      <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div class="p-4 bg-white border border-gray-200 rounded-xl shadow-sm">
          <GaugeChart :value="metrics.cpu_usage || 0" title="CPU" unit="%" height="180px" :dark="false" />
        </div>
        <div class="p-4 bg-white border border-gray-200 rounded-xl shadow-sm">
          <GaugeChart :value="metrics.memory_usage || 0" title="Memory" unit="%" height="180px" :dark="false" />
        </div>
        <div class="p-4 bg-white border border-gray-200 rounded-xl shadow-sm">
          <GaugeChart :value="metrics.disk_usage || 0" title="Disk" unit="%" height="180px" :dark="false" />
        </div>
        <div class="p-4 bg-white border border-gray-200 rounded-xl shadow-sm">
          <GaugeChart :value="metrics.network_usage || 0" title="Network" unit="%" height="180px" :dark="false" />
        </div>
      </div>
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <div class="p-4 bg-white border border-gray-200 rounded-xl shadow-sm">
          <h3 class="text-sm font-semibold mb-3">System Info</h3>
          <dl class="space-y-2 text-sm">
            <div class="flex justify-between"><dt class="text-gray-500">OS</dt><dd>{{ metrics.os || 'N/A' }}</dd></div>
            <div class="flex justify-between"><dt class="text-gray-500">Uptime</dt><dd>{{ metrics.uptime || 'N/A' }}</dd></div>
            <div class="flex justify-between"><dt class="text-gray-500">CPU Cores</dt><dd>{{ metrics.cpu_cores || 'N/A' }}</dd></div>
            <div class="flex justify-between"><dt class="text-gray-500">Total Memory</dt><dd>{{ metrics.total_memory || 'N/A' }}</dd></div>
            <div class="flex justify-between"><dt class="text-gray-500">Total Disk</dt><dd>{{ metrics.total_disk || 'N/A' }}</dd></div>
          </dl>
        </div>
        <div class="p-4 bg-white border border-gray-200 rounded-xl shadow-sm">
          <h3 class="text-sm font-semibold mb-3">Containers</h3>
          <div class="text-sm text-gray-500">Container data available via system metrics endpoint</div>
        </div>
      </div>
    </div>
  </div>
</template>
