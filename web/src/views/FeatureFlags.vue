<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { inject } from 'vue'
import type { Ref } from 'vue'
import { getFeatureFlags, updateFeatureFlag, setFeatureFlagOverride, deleteFeatureFlagOverride, getFeatureFlagOverrides } from '@/api/modules/featureFlags'

const { t } = useI18n()
const toast: (msg: string, type?: string) => void = inject<any>('toast')

interface FeatureFlag {
  id: string
  key: string
  name: string
  description: string
  status: string
  default_enabled: boolean
  required_tier: string
  required_use_type: string
  category: string
  enabled: boolean
  overridden_by: string
}

interface Override {
  id: string
  flag_key: string
  tenant_id: string
  enabled: boolean
  reason: string
  overridden_by: string
  created_at: string
  updated_at: string
}

const flags = ref<FeatureFlag[]>([])
const loading = ref(false)
const selectedFlag = ref<FeatureFlag | null>(null)
const overrides = ref<Override[]>([])
const overridesLoading = ref(false)

// Edit dialog
const editDialogOpen = ref(false)
const editForm = ref({
  name: '',
  description: '',
  status: 'enabled',
  default_enabled: true,
  required_tier: '',
  required_use_type: '',
  category: '',
})

// Override dialog
const overrideDialogOpen = ref(false)
const overrideForm = ref({
  tenant_id: '',
  enabled: true,
  reason: '',
})

// Category filter
const categoryFilter = ref('all')
const categories = computed(() => {
  const cats = new Set(flags.value.map(f => f.category))
  return Array.from(cats).sort()
})

const filteredFlags = computed(() => {
  if (categoryFilter.value === 'all') return flags.value
  return flags.value.filter(f => f.category === categoryFilter.value)
})

const statusColor = (status: string) => {
  return status === 'enabled' ? 'text-green-400' : 'text-red-400'
}

const statusBg = (status: string) => {
  return status === 'enabled' ? 'bg-green-500/10 border-green-500/20' : 'bg-red-500/10 border-red-500/20'
}

const tierBadge = (tier: string) => {
  const map: Record<string, string> = {
    community: 'bg-gray-500/10 text-gray-400',
    team: 'bg-blue-500/10 text-blue-400',
    pro: 'bg-purple-500/10 text-purple-400',
    enterprise: 'bg-amber-500/10 text-amber-400',
  }
  return map[tier] || 'bg-gray-500/10 text-gray-400'
}

async function fetchFlags() {
  loading.value = true
  try {
    const res = await getFeatureFlags()
    if (res.data.status === 'success') {
      flags.value = res.data.data.flags || []
    }
  } catch {
    toast(t('featureFlags.fetchFailed'), 'error')
  } finally {
    loading.value = false
  }
}

async function fetchOverrides(flagKey: string) {
  overridesLoading.value = true
  try {
    const res = await getFeatureFlagOverrides(flagKey)
    if (res.data.status === 'success') {
      overrides.value = res.data.data.overrides || []
    }
  } catch {
    overrides.value = []
  } finally {
    overridesLoading.value = false
  }
}

function openEditDialog(flag: FeatureFlag) {
  selectedFlag.value = flag
  editForm.value = {
    name: flag.name,
    description: flag.description,
    status: flag.status,
    default_enabled: flag.default_enabled,
    required_tier: flag.required_tier,
    required_use_type: flag.required_use_type,
    category: flag.category,
  }
  editDialogOpen.value = true
}

async function saveFlag() {
  if (!selectedFlag.value) return
  try {
    const res = await updateFeatureFlag(selectedFlag.value.key, editForm.value)
    if (res.data.status === 'success') {
      toast(t('featureFlags.updated'), 'success')
      editDialogOpen.value = false
      await fetchFlags()
    }
  } catch {
    toast(t('featureFlags.updateFailed'), 'error')
  }
}

function openOverrideDialog(flag: FeatureFlag) {
  selectedFlag.value = flag
  overrideForm.value = { tenant_id: '', enabled: true, reason: '' }
  overrideDialogOpen.value = true
}

async function saveOverride() {
  if (!selectedFlag.value || !overrideForm.value.tenant_id) return
  try {
    const res = await setFeatureFlagOverride(selectedFlag.value.key, overrideForm.value)
    if (res.data.status === 'success') {
      toast(t('featureFlags.overrideSet'), 'success')
      overrideDialogOpen.value = false
      await fetchOverrides(selectedFlag.value.key)
      await fetchFlags()
    }
  } catch {
    toast(t('featureFlags.overrideFailed'), 'error')
  }
}

