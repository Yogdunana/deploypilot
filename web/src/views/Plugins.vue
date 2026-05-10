<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useToast } from '@/composables/useToast'
import { useI18n } from 'vue-i18n'

import { Puzzle, Plus, Trash2, Power, PowerOff, RefreshCw } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import { listPlugins, createPlugin, updatePlugin, deletePlugin, enablePlugin, disablePlugin, reloadPlugin } from '@/api/modules/plugins'

const { t } = useI18n()
const { toast } = useToast()

interface Plugin {
  id: string
  tenant_id: string
  name: string
  display_name: string
  version: string
  description: string
  author: string
  provider: string
  type: string
  config: string
  enabled: boolean
  priority: number
  created_at: string
  updated_at: string
}

const plugins = ref<Plugin[]>([])
const loading = ref(false)
const dialogOpen = ref(false)
const isEdit = ref(false)
const selectedId = ref('')
const providerFilter = ref('')

const form = ref({
  name: '', display_name: '', version: '1.0.0', description: '', author: '',
  provider: 'dns', type: 'builtin', config: '', priority: 0,
})

const filteredPlugins = computed(() => {
  if (!providerFilter.value) return plugins.value
  return plugins.value.filter(p => p.provider === providerFilter.value)
})

const providers = computed(() => [...new Set(plugins.value.map(p => p.provider))].sort())

const providerColor = (p: string) => {
  const map: Record<string, string> = {
    dns: 'bg-blue-500/10 text-blue-400', notify: 'bg-green-500/10 text-green-400',
    registry: 'bg-purple-500/10 text-purple-400', cicd: 'bg-orange-500/10 text-orange-400',
    server: 'bg-cyan-500/10 text-cyan-400', ssl: 'bg-amber-500/10 text-amber-400',
  }
  return map[p] || 'bg-gray-500/10 text-gray-400'
}

async function fetchPlugins() {
  loading.value = true
  try {
    const res = await listPlugins(undefined, providerFilter.value || undefined)
    if (res.data.status === 'success') plugins.value = res.data.data || []
  } catch { toast(t('plugins.fetchFailed'), 'destructive') }
  finally { loading.value = false }
}

function openCreate() {
  isEdit.value = false
  form.value = { name: '', display_name: '', version: '1.0.0', description: '', author: '', provider: 'dns', type: 'builtin', config: '', priority: 0 }
  dialogOpen.value = true
}

function openEdit(p: Plugin) {
  isEdit.value = true
  selectedId.value = p.id
  form.value = { name: p.name, display_name: p.display_name || '', version: p.version || '1.0.0', description: p.description || '', author: p.author || '', provider: p.provider, type: p.type, config: p.config || '', priority: p.priority || 0 }
  dialogOpen.value = true
}

async function save() {
  try {
    if (isEdit.value) { await updatePlugin(selectedId.value, form.value); toast(t('plugins.updated'), 'success') }
    else { await createPlugin(form.value); toast(t('plugins.created'), 'success') }
    dialogOpen.value = false; await fetchPlugins()
  } catch { toast(isEdit.value ? t('plugins.updateFailed') : t('plugins.createFailed'), 'destructive') }
}

async function remove(id: string) {
  if (!confirm(t('plugins.confirmDelete'))) return
  try { await deletePlugin(id); toast(t('plugins.deleted'), 'success'); await fetchPlugins() }
  catch { toast(t('plugins.deleteFailed'), 'destructive') }
}

async function toggleEnable(p: Plugin) {
  try {
    if (p.enabled) { await disablePlugin(p.id); toast(t('plugins.disabled'), 'success') }
    else { await enablePlugin(p.id); toast(t('plugins.enabled'), 'success') }
    await fetchPlugins()
  } catch { toast(t('plugins.toggleFailed'), 'destructive') }
}

async function reload(p: Plugin) {
  try { await reloadPlugin(p.id); toast(t('plugins.reloaded'), 'success'); await fetchPlugins() }
  catch { toast(t('plugins.reloadFailed'), 'destructive') }
}

onMounted(fetchPlugins)
</script>

