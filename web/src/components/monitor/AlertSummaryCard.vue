<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import * as queryApi from '@/api/modules/monitor_query'
import { AlertTriangle, CheckCircle, Info, XCircle } from 'lucide-vue-next'

const stats = ref<any>(null)
const loading = ref(true)

const criticalCount = computed(() => stats.value?.stats?.critical || 0)
const warningCount = computed(() => stats.value?.stats?.warning || 0)
const firingCount = computed(() => stats.value?.stats?.firing || 0)

async function fetchStats() {
  loading.value = true
  try {
    const res = await queryApi.getAlertStats('24h')
    if (res.data.status === 'success') {
      stats.value = res.data.data
    }
  } catch { /* silent */ } finally {
    loading.value = false
  }
}

onMounted(fetchStats)
</script>

<template>
  <div class="bg-gray-900/50 border border-gray-800 rounded-xl p-4">
    <h3 class="text-sm font-semibold text-gray-400 mb-3">Alerts (24h)</h3>
    <div v-if="loading" class="text-center py-4 text-gray-600 text-sm">Loading...</div>
    <div v-else class="grid grid-cols-2 gap-3">
      <div class="flex items-center gap-2">
        <XCircle class="w-4 h-4 text-red-400" />
        <div>
          <p class="text-lg font-bold text-red-400">{{ criticalCount }}</p>
          <p class="text-[10px] text-gray-500">Critical</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <AlertTriangle class="w-4 h-4 text-yellow-400" />
        <div>
          <p class="text-lg font-bold text-yellow-400">{{ warningCount }}</p>
          <p class="text-[10px] text-gray-500">Warning</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <Info class="w-4 h-4 text-blue-400" />
        <div>
          <p class="text-lg font-bold text-blue-400">{{ firingCount }}</p>
          <p class="text-[10px] text-gray-500">Firing</p>
        </div>
      </div>
      <div class="flex items-center gap-2">
        <CheckCircle class="w-4 h-4 text-green-400" />
        <div>
          <p class="text-lg font-bold text-green-400">{{ stats?.stats?.resolved || 0 }}</p>
          <p class="text-[10px] text-gray-500">Resolved</p>
        </div>
      </div>
    </div>
  </div>
</template>