async function removeOverride(flagKey: string, tenantId: string) {
  try {
    const res = await deleteFeatureFlagOverride(flagKey, tenantId)
    if (res.data.status === 'success') {
      toast(t('featureFlags.overrideDeleted'), 'success')
      await fetchOverrides(flagKey)
      await fetchFlags()
    }
  } catch {
    toast(t('featureFlags.deleteFailed'), 'error')
  }
}

function showOverrides(flag: FeatureFlag) {
  selectedFlag.value = flag
  fetchOverrides(flag.key)
}

onMounted(fetchFlags)
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('featureFlags.title') }}</h1>
        <p class="mt-1 text-sm text-gray-400">{{ t('featureFlags.description') }}</p>
      </div>
      <div class="text-sm text-gray-400">
        {{ t('featureFlags.totalFlags', { count: flags.length }) }}
      </div>
    </div>

    <!-- Category Filter -->
    <div class="flex gap-2 flex-wrap">
      <button
        class="px-3 py-1.5 text-xs rounded-lg border transition-colors"
        :class="categoryFilter === 'all' ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="categoryFilter = 'all'"
      >
        {{ t('featureFlags.allCategories') }}
      </button>
      <button
        v-for="cat in categories"
        :key="cat"
        class="px-3 py-1.5 text-xs rounded-lg border transition-colors capitalize"
        :class="categoryFilter === cat ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'"
        @click="categoryFilter = cat"
      >
        {{ t(`featureFlags.category.${cat}`) || cat }}
      </button>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin h-8 w-8 border-2 border-white/20 border-t-white rounded-full" />
    </div>

    <!-- Flags Table -->
    <div v-else class="bg-gray-900/50 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800">
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('featureFlags.key') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('featureFlags.name') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('featureFlags.category') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('featureFlags.status') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('featureFlags.requiredTier') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('featureFlags.enabled') }}</th>
            <th class="text-right px-4 py-3 text-gray-400 font-medium">{{ t('featureFlags.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="flag in filteredFlags"
            :key="flag.id"
            class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors"
          >
            <td class="px-4 py-3">
              <code class="text-xs bg-gray-800 px-2 py-1 rounded text-gray-300">{{ flag.key }}</code>
            </td>
            <td class="px-4 py-3 text-gray-300">{{ flag.name }}</td>
            <td class="px-4 py-3">
              <span class="text-xs capitalize text-gray-400">{{ t(`featureFlags.category.${flag.category}`) || flag.category }}</span>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs border" :class="statusBg(flag.status)">
                <span :class="statusColor(flag.status)">{{ t(`featureFlags.${flag.status}`) }}</span>
              </span>
            </td>
            <td class="px-4 py-3">
              <span v-if="flag.required_tier" class="inline-flex items-center px-2 py-0.5 rounded text-xs" :class="tierBadge(flag.required_tier)">
                {{ flag.required_tier }}
              </span>
              <span v-else class="text-xs text-gray-500">{{ t('featureFlags.any') }}</span>
            </td>
            <td class="px-4 py-3">
              <span v-if="flag.enabled" class="text-green-400">● {{ t('featureFlags.on') }}</span>
              <span v-else class="text-red-400">● {{ t('featureFlags.off') }}</span>
            </td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-2">
                <button
                  class="text-xs text-gray-400 hover:text-white transition-colors"
                  @click="showOverrides(flag)"
                >
                  {{ t('featureFlags.overrides') }}
                </button>
                <button
                  class="text-xs text-blue-400 hover:text-blue-300 transition-colors"
                  @click="openEditDialog(flag)"
                >
                  {{ t('featureFlags.edit') }}
                </button>
                <button
                  class="text-xs text-purple-400 hover:text-purple-300 transition-colors"
                  @click="openOverrideDialog(flag)"
                >
                  {{ t('featureFlags.setOverride') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Overrides Panel -->
    <div v-if="selectedFlag" class="bg-gray-900/50 border border-gray-800 rounded-xl p-4">
      <div class="flex items-center justify-between mb-4">
        <h3 class="text-sm font-medium text-white">
          {{ t('featureFlags.overridesFor', { key: selectedFlag.key }) }}
        </h3>
        <button class="text-xs text-gray-400 hover:text-white" @click="selectedFlag = null; overrides = []">
          ✕
        </button>
      </div>
      <div v-if="overridesLoading" class="flex justify-center py-4">
        <div class="animate-spin h-5 w-5 border-2 border-white/20 border-t-white rounded-full" />
      </div>
      <div v-else-if="overrides.length === 0" class="text-center py-4 text-sm text-gray-500">
        {{ t('featureFlags.noOverrides') }}
      </div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800">
            <th class="text-left px-3 py-2 text-gray-400 font-medium text-xs">{{ t('featureFlags.tenantId') }}</th>
            <th class="text-left px-3 py-2 text-gray-400 font-medium text-xs">{{ t('featureFlags.enabled') }}</th>
            <th class="text-left px-3 py-2 text-gray-400 font-medium text-xs">{{ t('featureFlags.reason') }}</th>
            <th class="text-left px-3 py-2 text-gray-400 font-medium text-xs">{{ t('featureFlags.overriddenBy') }}</th>
            <th class="text-right px-3 py-2 text-gray-400 font-medium text-xs">{{ t('featureFlags.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="o in overrides" :key="o.id" class="border-b border-gray-800/50">
            <td class="px-3 py-2 text-gray-300 text-xs">{{ o.tenant_id }}</td>
            <td class="px-3 py-2">
              <span :class="o.enabled ? 'text-green-400' : 'text-red-400'" class="text-xs">
                {{ o.enabled ? t('featureFlags.on') : t('featureFlags.off') }}
              </span>
            </td>
            <td class="px-3 py-2 text-gray-400 text-xs">{{ o.reason || '-' }}</td>
            <td class="px-3 py-2 text-gray-400 text-xs">{{ o.overridden_by }}</td>
            <td class="px-3 py-2 text-right">
              <button
                class="text-xs text-red-400 hover:text-red-300"
                @click="removeOverride(o.flag_key, o.tenant_id)"
              >
                {{ t('featureFlags.delete') }}
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Edit Dialog -->
    <Teleport to="body">
      <div v-if="editDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="editDialogOpen = false">
        <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-lg mx-4">
          <h2 class="text-lg font-semibold text-white mb-4">{{ t('featureFlags.editFlag') }}</h2>
          <div class="space-y-4">
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.name') }}</label>
              <input v-model="editForm.name" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.description') }}</label>
              <textarea v-model="editForm.description" rows="2" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white resize-none" />
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.status') }}</label>
                <select v-model="editForm.status" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white">
                  <option value="enabled">{{ t('featureFlags.enabled') }}</option>
                  <option value="disabled">{{ t('featureFlags.disabled') }}</option>
                </select>
              </div>
              <div>
                <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.category') }}</label>
                <select v-model="editForm.category" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white">
                  <option value="general">General</option>
                  <option value="infrastructure">Infrastructure</option>
                  <option value="monitoring">Monitoring</option>
                  <option value="integration">Integration</option>
                  <option value="security">Security</option>
                  <option value="management">Management</option>
                  <option value="tools">Tools</option>
                  <option value="commercial">Commercial</option>
                </select>
              </div>
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.requiredTier') }}</label>
                <select v-model="editForm.required_tier" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white">
                  <option value="">{{ t('featureFlags.any') }}</option>
                  <option value="community">Community</option>
                  <option value="team">Team</option>
                  <option value="pro">Pro</option>
                  <option value="enterprise">Enterprise</option>
                </select>
              </div>
              <div>
                <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.requiredUseType') }}</label>
                <select v-model="editForm.required_use_type" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white">
                  <option value="">{{ t('featureFlags.any') }}</option>
                  <option value="non_commercial">{{ t('featureFlags.nonCommercial') }}</option>
                  <option value="commercial">{{ t('featureFlags.commercial') }}</option>
                </select>
              </div>
            </div>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <button class="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors" @click="editDialogOpen = false">
              {{ t('common.cancel') }}
            </button>
            <button class="px-4 py-2 text-sm bg-blue-600 hover:bg-blue-500 text-white rounded-lg transition-colors" @click="saveFlag">
              {{ t('common.save') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Override Dialog -->
    <Teleport to="body">
      <div v-if="overrideDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="overrideDialogOpen = false">
        <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
          <h2 class="text-lg font-semibold text-white mb-4">{{ t('featureFlags.setOverride') }}</h2>
          <div class="space-y-4">
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.tenantId') }}</label>
              <input v-model="overrideForm.tenant_id" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" :placeholder="t('featureFlags.tenantIdPlaceholder')" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.enabled') }}</label>
              <select v-model="overrideForm.enabled" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white">
                <option :value="true">{{ t('featureFlags.on') }}</option>
                <option :value="false">{{ t('featureFlags.off') }}</option>
              </select>
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('featureFlags.reason') }}</label>
              <textarea v-model="overrideForm.reason" rows="2" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white resize-none" :placeholder="t('featureFlags.reasonPlaceholder')" />
            </div>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <button class="px-4 py-2 text-sm text-gray-400 hover:text-white transition-colors" @click="overrideDialogOpen = false">
              {{ t('common.cancel') }}
            </button>
            <button class="px-4 py-2 text-sm bg-purple-600 hover:bg-purple-500 text-white rounded-lg transition-colors" @click="saveOverride">
              {{ t('common.save') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
