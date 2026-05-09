<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { inject } from 'vue'
import { Database, Plus, Trash2, ExternalLink } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import { listRegistries, createRegistry, updateRegistry, deleteRegistry } from '@/api/modules/registries'

const { t } = useI18n()
const toast: (msg: string, type?: string) => void = inject<any>('toast')

interface Registry {
  id: string
  tenant_id: string
  name: string
  provider: string
  url: string
  username: string
  created_at: string
  updated_at: string
}

const registries = ref<Registry[]>([])
const loading = ref(false)
const dialogOpen = ref(false)
const isEdit = ref(false)
const selectedId = ref('')

const form = ref({ name: '', provider: 'docker_hub', url: '', username: '', password: '' })

const providerBadge = (provider: string) => {
  const map: Record<string, string> = {
    docker_hub: 'bg-blue-500/10 text-blue-400',
    harbor: 'bg-cyan-500/10 text-cyan-400',
    gitlab: 'bg-orange-500/10 text-orange-400',
    ghcr: 'bg-gray-500/10 text-gray-400',
    ecr: 'bg-amber-500/10 text-amber-400',
    acr: 'bg-green-500/10 text-green-400',
  }
  return map[provider] || 'bg-gray-500/10 text-gray-400'
}

async function fetchRegistries() {
  loading.value = true
  try {
    const res = await listRegistries()
    if (res.data.status === 'success') registries.value = res.data.data || []
  } catch { toast(t('registries.fetchFailed'), 'error') }
  finally { loading.value = false }
}

function openCreate() {
  isEdit.value = false
  form.value = { name: '', provider: 'docker_hub', url: '', username: '', password: '' }
  dialogOpen.value = true
}

function openEdit(r: Registry) {
  isEdit.value = true
  selectedId.value = r.id
  form.value = { name: r.name, provider: r.provider, url: r.url, username: r.username || '', password: '' }
  dialogOpen.value = true
}

async function save() {
  try {
    if (isEdit.value) {
      await updateRegistry(selectedId.value, form.value)
      toast(t('registries.updated'), 'success')
    } else {
      await createRegistry(form.value)
      toast(t('registries.created'), 'success')
    }
    dialogOpen.value = false
    await fetchRegistries()
  } catch { toast(isEdit.value ? t('registries.updateFailed') : t('registries.createFailed'), 'error') }
}

async function remove(id: string) {
  if (!confirm(t('registries.confirmDelete'))) return
  try { await deleteRegistry(id); toast(t('registries.deleted'), 'success'); await fetchRegistries() }
  catch { toast(t('registries.deleteFailed'), 'error') }
}

onMounted(fetchRegistries)
</script>

<template>
  <div>
    <PageHeader :title="t('registries.title')" :description="t('registries.description')">
      <Button @click="openCreate"><Plus class="w-4 h-4 mr-2" />{{ t('registries.create') }}</Button>
    </PageHeader>

    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="animate-spin h-8 w-8 border-2 border-white/20 border-t-white rounded-full" />
    </div>
    <div v-else-if="registries.length === 0">
      <EmptyState :icon="Database" :title="t('registries.emptyTitle')" :description="t('registries.emptyDescription')" />
    </div>
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card v-for="r in registries" :key="r.id" class="hover:border-gray-600 transition-colors">
        <div class="p-4">
          <div class="flex items-start justify-between mb-3">
            <div>
              <h3 class="text-sm font-medium text-white">{{ r.name }}</h3>
              <p class="text-xs text-gray-500 mt-1 flex items-center gap-1"><ExternalLink class="w-3 h-3" />{{ r.url }}</p>
            </div>
            <span class="inline-flex items-center px-2 py-0.5 rounded text-xs" :class="providerBadge(r.provider)">{{ r.provider }}</span>
          </div>
          <div class="text-xs text-gray-400">{{ r.username ? `${r.username}@${r.provider}` : r.provider }}</div>
          <div class="flex items-center gap-2 mt-4 pt-3 border-t border-gray-800">
            <Button variant="ghost" size="sm" @click="openEdit(r)">{{ t('common.edit') }}</Button>
            <Button variant="ghost" size="sm" class="text-red-400 hover:text-red-300" @click="remove(r.id)">{{ t('common.delete') }}</Button>
          </div>
        </div>
      </Card>
    </div>

    <Teleport to="body">
      <div v-if="dialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/60" @click.self="dialogOpen = false">
        <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md mx-4">
          <h2 class="text-lg font-semibold text-white mb-4">{{ isEdit ? t('registries.edit') : t('registries.create') }}</h2>
          <div class="space-y-3">
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('registries.name') }}</label>
              <input v-model="form.name" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('registries.provider') }}</label>
              <select v-model="form.provider" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white">
                <option value="docker_hub">Docker Hub</option>
                <option value="harbor">Harbor</option>
                <option value="gitlab">GitLab</option>
                <option value="ghcr">GHCR</option>
                <option value="ecr">ECR</option>
                <option value="acr">ACR</option>
              </select>
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">URL</label>
              <input v-model="form.url" placeholder="https://registry.example.com" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('registries.username') }}</label>
              <input v-model="form.username" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('registries.password') }}</label>
              <input v-model="form.password" type="password" class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-sm text-white" />
            </div>
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
