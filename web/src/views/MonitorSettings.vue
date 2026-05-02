<script setup lang="ts">
import { ref } from 'vue'
import PageHeader from '@/components/common/PageHeader.vue'
import { Save, RotateCcw } from 'lucide-vue-next'

const defaultInterval = ref(60)
const defaultTimeout = ref(10)
const defaultRetries = ref(3)
const heartbeatDefaultTimeout = ref(120)
const schedulerEnabled = ref(true)
const metricsPublic = ref(false)
const saving = ref(false)
const saved = ref(false)

async function saveSettings() {
  saving.value = true
  saved.value = false
  await new Promise(r => setTimeout(r, 500))
  saving.value = false
  saved.value = true
  setTimeout(() => { saved.value = false }, 3000)
}

function resetDefaults() {
  defaultInterval.value = 60
  defaultTimeout.value = 10
  defaultRetries.value = 3
  heartbeatDefaultTimeout.value = 120
  schedulerEnabled.value = true
  metricsPublic.value = false
}
</script>

<template>
  <div class="space-y-6">
    <PageHeader title="Monitoring Settings" description="Configure check intervals, thresholds, and scheduler behavior">
      <template #actions>
        <button class="px-3 py-2 text-sm border rounded-md hover:bg-gray-50 flex items-center gap-1" @click="resetDefaults">
          <RotateCcw class="w-4 h-4" /> Reset
        </button>
        <button class="px-3 py-2 text-sm bg-blue-600 text-white rounded-md hover:bg-blue-700 flex items-center gap-1" :disabled="saving" @click="saveSettings">
          <Save class="w-4 h-4" /> {{ saving ? 'Saving...' : 'Save' }}
        </button>
      </template>
    </PageHeader>
    <div v-if="saved" class="p-3 bg-green-50 border border-green-200 rounded-lg text-sm text-green-700">Settings saved successfully.</div>
    <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
      <h3 class="text-base font-semibold mb-4">Uptime Monitor Defaults</h3>
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div>
          <label class="block text-sm font-medium mb-1">Default Check Interval (seconds)</label>
          <input v-model.number="defaultInterval" type="number" min="10" max="3600" class="w-full px-3 py-2 border rounded-md text-sm" />
          <p class="text-xs text-gray-500 mt-1">How often to check each monitor (10-3600s)</p>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Default Timeout (seconds)</label>
          <input v-model.number="defaultTimeout" type="number" min="1" max="60" class="w-full px-3 py-2 border rounded-md text-sm" />
          <p class="text-xs text-gray-500 mt-1">Request timeout before marking as failed (1-60s)</p>
        </div>
        <div>
          <label class="block text-sm font-medium mb-1">Default Retries</label>
          <input v-model.number="defaultRetries" type="number" min="1" max="10" class="w-full px-3 py-2 border rounded-md text-sm" />
          <p class="text-xs text-gray-500 mt-1">Consecutive failures before alerting (1-10)</p>
        </div>
      </div>
    </div>
    <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
      <h3 class="text-base font-semibold mb-4">Heartbeat Defaults</h3>
      <div class="max-w-md">
        <label class="block text-sm font-medium mb-1">Default Heartbeat Timeout (seconds)</label>
        <input v-model.number="heartbeatDefaultTimeout" type="number" min="30" max="86400" class="w-full px-3 py-2 border rounded-md text-sm" />
        <p class="text-xs text-gray-500 mt-1">Alert after no heartbeat received for this duration (30-86400s)</p>
      </div>
    </div>
    <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
      <h3 class="text-base font-semibold mb-4">Scheduler</h3>
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium">Enable Background Scheduler</p>
          <p class="text-xs text-gray-500 mt-0.5">Automatically run periodic monitor checks and heartbeat timeout detection</p>
        </div>
        <button class="relative w-11 h-6 rounded-full transition-colors" :class="schedulerEnabled ? 'bg-blue-600' : 'bg-gray-300'" @click="schedulerEnabled = !schedulerEnabled">
          <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform" :class="schedulerEnabled ? 'translate-x-5' : ''"></span>
        </button>
      </div>
    </div>
    <div class="p-6 bg-white border border-gray-200 rounded-xl shadow-sm">
      <h3 class="text-base font-semibold mb-4">Prometheus Metrics</h3>
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm font-medium">Public Metrics Endpoint</p>
          <p class="text-xs text-gray-500 mt-0.5">Allow unauthenticated access to /metrics for Prometheus scraping. When disabled, JWT authentication is required.</p>
        </div>
        <button class="relative w-11 h-6 rounded-full transition-colors" :class="metricsPublic ? 'bg-blue-600' : 'bg-gray-300'" @click="metricsPublic = !metricsPublic">
          <span class="absolute top-0.5 left-0.5 w-5 h-5 bg-white rounded-full shadow transition-transform" :class="metricsPublic ? 'translate-x-5' : ''"></span>
        </button>
      </div>
    </div>
  </div>
</template>
