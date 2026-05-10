<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import {
  LayoutDashboard,
  Plus,
  Pencil,
  Trash2,
  RefreshCw,
  Tag,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import {
  listGrafanaDashboards,
  createGrafanaDashboard,
  updateGrafanaDashboard,
  deleteGrafanaDashboard,
  syncGrafana,
} from '@/api/modules/grafana'
import type { GrafanaDashboard } from '@/api/modules/grafana'

const { toast } = useToast()

const dashboards = ref<GrafanaDashboard[]>([])
const loading = ref(false)
const error = ref('')
const syncing = ref(false)

// Upload dialog state
const uploadDialogOpen = ref(false)
const uploading = ref(false)
const uploadForm = ref({
  name: '',
  tags: '',
  json: '',
})

// Edit dialog state
const editDialogOpen = ref(false)
const editing = ref(false)
const editForm = ref({
  id: '',
  name: '',
  tags: '',
  json: '',
  enabled: true,
})

// Delete dialog state
const deleteDialogOpen = ref(false)
const deletingDashboard = ref<GrafanaDashboard | null>(null)
const deleting = ref(false)

async function fetchDashboards() {
  loading.value = true
  error.value = ''
  try {
    const res = await listGrafanaDashboards()
    if (res.data.status === 'success') {
      dashboards.value = res.data.data
    }
  } catch (e: any) {
    error.value = e?.message || 'Failed to load dashboards'
  } finally {
    loading.value = false
  }
}

function openUploadDialog() {
  uploadForm.value = { name: '', tags: '', json: '' }
  uploadDialogOpen.value = true
}

async function handleUpload() {
  if (!uploadForm.value.name.trim() || !uploadForm.value.json.trim()) {
    toast('Name and JSON are required', 'destructive')
    return
  }
  // Validate JSON
  try {
    JSON.parse(uploadForm.value.json)
  } catch {
    toast('Invalid JSON format', 'destructive')
    return
  }
  uploading.value = true
  try {
    await createGrafanaDashboard({
      name: uploadForm.value.name.trim(),
      json: uploadForm.value.json.trim(),
      tags: uploadForm.value.tags.trim() || undefined,
    })
    toast('Dashboard created successfully', 'success')
    uploadDialogOpen.value = false
    fetchDashboards()
  } catch (e: any) {
    toast(e?.message || 'Failed to create dashboard', 'destructive')
  } finally {
    uploading.value = false
  }
}

function openEditDialog(dashboard: GrafanaDashboard) {
  editForm.value = {
    id: dashboard.id,
    name: dashboard.name,
    tags: dashboard.tags,
    json: '',
    enabled: dashboard.enabled,
  }
  editDialogOpen.value = true
}

async function handleEdit() {
  if (!editForm.value.name.trim()) {
    toast('Name is required', 'destructive')
    return
  }
  const data: { name?: string; tags?: string; json?: string; enabled?: boolean } = {
    name: editForm.value.name.trim(),
    tags: editForm.value.tags.trim() || undefined,
    enabled: editForm.value.enabled,
  }
  if (editForm.value.json.trim()) {
    try {
      JSON.parse(editForm.value.json)
      data.json = editForm.value.json.trim()
    } catch {
      toast('Invalid JSON format', 'destructive')
      return
    }
  }
  editing.value = true
  try {
    await updateGrafanaDashboard(editForm.value.id, data)
    toast('Dashboard updated successfully', 'success')
    editDialogOpen.value = false
    fetchDashboards()
  } catch (e: any) {
    toast(e?.message || 'Failed to update dashboard', 'destructive')
  } finally {
    editing.value = false
  }
}

function openDeleteDialog(dashboard: GrafanaDashboard) {
  deletingDashboard.value = dashboard
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingDashboard.value) return
  deleting.value = true
  try {
    await deleteGrafanaDashboard(deletingDashboard.value.id)
    toast('Dashboard deleted', 'success')
    deleteDialogOpen.value = false
    deletingDashboard.value = null
    fetchDashboards()
  } catch (e: any) {
    toast(e?.message || 'Failed to delete dashboard', 'destructive')
  } finally {
    deleting.value = false
  }
}

async function handleSyncAll() {
  syncing.value = true
  try {
    const res = await syncGrafana()
    if (res.data.status === 'success') {
      toast(`Synced ${res.data.data.synced} dashboards`, 'success')
      fetchDashboards()
    }
  } catch (e: any) {
    toast(e?.message || 'Sync failed', 'destructive')
  } finally {
    syncing.value = false
  }
}

