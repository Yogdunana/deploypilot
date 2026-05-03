<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { inject } from 'vue'
import { getTrialStatus, extendTrial, listTrialPeriods } from '@/api/modules/trial'

const { t } = useI18n()
const toast: (msg: string, type?: string) => void = inject<any>('toast')

interface TrialStatus {
  status: string
  machine_id: string
  started_at: string
  expires_at: string
  days_remaining: number
  extended_days: number
  original_days: number
  is_active: boolean
  is_expired: boolean
  is_converted: boolean
}

interface TrialListItem {
  id: string
  machine_id: string
  status: string
  started_at: string
  expires_at: string
  days_remaining: number
  extended_days: number
  original_days: number
  converted_at: string | null
  last_checked_at: string
}

const currentTrial = ref<TrialStatus | null>(null)
const trials = ref<TrialListItem[]>([])
const loading = ref(false)
const activeTab = ref<'current' | 'all'>('current')

// Extend dialog
const extendDialogOpen = ref(false)
const extendForm = ref({
  machine_id: '',
  days: 30,
  reason: '',
})

const statusBadge = (status: string) => {
  const map: Record<string, string> = {
    active: 'bg-green-500/10 border-green-500/20 text-green-400',
    expired: 'bg-red-500/10 border-red-500/20 text-red-400',
    extended: 'bg-blue-500/10 border-blue-500/20 text-blue-400',
    converted: 'bg-purple-500/10 border-purple-500/20 text-purple-400',
  }
  return map[status] || 'bg-gray-500/10 border-gray-500/20 text-gray-400'
}

async function fetchCurrentTrial() {
  loading.value = true
  try {
    const res = await getTrialStatus()
    if (res.data.status === 'success') {
      currentTrial.value = res.data.data
    }
  } catch {
    toast(t('trial.fetchFailed'), 'error')
  } finally {
    loading.value = false
  }
}

async function fetchAllTrials() {
  loading.value = true
  try {
    const res = await listTrialPeriods()
    if (res.data.status === 'success') {
      trials.value = res.data.data.trials || []
    }
  } catch {
    toast(t('trial.fetchFailed'), 'error')
  } finally {
    loading.value = false
  }
}

function switchTab(tab: 'current' | 'all') {
  activeTab.value = tab
  if (tab === 'all') {
    fetchAllTrials()
  }
}

function openExtendDialog(machineId: string) {
  extendForm.value = { machine_id: machineId, days: 30, reason: '' }
  extendDialogOpen.value = true
}

async function saveExtend() {
  try {
    const res = await extendTrial(extendForm.value)
    if (res.data.status === 'success') {
      toast(t('trial.extendSuccess'), 'success')
      extendDialogOpen.value = false
      if (activeTab.value === 'current') {
        fetchCurrentTrial()
      } else {
        fetchAllTrials()
      }
    }
  } catch {
    toast(t('trial.extendFailed'), 'error')
  }
}

