<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  Activity,
  Plus,
  RefreshCw,
  Zap,
  MoreHorizontal,
  Globe,
  Server,
  Wifi,
  Pencil,
  Trash2,
} from 'lucide-vue-next'
import * as uptimeApi from '@/api/modules/uptime'
import type { UptimeMonitor, MonitorSLA, MonitorCheckResult } from '@/types/monitor'

const router = useRouter()

const monitors = ref<UptimeMonitor[]>([])
const loading = ref(false)
const searchQuery = ref('')
const checkingId = ref<string | null>(null)

// Dialog state
const dialogOpen = ref(false)
const dialogTitle = ref('')
const editingMonitor = ref<UptimeMonitor | null>(null)
const formName = ref('')
const formType = ref('http')
const formTarget = ref('')
const formInterval = ref(60)
const formTimeout = ref(10)
const submitting = ref(false)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingMonitor = ref<UptimeMonitor | null>(null)
const deleting = ref(false)

// SLA dialog
const slaDialogOpen = ref(false)
const slaMonitorName = ref('')
const slaLoading = ref(false)
const slaData = ref<MonitorSLA | null>(null)

// Results dialog
const resultsDialogOpen = ref(false)
const resultsMonitorName = ref('')
const resultsLoading = ref(false)
const resultsData = ref<MonitorCheckResult[]>([])

const typeOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'TCP', value: 'tcp' },
  { label: 'Ping (ICMP)', value: 'ping' },
]

const filteredMonitors = computed(() => {
  if (!searchQuery.value) return monitors.value
  const q = searchQuery.value.toLowerCase()
  return monitors.value.filter(
    (m) => m.name.toLowerCase().includes(q) || m.target.toLowerCase().includes(q),
  )
})

function mapStatus(status: string): string {
  if (status === 'up') return 'success'
  if (status === 'down') return 'destructive'
  return 'warning'
}

function getTypeIcon(type: string) {
  switch (type) {
    case 'http': return Globe
    case 'tcp': return Server
    case 'ping': return Wifi
    default: return Activity
  }
}

function formatUptime(value: number): string {
  return `${value.toFixed(2)}%`
}

function formatLatency(ms: number): string {
  if (ms < 1000) return `${ms.toFixed(0)}ms`
  return `${(ms / 1000).toFixed(2)}s`
}

async function fetchMonitors() {
  loading.value = true
  try {
    const res = await uptimeApi.listMonitors()
    if (res.data.status === 'success') {
      monitors.value = res.data.data
    }
  } catch {
    /* handled silently */
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  editingMonitor.value = null
  dialogTitle.value = 'Create Monitor'
  formName.value = ''
  formType.value = 'http'
  formTarget.value = ''
  formInterval.value = 60
  formTimeout.value = 10
  dialogOpen.value = true
}

function openEditDialog(monitor: UptimeMonitor) {
  editingMonitor.value = monitor
  dialogTitle.value = 'Edit Monitor'
  formName.value = monitor.name
  formType.value = monitor.type
  formTarget.value = monitor.target
  formInterval.value = monitor.interval
  formTimeout.value = monitor.timeout
  dialogOpen.value = true
}

async function handleSubmit() {
  if (!formName.value.trim() || !formTarget.value.trim()) return
  submitting.value = true
  try {
    const payload: Partial<UptimeMonitor> = {
      name: formName.value.trim(),
      type: formType.value,
      target: formTarget.value.trim(),
      interval: formInterval.value,
      timeout: formTimeout.value,
    }
    if (editingMonitor.value) {
      await uptimeApi.updateMonitor(editingMonitor.value.id, payload)
    } else {
      await uptimeApi.createMonitor(payload)
    }
    dialogOpen.value = false
    fetchMonitors()
  } catch {
    /* handled silently */
  } finally {
    submitting.value = false
  }
}

function openDeleteDialog(monitor: UptimeMonitor) {
  deletingMonitor.value = monitor
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingMonitor.value) return
  deleting.value = true
  try {
    await uptimeApi.deleteMonitor(deletingMonitor.value.id)
    fetchMonitors()
  } catch {
    /* handled silently */
  } finally {
    deleting.value = false
    deletingMonitor.value = null
  }
}

async function handleCheck(monitor: UptimeMonitor) {
  checkingId.value = monitor.id
  try {
    await uptimeApi.checkMonitor(monitor.id)
    fetchMonitors()
  } catch {
    /* handled silently */
  } finally {
    checkingId.value = null
  }
}

