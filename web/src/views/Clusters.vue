<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { inject } from 'vue'
import { Server, Plus, Globe, Trash2, RefreshCw, Settings } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import { listClusters, createCluster, updateCluster, deleteCluster, testClusterConnection } from '@/api/modules/clusters'

const { t } = useI18n()
const toast: (msg: string, type?: string) => void = inject<any>('toast')

interface Cluster {
  id: string
  tenant_id: string
  name: string
  description: string
  api_server: string
  context: string
  namespace: string
  status: string
  version: string
  node_count: number
  tags: string
  created_at: string
  updated_at: string
}

const clusters = ref<Cluster[]>([])
const loading = ref(false)
const createDialogOpen = ref(false)
const editDialogOpen = ref(false)
const selectedCluster = ref<Cluster | null>(null)
const testingId = ref<string | null>(null)

const form = ref({
  name: '',
  description: '',
  api_server: '',
  kube_config: '',
  context: '',
  namespace: 'default',
  token: '',
  ca_data: '',
  tags: '',
})

const statusColor = (status: string) => {
  const map: Record<string, string> = {
    active: 'bg-green-500/10 border-green-500/20 text-green-400',
    connecting: 'bg-amber-500/10 border-amber-500/20 text-amber-400',
    error: 'bg-red-500/10 border-red-500/20 text-red-400',
    unknown: 'bg-gray-500/10 border-gray-500/20 text-gray-400',
  }
  return map[status] || 'bg-gray-500/10 border-gray-500/20 text-gray-400'
}

