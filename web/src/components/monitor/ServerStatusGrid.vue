<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as serverApi from '@/api/modules/servers'

interface ServerStatus {
  id: string
  name: string
  host: string
  status: string
}

const servers = ref<ServerStatus[]>([])
const loading = ref(true)

async function fetchServers() {
  loading.value = true
  try {
    const res = await serverApi.list()
    if (res.data.status === 'success') {
      servers.value = (res.data.data || []).map((s: any) => ({
        id: s.id, name: s.name, host: s.host, status: s.status || 'unknown',
      }))
    }
  } catch { /* silent */ } finally {
    loading.value = false
  }
}

function statusColor(status: string) {
  switch (status) {
    case 'reachable': case 'online': return 'bg-green-500'
    case 'unreachable': case 'offline': return 'bg-red-500'
    default: return 'bg-gray-500'
  }
}

onMounted(fetchServers)
</script>

<template>
  <div class="bg-gray-900/50 border border-gray-800 rounded-xl p-4">
    <h3 class="text-sm font-semibold text-gray-400 mb-3">Servers</h3>
    <div v-if="loading" class="text-center py-4 text-gray-600 text-sm">Loading...</div>
    <div v-else-if="servers.length === 0" class="text-center py-4 text-gray-600 text-sm">No servers</div>
    <div v-else class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-2">
      <div v-for="s in servers" :key="s.id" class="flex items-center gap-2 px-3 py-2 bg-gray-800/50 rounded-lg">
        <span class="w-2 h-2 rounded-full shrink-0" :class="statusColor(s.status)"></span>
        <div class="min-w-0">
          <p class="text-xs font-medium truncate">{{ s.name }}</p>
          <p class="text-[10px] text-gray-500 font-mono truncate">{{ s.host }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