async function handleCheckAll() {
  try {
    await uptimeApi.checkAllMonitors()
    setTimeout(fetchMonitors, 2000)
  } catch {
    /* handled silently */
  }
}

async function openSLADialog(monitor: UptimeMonitor) {
  slaMonitorName.value = monitor.name
  slaLoading.value = true
  slaDialogOpen.value = true
  try {
    const res = await uptimeApi.getMonitorSLA(monitor.id, 30)
    if (res.data.status === 'success') {
      slaData.value = res.data.data
    }
  } catch {
    /* handled silently */
  } finally {
    slaLoading.value = false
  }
}

async function openResultsDialog(monitor: UptimeMonitor) {
  resultsMonitorName.value = monitor.name
  resultsLoading.value = true
  resultsDialogOpen.value = true
  try {
    const res = await uptimeApi.getMonitorResults(monitor.id, 20)
    if (res.data.status === 'success') {
      resultsData.value = res.data.data
    }
  } catch {
    /* handled silently */
  } finally {
    resultsLoading.value = false
  }
}

onMounted(fetchMonitors)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Uptime Monitors</h1>
      <div class="flex gap-2">
        <button class="px-3 py-2 text-sm border rounded-md hover:bg-gray-50" @click="handleCheckAll">
          Check All
        </button>
        <button class="px-3 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700" @click="openCreateDialog">
          + Create
        </button>
      </div>
    </div>

    <div class="relative w-full sm:w-72">
      <input v-model="searchQuery" placeholder="Search monitors..." class="w-full px-3 py-2 text-sm border rounded-md" />
    </div>

    <div v-if="loading" class="text-center py-8 text-gray-500">Loading...</div>

    <div v-else-if="filteredMonitors.length > 0" class="border rounded-lg overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 border-b">
          <tr>
            <th class="px-4 py-3 text-left">Name</th>
            <th class="px-4 py-3 text-left">Type</th>
            <th class="px-4 py-3 text-left">Target</th>
            <th class="px-4 py-3 text-left">Status</th>
            <th class="px-4 py-3 text-left">Uptime</th>
            <th class="px-4 py-3 text-left">Latency</th>
            <th class="px-4 py-3 text-left">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in filteredMonitors" :key="m.id" class="border-b last:border-0 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium">{{ m.name }}</td>
            <td class="px-4 py-3"><span class="px-2 py-0.5 text-xs border rounded-full">{{ m.type.toUpperCase() }}</span></td>
            <td class="px-4 py-3 font-mono text-gray-500">{{ m.target }}</td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full" :class="m.status === 'up' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'">
                <span class="w-1.5 h-1.5 rounded-full" :class="m.status === 'up' ? 'bg-green-500' : 'bg-red-500'"></span>
                {{ m.status === 'up' ? 'UP' : 'DOWN' }}
              </span>
            </td>
            <td class="px-4 py-3">{{ formatUptime(m.uptime) }}</td>
            <td class="px-4 py-3 text-gray-500">{{ formatLatency(m.avg_latency) }}</td>
            <td class="px-4 py-3">
              <div class="flex gap-1">
                <button class="p-1 hover:bg-gray-100 rounded" title="Check now" :disabled="checkingId === m.id" @click="handleCheck(m)">
                  <Zap class="w-4 h-4" />
                </button>
                <button class="p-1 hover:bg-gray-100 rounded" title="SLA" @click="openSLADialog(m)">
                  <Activity class="w-4 h-4" />
                </button>
                <button class="p-1 hover:bg-gray-100 rounded" title="Edit" @click="openEditDialog(m)">
                  <Pencil class="w-4 h-4" />
                </button>
                <button class="p-1 hover:bg-red-50 text-red-600 rounded" title="Delete" @click="openDeleteDialog(m)">
                  <Trash2 class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="text-center py-12 text-gray-500">
      <Activity class="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p>No monitors yet. Create your first uptime monitor.</p>
    </div>

    <!-- Create/Edit Dialog -->
    <div v-if="dialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <h2 class="text-lg font-semibold mb-4">{{ dialogTitle }}</h2>
        <div class="space-y-3">
          <div>
            <label class="block text-sm font-medium mb-1">Name</label>
            <input v-model="formName" class="w-full px-3 py-2 border rounded-md text-sm" placeholder="e.g. Production API" />
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Type</label>
            <select v-model="formType" class="w-full px-3 py-2 border rounded-md text-sm">
              <option v-for="t in typeOptions" :key="t.value" :value="t.value">{{ t.label }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium mb-1">Target</label>
            <input v-model="formTarget" class="w-full px-3 py-2 border rounded-md text-sm" placeholder="https://example.com/health" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-sm font-medium mb-1">Interval (s)</label>
              <input v-model.number="formInterval" type="number" class="w-full px-3 py-2 border rounded-md text-sm" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">Timeout (s)</label>
              <input v-model.number="formTimeout" type="number" class="w-full px-3 py-2 border rounded-md text-sm" />
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-2 mt-5">
          <button class="px-4 py-2 text-sm border rounded-md hover:bg-gray-50" @click="dialogOpen = false">Cancel</button>
          <button class="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700" :disabled="submitting" @click="handleSubmit">
            {{ submitting ? 'Saving...' : editingMonitor ? 'Save' : 'Create' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="deleteDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-sm p-6">
        <h2 class="text-lg font-semibold mb-2">Delete Monitor</h2>
        <p class="text-sm text-gray-600 mb-4">Are you sure you want to delete "{{ deletingMonitor?.name }}"? This cannot be undone.</p>
        <div class="flex justify-end gap-2">
          <button class="px-4 py-2 text-sm border rounded-md hover:bg-gray-50" @click="deleteDialogOpen = false">Cancel</button>
          <button class="px-4 py-2 text-sm bg-red-600 text-white rounded-md hover:bg-red-700" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>

    <!-- SLA Dialog -->
    <div v-if="slaDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <h2 class="text-lg font-semibold mb-4">{{ slaMonitorName }} - SLA Report</h2>
        <div v-if="slaLoading" class="text-center py-4 text-gray-500">Loading...</div>
        <div v-else-if="slaData" class="space-y-3">
          <div class="flex justify-between items-center">
            <span class="text-sm text-gray-500">Uptime</span>
            <span class="text-2xl font-bold" :class="slaData.uptime_pct >= 99 ? 'text-green-600' : 'text-red-600'">
              {{ formatUptime(slaData.uptime_pct) }}
            </span>
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div class="p-3 bg-gray-50 rounded-md">
              <p class="text-xs text-gray-500">Total Checks</p>
              <p class="text-lg font-semibold">{{ slaData.total_checks }}</p>
            </div>
            <div class="p-3 bg-gray-50 rounded-md">
              <p class="text-xs text-gray-500">Successful</p>
              <p class="text-lg font-semibold text-green-600">{{ slaData.up_checks }}</p>
            </div>
            <div class="p-3 bg-gray-50 rounded-md">
              <p class="text-xs text-gray-500">Failed</p>
              <p class="text-lg font-semibold text-red-600">{{ slaData.total_checks - slaData.up_checks }}</p>
            </div>
            <div class="p-3 bg-gray-50 rounded-md">
              <p class="text-xs text-gray-500">Avg Latency</p>
              <p class="text-lg font-semibold">{{ formatLatency(slaData.avg_latency) }}</p>
            </div>
          </div>
        </div>
        <div class="flex justify-end mt-4">
          <button class="px-4 py-2 text-sm border rounded-md hover:bg-gray-50" @click="slaDialogOpen = false">Close</button>
        </div>
      </div>
    </div>

    <!-- Results Dialog -->
    <div v-if="resultsDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-lg p-6">
        <h2 class="text-lg font-semibold mb-4">{{ resultsMonitorName }} - Check History</h2>
        <div v-if="resultsLoading" class="text-center py-4 text-gray-500">Loading...</div>
        <div v-else-if="resultsData.length > 0" class="max-h-64 overflow-auto space-y-1">
          <div v-for="r in resultsData" :key="r.id" class="flex items-center justify-between px-3 py-2 rounded text-sm" :class="r.status === 'up' ? 'bg-green-50' : 'bg-red-50'">
            <div class="flex items-center gap-2">
              <span class="w-2 h-2 rounded-full" :class="r.status === 'up' ? 'bg-green-500' : 'bg-red-500'"></span>
              <span class="text-gray-600">{{ r.message }}</span>
            </div>
            <div class="flex items-center gap-3">
              <span class="text-xs text-gray-400">{{ formatLatency(r.latency) }}</span>
              <span class="text-xs text-gray-400">{{ new Date(r.created_at).toLocaleString() }}</span>
            </div>
          </div>
        </div>
        <div v-else class="text-center py-4 text-gray-500">No check records</div>
        <div class="flex justify-end mt-4">
          <button class="px-4 py-2 text-sm border rounded-md hover:bg-gray-50" @click="resultsDialogOpen = false">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>