function parseTags(tags: string): string[] {
  if (!tags) return []
  return tags.split(',').map(t => t.trim()).filter(Boolean)
}

onMounted(fetchDashboards)
</script>

<template>
  <div class="space-y-4">
    <PageHeader title="Grafana Dashboards" description="Manage built-in and custom Grafana dashboards">
      <template #actions>
        <Button variant="outline" :loading="syncing" @click="handleSyncAll">
          <template #icon>
            <RefreshCw class="w-4 h-4" />
          </template>
          Sync All
        </Button>
        <Button @click="openUploadDialog">
          <template #icon>
            <Plus class="w-4 h-4" />
          </template>
          Upload Dashboard
        </Button>
      </template>
    </PageHeader>

    <!-- Loading State -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="i in 6" :key="i" class="rounded-lg border border-border bg-card p-6 space-y-3">
        <Skeleton class="h-5 w-3/4" />
        <Skeleton class="h-4 w-full" />
        <div class="flex gap-2">
          <Skeleton variant="circular" class="h-6 w-16 !rounded-full" />
          <Skeleton variant="circular" class="h-6 w-16 !rounded-full" />
        </div>
        <Skeleton class="h-4 w-1/2" />
      </div>
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
      {{ error }}
    </div>

    <!-- Empty State -->
    <EmptyState
      v-else-if="dashboards.length === 0"
      :icon="LayoutDashboard"
      title="No dashboards found"
      description="Sync dashboards from Grafana or upload a custom dashboard JSON."
      action-text="Sync All"
      @action="handleSyncAll"
    />

    <!-- Dashboard Cards -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card v-for="dashboard in dashboards" :key="dashboard.id" class="p-0">
        <div class="p-6 space-y-3">
          <!-- Header: Name + Type badge -->
          <div class="flex items-start justify-between">
            <h3 class="text-sm font-semibold text-foreground truncate">{{ dashboard.name }}</h3>
            <span
              class="inline-flex items-center px-2 py-0.5 text-xs rounded-full font-medium shrink-0 ml-2"
              :class="dashboard.is_built_in ? 'bg-blue-100 text-blue-700' : 'bg-purple-100 text-purple-700'"
            >
              {{ dashboard.is_built_in ? 'Built-in' : 'Custom' }}
            </span>
          </div>

          <!-- Description -->
          <p v-if="dashboard.description" class="text-xs text-muted-foreground line-clamp-2">
            {{ dashboard.description }}
          </p>

          <!-- Tags -->
          <div v-if="dashboard.tags" class="flex items-center gap-1.5 flex-wrap">
            <Tag class="w-3 h-3 text-muted-foreground shrink-0" />
            <span
              v-for="tag in parseTags(dashboard.tags)"
              :key="tag"
              class="inline-flex items-center px-1.5 py-0.5 text-xs rounded bg-gray-100 text-gray-600"
            >
              {{ tag }}
            </span>
          </div>

          <!-- Enabled status -->
          <div class="flex items-center gap-2">
            <span
              class="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs rounded-full"
              :class="dashboard.enabled ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'"
            >
              <span class="w-1.5 h-1.5 rounded-full" :class="dashboard.enabled ? 'bg-green-500' : 'bg-gray-400'" />
              {{ dashboard.enabled ? 'Enabled' : 'Disabled' }}
            </span>
          </div>

          <!-- Updated time -->
          <div class="text-xs text-muted-foreground">
            Updated: <RelativeTime :date="dashboard.updated_at" />
          </div>
        </div>

        <!-- Card Actions -->
        <div class="flex items-center border-t border-border px-4 py-2 gap-1">
          <Button variant="ghost" size="sm" class="h-7 text-xs" @click="openEditDialog(dashboard)">
            <template #icon>
              <Pencil class="w-3.5 h-3.5" />
            </template>
            {{ dashboard.is_built_in ? 'View' : 'Edit' }}
          </Button>
          <template v-if="!dashboard.is_built_in">
            <div class="flex-1" />
            <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive hover:text-destructive hover:bg-destructive/10" @click="openDeleteDialog(dashboard)">
              <Trash2 class="w-3.5 h-3.5" />
            </Button>
          </template>
        </div>
      </Card>
    </div>

    <!-- Upload Dashboard Dialog -->
    <Teleport to="body">
      <div v-if="uploadDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-lg p-6 border border-border max-h-[90vh] overflow-auto">
          <h2 class="text-lg font-semibold text-foreground mb-4">Upload Dashboard</h2>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-1">Name</label>
              <input
                v-model="uploadForm.name"
                type="text"
                placeholder="My Dashboard"
                class="w-full px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">Tags</label>
              <input
                v-model="uploadForm.tags"
                type="text"
                placeholder="monitoring, app (comma-separated)"
                class="w-full px-3 py-2 border rounded-md text-sm"
              />
              <p class="text-xs text-gray-500 mt-1">Comma-separated tags for categorization</p>
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">Dashboard JSON</label>
              <textarea
                v-model="uploadForm.json"
                rows="10"
                placeholder='Paste Grafana dashboard JSON here...'
                class="w-full px-3 py-2 border rounded-md text-sm font-mono resize-y"
              />
              <p class="text-xs text-gray-500 mt-1">Paste the full Grafana dashboard JSON exported from Grafana</p>
            </div>
          </div>
          <div class="flex justify-end gap-2 mt-6">
            <Button variant="outline" size="sm" @click="uploadDialogOpen = false">Cancel</Button>
            <Button size="sm" :loading="uploading" @click="handleUpload">
              <template #icon>
                <Plus class="w-3.5 h-3.5" />
              </template>
              Create
            </Button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Edit Dashboard Dialog -->
    <Teleport to="body">
      <div v-if="editDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-lg p-6 border border-border max-h-[90vh] overflow-auto">
          <h2 class="text-lg font-semibold text-foreground mb-4">{{ editForm.enabled ? 'Edit' : 'View' }} Dashboard</h2>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-1">Name</label>
              <input
                v-model="editForm.name"
                type="text"
                :disabled="editForm.enabled === false"
                class="w-full px-3 py-2 border rounded-md text-sm"
                :class="{ 'bg-gray-50 cursor-not-allowed': !editForm.enabled }"
              />
            </div>
            <div>
              <label class="block text-sm font-medium mb-1">Tags</label>
              <input
                v-model="editForm.tags"
                type="text"
                placeholder="monitoring, app (comma-separated)"
                :disabled="editForm.enabled === false"
                class="w-full px-3 py-2 border rounded-md text-sm"
                :class="{ 'bg-gray-50 cursor-not-allowed': !editForm.enabled }"
              />
            </div>
            <div v-if="!editForm.enabled">
              <label class="block text-sm font-medium mb-1">Dashboard JSON</label>
              <textarea
                v-model="editForm.json"
                rows="6"
                placeholder="Leave empty to keep existing JSON, or paste new JSON to replace"
                class="w-full px-3 py-2 border rounded-md text-sm font-mono resize-y bg-gray-50 cursor-not-allowed"
                disabled
              />
              <p class="text-xs text-gray-500 mt-1">Built-in dashboards cannot be edited</p>
            </div>
            <div v-else>
              <label class="block text-sm font-medium mb-1">Dashboard JSON</label>
              <textarea
                v-model="editForm.json"
                rows="6"
                placeholder="Leave empty to keep existing JSON, or paste new JSON to replace"
                class="w-full px-3 py-2 border rounded-md text-sm font-mono resize-y"
              />
              <p class="text-xs text-gray-500 mt-1">Leave empty to keep existing JSON unchanged</p>
            </div>
          </div>
          <div class="flex justify-end gap-2 mt-6">
            <Button variant="outline" size="sm" @click="editDialogOpen = false">Cancel</Button>
            <Button v-if="editForm.enabled" size="sm" :loading="editing" @click="handleEdit">
              <template #icon>
                <Pencil class="w-3.5 h-3.5" />
              </template>
              Save
            </Button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Delete Confirmation Dialog -->
    <Teleport to="body">
      <div v-if="deleteDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-sm p-6 border border-border">
          <h2 class="text-lg font-semibold text-foreground mb-2">Delete Dashboard</h2>
          <p class="text-sm text-muted-foreground mb-4">
            Are you sure you want to delete "{{ deletingDashboard?.name }}"? This action cannot be undone.
          </p>
          <div class="flex justify-end gap-2">
            <Button variant="outline" size="sm" @click="deleteDialogOpen = false">Cancel</Button>
            <Button variant="destructive" size="sm" :loading="deleting" @click="confirmDelete">
              <template v-if="!deleting" #icon>
                <Trash2 class="w-3.5 h-3.5" />
              </template>
              Delete
            </Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
