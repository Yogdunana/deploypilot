<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as deployApi from '@/api/modules/deployments'

interface Deployment {
  id: string
  app_name: string
  status: string
  created_at: string
}

const deployments = ref<Deployment[]>([])
const loading = ref(true)

async function fetchDeployments() {
  loading.value = true
  try {
    const res = await deployApi.list(undefined, undefined, { page: 1, page_size: 10 })
    if (res.data.status === 'success') {
      deployments.value = (res.data.data || []).slice(0, 10).map((d: any) => ({
        id: d.id, app_name: d.app_name, status: d.status, created_at: d.created_at,
      }))
    }
  } catch { /* silent */ } finally {
    loading.value = false
  }
}

function statusIcon(status: string) {
  switch (status) {
    case 'success': return '✓'
    case 'failed': return '✗'
    case 'deploying': return '⟳'
    default: return '○'
  }
}

function statusColor(status: string) {
  switch (status) {
    case 'success': return 'text-green-400'
    case 'failed': return 'text-red-400'
    case 'deploying': return 'text-blue-400'
    default: return 'text-gray-400'
  }
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

onMounted(fetchDeployments)
</script>

<template>
  <div class="bg-gray-900/50 border border-gray-800 rounded-xl p-4">
    <h3 class="text-sm font-semibold text-gray-400 mb-3">Recent Deployments</h3>
    <div v-if="loading" class="text-center py-4 text-gray-600 text-sm">Loading...</div>
    <div v-else-if="deployments.length === 0" class="text-center py-4 text-gray-600 text-sm">No deployments</div>
    <div v-else class="space-y-1.5 max-h-64 overflow-auto">
      <div v-for="d in deployments" :key="d.id" class="flex items-center justify-between px-2 py-1.5 rounded hover:bg-gray-800/50">
        <div class="flex items-center gap-2 min-w-0">
          <span class="text-sm" :class="statusColor(d.status)">{{ statusIcon(d.status) }}</span>
          <span class="text-xs font-medium truncate">{{ d.app_name }}</span>
        </div>
        <span class="text-[10px] text-gray-500 shrink-0">{{ timeAgo(d.created_at) }}</span>
      </div>
    </div>
  </div>
</template>
