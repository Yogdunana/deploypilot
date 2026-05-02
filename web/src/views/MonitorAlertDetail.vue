<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as queryApi from '@/api/modules/monitor_query'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { AlertTriangle, Clock, CheckCircle, ArrowLeft } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const alertId = route.params.id as string

const alert = ref<any>(null)
const related = ref<any[]>([])
const loading = ref(true)

async function fetchData() {
  loading.value = true
  try {
    const res = await queryApi.getAlertHistory(alertId)
    if (res.data.status === 'success') {
      alert.value = res.data.data.alert
      related.value = res.data.data.related || []
    }
  } catch { /* silent */ } finally {
    loading.value = false
  }
}

function severityColor(severity: string) {
  switch (severity) {
    case 'critical': return 'text-red-600 bg-red-50'
    case 'warning': return 'text-yellow-600 bg-yellow-50'
    case 'info': return 'text-blue-600 bg-blue-50'
    default: return 'text-gray-600 bg-gray-50'
  }
}

function formatDate(dateStr: string) {
  if (!dateStr) return 'N/A'
  return new Date(dateStr).toLocaleString()
}

onMounted(fetchData)
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Alert Detail">
      <template #actions>
        <button class="px-3 py-2 text-sm border rounded-md hover:bg-gray-50 flex items-center gap-1" @click="router.back()">
          <ArrowLeft class="w-4 h-4" /> Back
        </button>
      </template>
    </PageHeader>
    <div v-if="loading" class="text-center py-12 text-gray-500">Loading...</div>
    <div v-else-if="alert" class="space-y-6">
      <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
        <div class="flex items-start justify-between mb-4">
          <div class="flex items-center gap-3">
            <AlertTriangle class="w-6 h-6" :class="alert.severity === 'critical' ? 'text-red-500' : 'text-yellow-500'" />
            <div>
              <h2 class="text-lg font-semibold">{{ alert.rule_name }}</h2>
              <p class="text-sm text-gray-500 mt-0.5">{{ alert.message }}</p>
            </div>
          </div>
          <StatusBadge :status="alert.status" />
        </div>
        <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 mt-4">
          <div class="p-3 bg-gray-50 rounded-lg">
            <p class="text-xs text-gray-500">Severity</p>
            <span class="inline-block mt-1 px-2 py-0.5 text-xs font-medium rounded" :class="severityColor(alert.severity)">
              {{ alert.severity?.toUpperCase() }}
            </span>
          </div>
          <div class="p-3 bg-gray-50 rounded-lg">
            <p class="text-xs text-gray-500">Value</p>
            <p class="text-lg font-semibold mt-1">{{ alert.value }}</p>
          </div>
          <div class="p-3 bg-gray-50 rounded-lg">
            <p class="text-xs text-gray-500">Threshold</p>
            <p class="text-lg font-semibold mt-1">{{ alert.threshold }}</p>
          </div>
          <div class="p-3 bg-gray-50 rounded-lg">
            <p class="text-xs text-gray-500">Duration</p>
            <p class="text-lg font-semibold mt-1">
              {{ alert.resolved_at ? `${Math.round((new Date(alert.resolved_at).getTime() - new Date(alert.fired_at).getTime()) / 60000)}m` : 'Active' }}
            </p>
          </div>
        </div>
        <div class="flex gap-6 mt-4 text-sm text-gray-500">
          <div class="flex items-center gap-1"><Clock class="w-4 h-4" /> Fired: {{ formatDate(alert.fired_at) }}</div>
          <div v-if="alert.resolved_at" class="flex items-center gap-1"><CheckCircle class="w-4 h-4" /> Resolved: {{ formatDate(alert.resolved_at) }}</div>
        </div>
      </div>
      <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
        <h3 class="text-sm font-semibold mb-4">Related Events</h3>
        <div v-if="related.length === 0" class="text-sm text-gray-500">No related events</div>
        <div v-else class="space-y-3">
          <div v-for="r in related" :key="r.id" class="flex items-center gap-3 px-3 py-2 bg-gray-50 rounded-lg">
            <span class="w-2 h-2 rounded-full" :class="r.status === 'firing' ? 'bg-red-500' : 'bg-green-500'"></span>
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium truncate">{{ r.rule_name }}</p>
              <p class="text-xs text-gray-500">{{ r.message }}</p>
            </div>
            <span class="text-xs text-gray-400 shrink-0">{{ formatDate(r.fired_at) }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