async function fetchClusters() {
  loading.value = true
  try {
    const res = await listClusters()
    if (res.data.status === 'success') {
      clusters.value = res.data.data || []
    }
  } catch {
    toast(t('clusters.fetchFailed'), 'error')
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.value = { name: '', description: '', api_server: '', kube_config: '', context: '', namespace: 'default', token: '', ca_data: '', tags: '' }
  createDialogOpen.value = true
}

function openEdit(cluster: Cluster) {
  selectedCluster.value = cluster
  form.value = {
    name: cluster.name,
    description: cluster.description || '',
    api_server: cluster.api_server || '',
    kube_config: '',
    context: cluster.context || '',
    namespace: cluster.namespace || 'default',
    token: '',
    ca_data: '',
    tags: cluster.tags || '',
  }
  editDialogOpen.value = true
}

async function saveCreate() {
  try {
    await createCluster(form.value)
    toast(t('clusters.created'), 'success')
    createDialogOpen.value = false
    await fetchClusters()
  } catch {
    toast(t('clusters.createFailed'), 'error')
  }
}

async function saveEdit() {
  if (!selectedCluster.value) return
  try {
    await updateCluster(selectedCluster.value.id, form.value)
    toast(t('clusters.updated'), 'success')
    editDialogOpen.value = false
    await fetchClusters()
  } catch {
    toast(t('clusters.updateFailed'), 'error')
  }
}

async function removeCluster(id: string) {
  if (!confirm(t('clusters.confirmDelete'))) return
  try {
    await deleteCluster(id)
    toast(t('clusters.deleted'), 'success')
    await fetchClusters()
  } catch {
    toast(t('clusters.deleteFailed'), 'error')
  }
}

async function testConnection(id: string) {
  testingId.value = id
  try {
    const res = await testClusterConnection(id)
    if (res.data.status === 'success') {
      toast(t('clusters.connectionOk'), 'success')
    } else {
      toast(t('clusters.connectionFailed'), 'error')
    }
  } catch {
    toast(t('clusters.connectionFailed'), 'error')
  } finally {
    testingId.value = null
  }
}

onMounted(fetchClusters)
</script>

<template>
  <div>
    <PageHeader :title="t('clusters.title')" :description="t('clusters.description')">
      <Button @click="openCreate">
        <Plus class="w-4 h-4 mr-2" />
        {{ t('clusters.create') }}
      </Button>
    </PageHeader>

    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin h-8 w-8 border-2 border-white/20 border-t-white rounded-full" />
    </div>

    <div v-else-if="clusters.length === 0">
      <EmptyState :icon="Server" :title="t('clusters.emptyTitle')" :description="t('clusters.emptyDescription')" />
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card v-for="cluster in clusters" :key="cluster.id" class="hover:border-gray-600 transition-colors">
        <div class="p-4">
          <div class="flex items-start justify-between mb-3">
            <div>
              <h3 class="text-sm font-medium text-white">{{ cluster.name }}</h3>
              <p class="text-xs text-gray-500 mt-1">{{ cluster.description || cluster.api_server }}</p>
            </div>
            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs border" :class="statusColor(cluster.status)">
              {{ cluster.status }}
            </span>
          </div>
          <div class="space-y-1.5 text-xs text-gray-400">
            <div class="flex items-center gap-2">
              <Globe class="w-3 h-3" />
              <span class="truncate">{{ cluster.api_server }}</span>
            </div>
            <div v-if="cluster.version" class="flex items-center gap-2">
              <Server class="w-3 h-3" />
              <span>v{{ cluster.version }} · {{ cluster.node_count || 0 }} nodes</span>
            </div>
            <div v-if="cluster.context" class="flex items-center gap-2">
              <Settings class="w-3 h-3" />
              <span>{{ cluster.context }} / {{ cluster.namespace || 'default' }}</span>
            </div>
          </div>
          <div class="flex items-center gap-2 mt-4 pt-3 border-t border-gray-800">
            <Button variant="ghost" size="sm" :disabled="testingId === cluster.id" @click="testConnection(cluster.id)">
              <RefreshCw class="w-3 h-3 mr-1" :class="{ 'animate-spin': testingId === cluster.id }" />
              {{ t('clusters.test') }}
            </Button>
            <Button variant="ghost" size="sm" @click="openEdit(cluster)">
              <Settings class="w-3 h-3 mr-1" />
              {{ t('common.edit') }}
            </Button>
            <Button variant="ghost" size="sm" class="text-red-400 hover:text-red-300" @click="removeCluster(cluster.id)">
              <Trash2 class="w-3 h-3 mr-1" />
              {{ t('common.delete') }}
            </Button>
          </div>
        </div>
      </Card>
    </div>

    <!-- Create Dialog -->
    <Teleport to="body">
      <div v-if="createDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="createDialogOpen = false">
        <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
          <h2 class="text-lg font-semibold text-white mb-4">{{ t('clusters.create') }}</h2>
          <div class="space-y-3">
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('clusters.name') }}</label>
              <input v-model="form.name" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('clusters.description') }}</label>
              <input v-model="form.description" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">API Server</label>
              <input v-model="form.api_server" placeholder="https://k8s-api.example.com:6443" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">Kube Config</label>
              <textarea v-model="form.kube_config" rows="3" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white font-mono resize-none" />
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-sm text-gray-400 mb-1">Context</label>
                <input v-model="form.context" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
              </div>
              <div>
                <label class="block text-sm text-gray-400 mb-1">Namespace</label>
                <input v-model="form.namespace" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
              </div>
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">Token</label>
              <input v-model="form.token" type="password" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">CA Data</label>
              <textarea v-model="form.ca_data" rows="2" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white font-mono resize-none" />
            </div>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <Button variant="ghost" @click="createDialogOpen = false">{{ t('common.cancel') }}</Button>
            <Button @click="saveCreate">{{ t('common.save') }}</Button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Edit Dialog -->
    <Teleport to="body">
      <div v-if="editDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="editDialogOpen = false">
        <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
          <h2 class="text-lg font-semibold text-white mb-4">{{ t('clusters.edit') }}</h2>
          <div class="space-y-3">
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('clusters.name') }}</label>
              <input v-model="form.name" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('clusters.description') }}</label>
              <input v-model="form.description" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">API Server</label>
              <input v-model="form.api_server" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-sm text-gray-400 mb-1">Context</label>
                <input v-model="form.context" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
              </div>
              <div>
                <label class="block text-sm text-gray-400 mb-1">Namespace</label>
                <input v-model="form.namespace" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
              </div>
            </div>
          </div>
          <div class="flex justify-end gap-3 mt-6">
            <Button variant="ghost" @click="editDialogOpen = false">{{ t('common.cancel') }}</Button>
            <Button @click="saveEdit">{{ t('common.save') }}</Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
