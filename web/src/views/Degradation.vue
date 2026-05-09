<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { useI18n } from 'vue-i18n'

import { getDegradationStatus, getDegradationAudits, getExportSummary } from '@/api/modules/degradation'

const { t } = useI18n()
const { toast } = useToast()

interface DegradationStatus {
  level: string
  license_status: string
  trial_status: string
  tier: string
  gated_features: string[]
  read_only_reason: string
  expires_at: string
  grace_days_left: number
}

interface AuditEntry {
  id: string
  action: string
  feature: string
  reason: string
  tenant_id: string
  user_id: string
  ip_address: string
  created_at: string
}

interface ExportSummary {
  apps: number
  servers: number
  deployments: number
  licenses: number
  users: number
  exported_at: string
  machine_id: string
}

const status = ref<DegradationStatus | null>(null)
const audits = ref<AuditEntry[]>([])
const exportSummary = ref<ExportSummary | null>(null)
const loading = ref(false)
const activeTab = ref<'status' | 'audits' | 'export'>('status')

const levelBadge = (level: string) => {
  const map: Record<string, string> = {
    none: 'bg-green-500/10 border-green-500/20 text-green-400',
    partial: 'bg-amber-500/10 border-amber-500/20 text-amber-400',
    readonly: 'bg-red-500/10 border-red-500/20 text-red-400',
  }
  return map[level] || 'bg-gray-500/10 border-gray-500/20 text-gray-400'
}

const actionBadge = (action: string) => {
  const map: Record<string, string> = {
    feature_gated: 'bg-purple-500/10 text-purple-400',
    read_only_blocked: 'bg-red-500/10 text-red-400',
    tier_downgrade: 'bg-amber-500/10 text-amber-400',
    data_export: 'bg-blue-500/10 text-blue-400',
  }
  return map[action] || 'bg-gray-500/10 text-gray-400'
}

async function fetchStatus() {
  loading.value = true
  try {
    const res = await getDegradationStatus()
    if (res.data.status === 'success') {
      status.value = res.data.data
    }
  } catch {
    toast(t('degradation.fetchFailed'), 'error')
  } finally {
    loading.value = false
  }
}

async function fetchAudits() {
  loading.value = true
  try {
    const res = await getDegradationAudits(100)
    if (res.data.status === 'success') {
      audits.value = res.data.data.audits || []
    }
  } catch {
    toast(t('degradation.fetchFailed'), 'error')
  } finally {
    loading.value = false
  }
}

async function fetchExportSummary() {
  loading.value = true
  try {
    const res = await getExportSummary()
    if (res.data.status === 'success') {
      exportSummary.value = res.data.data
    }
  } catch {
    toast(t('degradation.fetchFailed'), 'error')
  } finally {
    loading.value = false
  }
}

function switchTab(tab: 'status' | 'audits' | 'export') {
  activeTab.value = tab
  if (tab === 'status') fetchStatus()
  else if (tab === 'audits') fetchAudits()
  else fetchExportSummary()
}

