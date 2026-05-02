<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Heart, Plus, Trash2, Copy, ExternalLink, MoreHorizontal } from 'lucide-vue-next'
import * as uptimeApi from '@/api/modules/uptime'
import type { HeartbeatMonitor } from '@/types/monitor'

const heartbeats = ref<HeartbeatMonitor[]>([])
const loading = ref(false)

const dialogOpen = ref(false)
const formName = ref('')
const formInterval = ref(60)
const formTimeout = ref(120)
const submitting = ref(false)
const createdToken = ref('')

const deleteDialogOpen = ref(false)
const deletingHeartbeat = ref<HeartbeatMonitor | null>(null)
const deleting = ref(false)

function formatTime(date?: string): string {
  if (!date) return 'Never'
  return new Date(date).toLocaleString()
}

function copyToken(token: string) {
  navigator.clipboard.writeText(token)
}

function getPingURL(token: string): string {
  return `${window.location.origin}/api/v1/heartbeat/ping/${token}`
}

async function fetchHeartbeats() {
  loading.value = true
  try {
    const res = await uptimeApi.listHeartbeats()
    if (res.data.status === 'success') {
      heartbeats.value = res.data.data
    }
  } catch {
    /* handled silently */
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  formName.value = ''
  formInterval.value = 60
  formTimeout.value = 120
  createdToken.value = ''
  dialogOpen.value = true
}

async function handleSubmit() {
  if (!formName.value.trim()) return
  submitting.value = true
  try {
    const res = await uptimeApi.createHeartbeat({
      name: formName.value.trim(),
      interval: formInterval.value,
      timeout: formTimeout.value,
    })
    if (res.data.status === 'success') {
      createdToken.value = res.data.data.token
      fetchHeartbeats()
    }
  } catch {
    /* handled silently */
  } finally {
    submitting.value = false
  }
}

function openDeleteDialog(hb: HeartbeatMonitor) {
  deletingHeartbeat.value = hb
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingHeartbeat.value) return
  deleting.value = true
  try {
    await uptimeApi.deleteHeartbeat(deletingHeartbeat.value.id)
    fetchHeartbeats()
  } catch {
    /* handled silently */
  } finally {
    deleting.value = false
    deletingHeartbeat.value = null
  }
}

onMounted(fetchHeartbeats)
</script>

<template>
  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold">Heartbeats</h1>
      <button class="px-3 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700" @click="openCreateDialog">
        + Create
      </button>
    </div>

    <div class="p-3 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-700">
      Heartbeat monitors expect your application to call a unique ping URL at regular intervals.
      If no ping is received within the timeout period, the monitor will be marked as down.
    </div>

    <div v-if="loading" class="text-center py-8 text-gray-500">Loading...</div>

    <div v-else-if="heartbeats.length > 0" class="border rounded-lg overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-gray-50 border-b">
          <tr>
            <th class="px-4 py-3 text-left">Name</th>
            <th class="px-4 py-3 text-left">Status</th>
            <th class="px-4 py-3 text-left">Interval</th>
            <th class="px-4 py-3 text-left">Last Beat</th>
            <th class="px-4 py-3 text-left">Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="hb in heartbeats" :key="hb.id" class="border-b last:border-0 hover:bg-gray-50">
            <td class="px-4 py-3 font-medium">{{ hb.name }}</td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full" :class="hb.status === 'up' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'">
                <span class="w-1.5 h-1.5 rounded-full" :class="hb.status === 'up' ? 'bg-green-500' : 'bg-red-500'"></span>
                {{ hb.status === 'up' ? 'UP' : 'DOWN' }}
              </span>
            </td>
            <td class="px-4 py-3 text-gray-500">{{ hb.interval }}s</td>
            <td class="px-4 py-3 text-gray-500">{{ formatTime(hb.last_beat) }}</td>
            <td class="px-4 py-3">
              <div class="flex gap-1">
                <button class="p-1 hover:bg-gray-100 rounded" title="Copy URL" @click="copyToken(getPingURL(hb.token))">
                  <Copy class="w-4 h-4" />
                </button>
                <button class="p-1 hover:bg-gray-100 rounded" title="Open URL" @click="window.open(getPingURL(hb.token), '_blank')">
                  <ExternalLink class="w-4 h-4" />
                </button>
                <button class="p-1 hover:bg-red-50 text-red-600 rounded" title="Delete" @click="openDeleteDialog(hb)">
                  <Trash2 class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else class="text-center py-12 text-gray-500">
      <Heart class="w-12 h-12 mx-auto mb-3 text-gray-300" />
      <p>No heartbeats yet. Create your first heartbeat monitor.</p>
    </div>

    <!-- Create Dialog -->
    <div v-if="dialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <h2 class="text-lg font-semibold mb-4">Create Heartbeat</h2>
        <div v-if="createdToken" class="space-y-3">
          <div class="p-3 bg-green-50 border border-green-200 rounded-lg">
            <p class="text-sm font-medium text-green-700 mb-1">Heartbeat created!</p>
            <p class="text-xs text-green-600 mb-2">Call this URL at regular intervals:</p>
            <div class="flex items-center gap-2">
              <code class="flex-1 text-xs bg-white px-2 py-1 rounded border break-all">{{ getPingURL(createdToken) }}</code>
              <button class="p-1 hover:bg-green-100 rounded" @click="copyToken(getPingURL(createdToken))">
                <Copy class="w-4 h-4 text-green-600" />
              </button>
            </div>
          </div>
          <div class="flex justify-end">
            <button class="px-4 py-2 text-sm border rounded-md hover:bg-gray-50" @click="dialogOpen = false">Done</button>
          </div>
        </div>
        <div v-else class="space-y-3">
          <div>
            <label class="block text-sm font-medium mb-1">Name</label>
            <input v-model="formName" class="w-full px-3 py-2 border rounded-md text-sm" placeholder="e.g. Cron Job" />
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
          <div class="flex justify-end gap-2">
            <button class="px-4 py-2 text-sm border rounded-md hover:bg-gray-50" @click="dialogOpen = false">Cancel</button>
            <button class="px-4 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700" :disabled="submitting" @click="handleSubmit">
              {{ submitting ? 'Creating...' : 'Create' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Delete Confirmation -->
    <div v-if="deleteDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-white rounded-lg shadow-xl w-full max-w-sm p-6">
        <h2 class="text-lg font-semibold mb-2">Delete Heartbeat</h2>
        <p class="text-sm text-gray-600 mb-4">Are you sure you want to delete "{{ deletingHeartbeat?.name }}"?</p>
        <div class="flex justify-end gap-2">
          <button class="px-4 py-2 text-sm border rounded-md hover:bg-gray-50" @click="deleteDialogOpen = false">Cancel</button>
          <button class="px-4 py-2 text-sm bg-red-600 text-white rounded-md hover:bg-red-700" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? 'Deleting...' : 'Delete' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