onMounted(fetchCurrentTrial)
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold text-white">{{ t('trial.title') }}</h1>
      <p class="mt-1 text-sm text-gray-400">{{ t('trial.description') }}</p>
    </div>

    <!-- Tabs -->
    <div class="flex gap-2">
      <button
        class="px-4 py-2 text-sm rounded-lg border transition-colors"
        :class="activeTab === 'current' ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="switchTab('current')"
      >
        {{ t('trial.currentInstance') }}
      </button>
      <button
        class="px-4 py-2 text-sm rounded-lg border transition-colors"
        :class="activeTab === 'all' ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="switchTab('all')"
      >
        {{ t('trial.allInstances') }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin h-8 w-8 border-2 border-white/20 border-t-white rounded-full" />
    </div>

    <!-- Current Instance -->
    <div v-else-if="activeTab === 'current' && currentTrial" class="space-y-6">
      <!-- Status Card -->
      <div class="bg-gray-900/50 border border-gray-800 rounded-xl p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-white">{{ t('trial.instanceStatus') }}</h2>
          <span class="inline-flex items-center px-3 py-1 rounded-full text-xs border" :class="statusBadge(currentTrial.status)">
            {{ t(`trial.status.${currentTrial.status}`) }}
          </span>
        </div>

        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <p class="text-xs text-gray-500 mb-1">{{ t('trial.machineId') }}</p>
            <code class="text-xs text-gray-300 bg-gray-800 px-2 py-1 rounded block truncate">{{ currentTrial.machine_id }}</code>
          </div>
          <div>
            <p class="text-xs text-gray-500 mb-1">{{ t('trial.startedAt') }}</p>
            <p class="text-sm text-gray-300">{{ new Date(currentTrial.started_at).toLocaleDateString() }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 mb-1">{{ t('trial.expiresAt') }}</p>
            <p class="text-sm text-gray-300">{{ new Date(currentTrial.expires_at).toLocaleDateString() }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 mb-1">{{ t('trial.daysRemaining') }}</p>
            <p class="text-2xl font-bold" :class="currentTrial.is_expired ? 'text-red-400' : currentTrial.days_remaining <= 7 ? 'text-amber-400' : 'text-green-400'">
              {{ currentTrial.is_expired ? '0' : currentTrial.days_remaining }}
            </p>
          </div>
        </div>

        <!-- Progress bar -->
        <div v-if="currentTrial.is_active" class="mt-4">
          <div class="w-full bg-gray-800 rounded-full h-2">
            <div
              class="h-2 rounded-full transition-all"
              :class="currentTrial.days_remaining <= 7 ? 'bg-amber-500' : 'bg-green-500'"
              :style="{ width: Math.min(100, (currentTrial.days_remaining / currentTrial.original_days) * 100) + '%' }"
            />
          </div>
        </div>

        <!-- Info row -->
        <div class="mt-4 flex gap-4 text-xs text-gray-500">
          <span>{{ t('trial.originalDays', { days: currentTrial.original_days }) }}</span>
          <span v-if="currentTrial.extended_days > 0">{{ t('trial.extendedDays', { days: currentTrial.extended_days }) }}</span>
        </div>
      </div>

      <!-- Upgrade CTA -->
      <div v-if="currentTrial.is_active" class="bg-blue-500/5 border border-blue-500/20 rounded-xl p-6 text-center">
        <p class="text-blue-300 mb-2">{{ t('trial.upgradeMessage') }}</p>
        <router-link to="/license" class="inline-block px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white text-sm rounded-lg transition-colors">
          {{ t('trial.activateLicense') }}
        </router-link>
      </div>

      <!-- Expired warning -->
      <div v-if="currentTrial.is_expired" class="bg-red-500/5 border border-red-500/20 rounded-xl p-6 text-center">
        <p class="text-red-300 mb-2">{{ t('trial.expiredMessage') }}</p>
        <router-link to="/license" class="inline-block px-4 py-2 bg-red-600 hover:bg-red-500 text-white text-sm rounded-lg transition-colors">
          {{ t('trial.activateLicense') }}
        </router-link>
      </div>
    </div>

    <!-- No trial (community mode) -->
    <div v-else-if="activeTab === 'current' && currentTrial && currentTrial.status === 'none'" class="bg-gray-900/50 border border-gray-800 rounded-xl p-8 text-center">
      <p class="text-gray-400">{{ t('trial.noTrial') }}</p>
    </div>

    <!-- All Instances Table -->
    <div v-else-if="activeTab === 'all'" class="bg-gray-900/50 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800">
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('trial.machineId') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('trial.status') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('trial.startedAt') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('trial.expiresAt') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('trial.daysRemaining') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('trial.extended') }}</th>
            <th class="text-right px-4 py-3 text-gray-400 font-medium">{{ t('trial.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="trial in trials"
            :key="trial.id"
            class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
          >
            <td class="px-4 py-3">
              <code class="text-xs bg-gray-800 px-2 py-1 rounded text-gray-300">{{ trial.machine_id }}</code>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs border" :class="statusBadge(trial.status)">
                {{ t(`trial.status.${trial.status}`) }}
              </span>
            </td>
            <td class="px-4 py-3 text-gray-300 text-xs">{{ new Date(trial.started_at).toLocaleDateString() }}</td>
            <td class="px-4 py-3 text-gray-300 text-xs">{{ new Date(trial.expires_at).toLocaleDateString() }}</td>
            <td class="px-4 py-3 text-gray-300">{{ trial.days_remaining }}</td>
            <td class="px-4 py-3 text-gray-400 text-xs">{{ trial.extended_days > 0 ? trial.extended_days + 'd' : '-' }}</td>
            <td class="px-4 py-3 text-right">
              <button
                v-if="trial.status !== 'converted'"
                class="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                @click="openExtendDialog(trial.machine_id)"
              >
                {{ t('trial.extend') }}
              </button>
            </td>
          </tr>
          <tr v-if="trials.length === 0">
            <td colspan="7" class="px-4 py-8 text-center text-gray-500">{{ t('trial.noTrials') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Extend Dialog -->
    <Teleport to="body">
      <div v-if="extendDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="extendDialogOpen = false">
        <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
          <h2 class="text-lg font-semibold text-white mb-4">{{ t('trial.extendTrial') }}</h2>
          <div class="space-y-4">
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('trial.machineId') }}</label>
              <input v-model="extendForm.machine_id" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" disabled />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('trial.extensionDays') }}</label>
              <input v-model.number="extendForm.days" type="number" min="1" max="365" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('trial.reason') }}</label>
              <textarea v-model="extendForm.reason" rows="2" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white resize-none" :placeholder="t('trial.reasonPlaceholder')" />
            </div>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <button class="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors" @click="extendDialogOpen = false">
              {{ t('common.cancel') }}
            </button>
            <button class="px-4 py-2 text-sm bg-blue-600 hover:bg-blue-500 text-white rounded-lg transition-colors" @click="saveExtend">
              {{ t('trial.extend') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