onMounted(fetchStatus)
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold text-white">{{ t('degradation.title') }}</h1>
      <p class="mt-1 text-sm text-gray-400">{{ t('degradation.description') }}</p>
    </div>

    <!-- Tabs -->
    <div class="flex gap-2">
      <button
        class="px-4 py-2 text-sm rounded-lg border transition-colors"
        :class="activeTab === 'status' ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="switchTab('status')"
      >
        {{ t('degradation.status') }}
      </button>
      <button
        class="px-4 py-2 text-sm rounded-lg border transition-colors"
        :class="activeTab === 'audits' ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="switchTab('audits')"
      >
        {{ t('degradation.auditTrail') }}
      </button>
      <button
        class="px-4 py-2 text-sm rounded-lg border transition-colors"
        :class="activeTab === 'export' ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="switchTab('export')"
      >
        {{ t('degradation.dataExport') }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin h-8 w-8 border-2 border-white/20 border-t-white rounded-full" />
    </div>

    <!-- Status Tab -->
    <div v-else-if="activeTab === 'status' && status" class="space-y-6">
      <!-- Level Card -->
      <div class="bg-gray-900/50 border border-gray-800 rounded-xl p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold text-white">{{ t('degradation.currentLevel') }}</h2>
          <span class="inline-flex items-center px-3 py-1 rounded-full text-xs border" :class="levelBadge(status.level)">
            {{ t(`degradation.level.${status.level}`) }}
          </span>
        </div>

        <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div>
            <p class="text-xs text-gray-500 mb-1">{{ t('degradation.licenseStatus') }}</p>
            <p class="text-sm text-gray-300 capitalize">{{ status.license_status }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 mb-1">{{ t('degradation.trialStatus') }}</p>
            <p class="text-sm text-gray-300 capitalize">{{ status.trial_status }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 mb-1">{{ t('degradation.tier') }}</p>
            <p class="text-sm text-gray-300 capitalize">{{ status.tier }}</p>
          </div>
          <div>
            <p class="text-xs text-gray-500 mb-1">{{ t('degradation.graceDays') }}</p>
            <p class="text-sm" :class="status.grace_days_left <= 7 ? 'text-amber-400' : 'text-green-400'">
              {{ status.grace_days_left }}
            </p>
          </div>
        </div>

        <div v-if="status.expires_at" class="mt-4 text-xs text-gray-500">
          {{ t('degradation.expiresAt') }}: {{ new Date(status.expires_at).toLocaleString() }}
        </div>
      </div>

      <!-- Read-only Warning -->
      <div v-if="status.level === 'readonly'" class="bg-red-500/5 border border-red-500/20 rounded-xl p-6">
        <p class="text-red-300">{{ status.read_only_reason }}</p>
      </div>

      <!-- Partial Warning -->
      <div v-else-if="status.level === 'partial' && status.read_only_reason" class="bg-amber-500/5 border border-amber-500/20 rounded-xl p-6">
        <p class="text-amber-300">{{ status.read_only_reason }}</p>
      </div>

      <!-- Gated Features -->
      <div v-if="status.gated_features.length > 0" class="bg-gray-900/50 border border-gray-800 rounded-xl p-6">
        <h3 class="text-sm font-medium text-white mb-3">{{ t('degradation.gatedFeatures') }} ({{ status.gated_features.length }})</h3>
        <div class="flex flex-wrap gap-2">
          <span v-for="f in status.gated_features" :key="f" class="inline-flex items-center px-2 py-1 rounded text-xs bg-gray-800 text-gray-400">
            {{ f }}
          </span>
        </div>
      </div>

      <!-- No Degradation -->
      <div v-else class="bg-green-500/5 border border-green-500/20 rounded-xl p-6 text-center">
        <p class="text-green-300">{{ t('degradation.noDegradation') }}</p>
      </div>
    </div>

    <!-- Audits Tab -->
    <div v-else-if="activeTab === 'audits'" class="bg-gray-900/50 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800">
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('degradation.time') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('degradation.action') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('degradation.feature') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('degradation.reason') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('degradation.user') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('degradation.ip') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in audits" :key="a.id" class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors">
            <td class="px-4 py-3 text-gray-400 text-xs">{{ new Date(a.created_at).toLocaleString() }}</td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs" :class="actionBadge(a.action)">
                {{ a.action }}
              </span>
            </td>
            <td class="px-4 py-3 text-gray-300 text-xs">{{ a.feature }}</td>
            <td class="px-4 py-3 text-gray-400 text-xs">{{ a.reason }}</td>
            <td class="px-4 py-3 text-gray-400 text-xs">{{ a.user_id || '-' }}</td>
            <td class="px-4 py-3 text-gray-400 text-xs">{{ a.ip_address || '-' }}</td>
          </tr>
          <tr v-if="audits.length === 0">
            <td colspan="6" class="px-4 py-8 text-center text-gray-500">{{ t('degradation.noAudits') }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Export Tab -->
    <div v-else-if="activeTab === 'export' && exportSummary" class="space-y-6">
      <div class="bg-gray-900/50 border border-gray-800 rounded-xl p-6">
        <h2 class="text-lg font-semibold text-white mb-4">{{ t('degradation.exportSummary') }}</h2>
        <div class="grid grid-cols-2 md:grid-cols-5 gap-4">
          <div class="bg-gray-800/50 rounded-lg p-4 text-center">
            <p class="text-2xl font-bold text-white">{{ exportSummary.apps }}</p>
            <p class="text-xs text-gray-400 mt-1">{{ t('degradation.apps') }}</p>
          </div>
          <div class="bg-gray-800/50 rounded-lg p-4 text-center">
            <p class="text-2xl font-bold text-white">{{ exportSummary.servers }}</p>
            <p class="text-xs text-gray-400 mt-1">{{ t('degradation.servers') }}</p>
          </div>
          <div class="bg-gray-800/50 rounded-lg p-4 text-center">
            <p class="text-2xl font-bold text-white">{{ exportSummary.deployments }}</p>
            <p class="text-xs text-gray-400 mt-1">{{ t('degradation.deployments') }}</p>
          </div>
          <div class="bg-gray-800/50 rounded-lg p-4 text-center">
            <p class="text-2xl font-bold text-white">{{ exportSummary.licenses }}</p>
            <p class="text-xs text-gray-400 mt-1">{{ t('degradation.licenses') }}</p>
          </div>
          <div class="bg-gray-800/50 rounded-lg p-4 text-center">
            <p class="text-2xl font-bold text-white">{{ exportSummary.users }}</p>
            <p class="text-xs text-gray-400 mt-1">{{ t('degradation.users') }}</p>
          </div>
        </div>
        <div class="mt-4 text-xs text-gray-500">
          {{ t('degradation.exportedAt') }}: {{ new Date(exportSummary.exported_at).toLocaleString() }}
        </div>
      </div>
    </div>
  </div>
</template>
