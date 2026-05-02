<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as uptimeApi from '@/api/modules/uptime'
import * as queryApi from '@/api/modules/monitor_query'
import PageHeader from '@/components/common/PageHeader.vue'
import { Download, FileJson, FileSpreadsheet } from 'lucide-vue-next'

const monitors = ref<any[]>([])
const loading = ref(false)
const exportForm = ref({
  monitorId: '',
  format: 'csv' as 'csv' | 'json',
  start: new Date(Date.now() - 7 * 24 * 3600 * 1000).toISOString().slice(0, 16),
  end: new Date().toISOString().slice(0, 16),
  alertFormat: 'csv' as 'csv' | 'json',
  alertStatus: '',
  alertSeverity: '',
})

async function fetchMonitors() {
  try {
    const res = await uptimeApi.listMonitors()
    if (res.data.status === 'success') monitors.value = res.data.data || []
  } catch { /* silent */ }
}

async function exportMonitorData() {
  if (!exportForm.value.monitorId) return
  loading.value = true
  try {
    const res = await queryApi.exportMonitorData(exportForm.value.monitorId, exportForm.value.format, {
      start: new Date(exportForm.value.start).toISOString(),
      end: new Date(exportForm.value.end).toISOString(),
    })
    downloadBlob(res.data, `monitor-${exportForm.value.monitorId}.${exportForm.value.format}`)
  } catch { /* silent */ } finally { loading.value = false }
}

async function exportAlertHistory() {
  loading.value = true
  try {
    const params: any = { format: exportForm.value.alertFormat }
    if (exportForm.value.alertStatus) params.status = exportForm.value.alertStatus
    if (exportForm.value.alertSeverity) params.severity = exportForm.value.alertSeverity
    const res = await queryApi.exportAlertHistory(exportForm.value.alertFormat, params)
    downloadBlob(res.data, `alerts.${exportForm.value.alertFormat}`)
  } catch { /* silent */ } finally { loading.value = false }
}

function downloadBlob(data: Blob, filename: string) {
  const url = URL.createObjectURL(data)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(fetchMonitors)
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Data Export" description="Export monitoring data and alert history" />
    <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
      <h3 class="text-base font-semibold mb-4">Monitor Check Results</h3>
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
        <div>
          <label class="block text-sm font-medium mb-1">Monitor</label>
          <select v-model="exportForm.monitorId" class="w-full px-3 py-2 border rounded-md text-sm">
            <option value="">Select a monitor...</option>
            <option v-for="m in monitors" :key="m.id" :value="m.id">{{ m.name }} ({{ m.type }})</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Format</label>
          <div class="flex gap-2">
            <button class="flex items-center gap-1 px-3 py-2 text-sm border rounded-md" :class="exportForm.format === 'csv' ? 'bg-blue-50 border-blue-300 text-blue-700' : 'hover:bg-gray-50'" @click="exportForm.format = 'csv'">
              <FileSpreadsheet class="w-4 h-4" /> CSV
            </button>
            <button class="flex items-center gap-1 px-3 py-2 text-sm border rounded-md" :class="exportForm.format === 'json' ? 'bg-blue-50 border-blue-300 text-blue-700' : 'hover:bg-gray-50'" @click="exportForm.format = 'json'">
              <FileJson class="w-4 h-4" /> JSON
            </button>
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Start</label>
          <input v-model="exportForm.start" type="datetime-local" class="w-full px-3 py-2 border rounded-md text-sm" />
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">End</label>
          <input v-model="exportForm.end" type="datetime-local" class="w-full px-3 py-2 border rounded-md text-sm" />
        </div>
      </div>
      <button class="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 flex items-center gap-1" :disabled="!exportForm.monitorId || loading" @click="exportMonitorData">
        <Download class="w-4 h-4" /> {{ loading ? 'Exporting...' : 'Export Monitor Data' }}
      </button>
    </div>
    <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
      <h3 class="text-base font-semibold mb-4">Alert History</h3>
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
        <div>
          <label class="block text-sm font-medium mb-1">Format</label>
          <div class="flex gap-2">
            <button class="flex items-center gap-1 px-3 py-2 text-sm border rounded-md" :class="exportForm.alertFormat === 'csv' ? 'bg-blue-50 border-blue-300 text-blue-700' : 'hover:bg-gray-50'" @click="exportForm.alertFormat = 'csv'">
              <FileSpreadsheet class="w-4 h-4" /> CSV
            </button>
            <button class="flex items-center gap-1 px-3 py-2 text-sm border rounded-md" :class="exportForm.alertFormat === 'json' ? 'bg-blue-50 border-blue-300 text-blue-700' : 'hover:bg-gray-50'" @click="exportForm.alertFormat = 'json'">
              <FileJson class="w-4 h-4" /> JSON
            </button>
          </div>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Status</label>
          <select v-model="exportForm.alertStatus" class="w-full px-3 py-2 border rounded-md text-sm">
            <option value="">All</option>
            <option value="firing">Firing</option>
            <option value="resolved">Resolved</option>
          </select>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Severity</label>
          <select v-model="exportForm.alertSeverity" class="w-full px-3 py-2 border rounded-md text-sm">
            <option value="">All</option>
            <option value="critical">Critical</option>
            <option value="warning">Warning</option>
            <option value="info">Info</option>
          </select>
        </div>
      </div>
      <button class="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 flex items-center gap-1" :disabled="loading" @click="exportAlertHistory">
        <Download class="w-4 h-4" /> {{ loading ? 'Exporting...' : 'Export Alert History' }}
      </button>
    </div>
  </div>
</template>