<template>
  <div>
    <PageHeader :title="t('plugins.title')" :description="t('plugins.description')">
      <Button @click="openCreate"><Plus class="w-4 h-4 mr-2" />{{ t('plugins.create') }}</Button>
    </PageHeader>

    <!-- Provider Filter -->
    <div v-if="providers.length > 0" class="flex gap-2 flex-wrap mb-4">
      <button class="px-3 py-1.5 text-xs rounded-lg border transition-colors" :class="!providerFilter ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'" @click="providerFilter = ''; fetchPlugins()">
        {{ t('plugins.allProviders') }}
      </button>
      <button v-for="p in providers" :key="p" class="px-3 py-1.5 text-xs rounded-lg border transition-colors capitalize" :class="providerFilter === p ? 'bg-white/10 border-white/20 text-white' : 'bg-transparent border-gray-700 text-gray-400 hover:text-white'" @click="providerFilter = p; fetchPlugins()">
        {{ p }}
      </button>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin h-8 w-8 border-2 border-white/20 border-t-white rounded-full" />
    </div>
    <div v-else-if="plugins.length === 0">
      <EmptyState :icon="Puzzle" :title="t('plugins.emptyTitle')" :description="t('plugins.emptyDescription')" />
    </div>
    <div v-else class="bg-gray-900/50 border border-gray-800 rounded-xl overflow-hidden">
      <table class="w-full text-sm">
        <thead>
          <tr class="border-b border-gray-800">
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('plugins.name') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('plugins.provider') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('plugins.version') }}</th>
            <th class="text-left px-4 py-3 text-gray-400 font-medium">{{ t('plugins.status') }}</th>
            <th class="text-right px-4 py-3 text-gray-400 font-medium">{{ t('plugins.actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="p in filteredPlugins" :key="p.id" class="border-b border-gray-800/50 hover:bg-gray-800/30 transition-colors">
            <td class="px-4 py-3">
              <div class="text-gray-300">{{ p.display_name || p.name }}</div>
              <div class="text-xs text-gray-500 mt-0.5">{{ p.description || p.type }}</div>
            </td>
            <td class="px-4 py-3">
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs" :class="providerColor(p.provider)">{{ p.provider }}</span>
            </td>
            <td class="px-4 py-3 text-gray-400 text-xs">v{{ p.version }}</td>
            <td class="px-4 py-3">
              <button class="inline-flex items-center gap-1 text-xs" :class="p.enabled ? 'text-green-400' : 'text-gray-500'" @click="toggleEnable(p)">
                <component :is="p.enabled ? Power : PowerOff" class="w-3 h-3" />
                {{ p.enabled ? t('plugins.on') : t('plugins.off') }}
              </button>
            </td>
            <td class="px-4 py-3 text-right">
              <div class="flex items-center justify-end gap-2">
                <button class="text-xs text-blue-400 hover:text-blue-300" @click="reload(p)"><RefreshCw class="w-3 h-3" /></button>
                <button class="text-xs text-gray-400 hover:text-white" @click="openEdit(p)">{{ t('common.edit') }}</button>
                <button class="text-xs text-red-400 hover:text-red-300" @click="remove(p.id)">{{ t('common.delete') }}</button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <Teleport to="body">
      <div v-if="dialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="dialogOpen = false">
        <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
          <h2 class="text-lg font-semibold text-white mb-4">{{ isEdit ? t('plugins.edit') : t('plugins.create') }}</h2>
          <div class="space-y-3">
            <div class="grid grid-cols-2 gap-3">
              <div><label class="block text-sm text-gray-400 mb-1">{{ t('plugins.name') }}</label><input v-model="form.name" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" /></div>
              <div><label class="block text-sm text-gray-400 mb-1">{{ t('plugins.version') }}</label><input v-model="form.version" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" /></div>
            </div>
            <div><label class="block text-sm text-gray-400 mb-1">{{ t('plugins.descText') }}</label><textarea v-model="form.description" rows="2" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white resize-none" /></div>
            <div class="grid grid-cols-2 gap-3">
              <div><label class="block text-sm text-gray-400 mb-1">{{ t('plugins.provider') }}</label>
                <select v-model="form.provider" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white">
                  <option value="dns">DNS</option><option value="notify">Notify</option><option value="registry">Registry</option>
                  <option value="cicd">CICD</option><option value="server">Server</option><option value="ssl">SSL</option>
                </select>
              </div>
              <div><label class="block text-sm text-gray-400 mb-1">{{ t('plugins.author') }}</label><input v-model="form.author" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" /></div>
            </div>
            <div><label class="block text-sm text-gray-400 mb-1">Config (JSON)</label><textarea v-model="form.config" rows="3" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white font-mono resize-none" /></div>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <Button variant="ghost" @click="dialogOpen = false">{{ t('common.cancel') }}</Button>
            <Button @click="save">{{ t('common.save') }}</Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
