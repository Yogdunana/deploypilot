<script setup lang="ts">
import { ref, onMounted, inject } from 'vue'
import {
  KeyRound,
  Plus,
  Pencil,
  Trash2,
  Copy,
  RefreshCw,
  Check,
  Loader2,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as oauth2Api from '@/api/modules/oauth2'
import type { OAuth2Client } from '@/api/modules/oauth2'

const { toast } = inject<any>('toast')!

const clients = ref<OAuth2Client[]>([])
const loading = ref(false)
const error = ref('')

// Create dialog state
const createDialogOpen = ref(false)
const creating = ref(false)
const createForm = ref({
  name: '',
  redirect_uris: [''],
  scopes: [] as string[],
  grant_types: [] as string[],
})

// Edit dialog state
const editDialogOpen = ref(false)
const editing = ref(false)
const editClient = ref<OAuth2Client | null>(null)
const editForm = ref({
  name: '',
  redirect_uris: [''],
  scopes: [] as string[],
  grant_types: [] as string[],
  enabled: true,
})

// Delete dialog state
const deleteDialogOpen = ref(false)
const deletingClient = ref<OAuth2Client | null>(null)
const deleting = ref(false)

// Regenerate secret dialog state
const regenerateDialogOpen = ref(false)
const regeneratingClient = ref<OAuth2Client | null>(null)
const regenerating = ref(false)

// Secret reveal dialog state
const secretDialogOpen = ref(false)
const revealedSecret = ref('')
const clientSecretCopied = ref(false)

const availableScopes = ['read', 'write', 'admin', 'apps', 'servers', 'deployments', 'monitor']
const availableGrantTypes = ['client_credentials', 'authorization_code']

function truncateClientId(id: string, maxLen = 24): string {
  if (id.length <= maxLen) return id
  return id.substring(0, maxLen) + '...'
}

async function fetchClients() {
  loading.value = true
  error.value = ''
  try {
    const res = await oauth2Api.listOAuth2Clients()
    if (res.data.status === 'success') {
      clients.value = res.data.data
    }
  } catch (e: any) {
    error.value = e?.message || 'Failed to load OAuth2 clients'
  } finally {
    loading.value = false
  }
}

function openCreateDialog() {
  createForm.value = {
    name: '',
    redirect_uris: [''],
    scopes: [],
    grant_types: [],
  }
  createDialogOpen.value = true
}

async function handleCreate() {
  if (!createForm.value.name.trim()) return
  creating.value = true
  try {
    const data: any = { name: createForm.value.name }
    const uris = createForm.value.redirect_uris.filter(u => u.trim())
    if (uris.length > 0) data.redirect_uris = uris
    if (createForm.value.scopes.length > 0) data.scopes = createForm.value.scopes
    if (createForm.value.grant_types.length > 0) data.grant_types = createForm.value.grant_types

    const res = await oauth2Api.createOAuth2Client(data)
    if (res.data.status === 'success') {
      createDialogOpen.value = false
      revealedSecret.value = res.data.data.client_secret
      secretDialogOpen.value = true
      fetchClients()
    }
  } catch (e: any) {
    toast(e?.message || 'Failed to create OAuth2 client', 'destructive')
  } finally {
    creating.value = false
  }
}

function addRedirectUri() {
  createForm.value.redirect_uris.push('')
}

function removeRedirectUri(index: number) {
  createForm.value.redirect_uris.splice(index, 1)
}

function addEditRedirectUri() {
  editForm.value.redirect_uris.push('')
}

function removeEditRedirectUri(index: number) {
  editForm.value.redirect_uris.splice(index, 1)
}

function openEditDialog(client: OAuth2Client) {
  editClient.value = client
  editForm.value = {
    name: client.name,
    redirect_uris: client.redirect_uris.length > 0 ? [...client.redirect_uris] : [''],
    scopes: [...client.scopes],
    grant_types: [...client.grant_types],
    enabled: client.enabled,
  }
  editDialogOpen.value = true
}

async function handleEdit() {
  if (!editClient.value || !editForm.value.name.trim()) return
  editing.value = true
  try {
    const data: any = { name: editForm.value.name }
    const uris = editForm.value.redirect_uris.filter(u => u.trim())
    data.redirect_uris = uris
    data.scopes = editForm.value.scopes
    data.grant_types = editForm.value.grant_types
    data.enabled = editForm.value.enabled

    await oauth2Api.updateOAuth2Client(editClient.value.id, data)
    toast('OAuth2 client updated', 'success')
    editDialogOpen.value = false
    editClient.value = null
    fetchClients()
  } catch (e: any) {
    toast(e?.message || 'Failed to update OAuth2 client', 'destructive')
  } finally {
    editing.value = false
  }
}

function openDeleteDialog(client: OAuth2Client) {
  deletingClient.value = client
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingClient.value) return
  deleting.value = true
  try {
    await oauth2Api.deleteOAuth2Client(deletingClient.value.id)
    toast('OAuth2 client deleted', 'success')
    deleteDialogOpen.value = false
    deletingClient.value = null
    fetchClients()
  } catch (e: any) {
    toast(e?.message || 'Failed to delete OAuth2 client', 'destructive')
  } finally {
    deleting.value = false
  }
}

function openRegenerateDialog(client: OAuth2Client) {
  regeneratingClient.value = client
  regenerateDialogOpen.value = true
}

async function confirmRegenerate() {
  if (!regeneratingClient.value) return
  regenerating.value = true
  try {
    const res = await oauth2Api.regenerateOAuth2Secret(regeneratingClient.value.id)
    if (res.data.status === 'success') {
      regenerateDialogOpen.value = false
      revealedSecret.value = res.data.data.client_secret
      secretDialogOpen.value = true
      regeneratingClient.value = null
    }
  } catch (e: any) {
    toast(e?.message || 'Failed to regenerate secret', 'destructive')
  } finally {
    regenerating.value = false
  }
}

async function copySecret() {
  try {
    await navigator.clipboard.writeText(revealedSecret.value)
    clientSecretCopied.value = true
    toast('Client secret copied to clipboard', 'success')
    setTimeout(() => {
      clientSecretCopied.value = false
    }, 2000)
  } catch {
    toast('Failed to copy to clipboard', 'destructive')
  }
}

async function copyClientId(clientId: string) {
  try {
    await navigator.clipboard.writeText(clientId)
    toast('Client ID copied to clipboard', 'success')
  } catch {
    toast('Failed to copy to clipboard', 'destructive')
  }
}

function closeSecretDialog() {
  secretDialogOpen.value = false
  revealedSecret.value = ''
  clientSecretCopied.value = false
}

onMounted(fetchClients)
</script>

<template>
  <div class="space-y-4">
    <PageHeader title="OAuth2 Applications" description="Manage OAuth2 applications for API access">
      <template #actions>
        <Button @click="openCreateDialog">
          <template #icon>
            <Plus class="w-4 h-4" />
          </template>
          Create App
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
      v-else-if="clients.length === 0"
      :icon="KeyRound"
      title="No OAuth2 applications"
      description="Create your first OAuth2 application to enable third-party API access."
      action-text="Create App"
      @action="openCreateDialog"
    />

    <!-- Client Cards -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card v-for="client in clients" :key="client.id" class="p-0">
        <div class="p-6 space-y-3">
          <!-- Header: Name + Enabled indicator -->
          <div class="flex items-start justify-between">
            <h3 class="text-sm font-semibold text-foreground truncate">{{ client.name }}</h3>
            <span
              class="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs rounded-full shrink-0 ml-2"
              :class="client.enabled ? 'bg-success/15 text-success' : 'bg-accent text-muted-foreground'"
            >
              <span class="w-1.5 h-1.5 rounded-full" :class="client.enabled ? 'bg-success' : 'bg-muted-foreground'" />
              {{ client.enabled ? 'Enabled' : 'Disabled' }}
            </span>
          </div>

          <!-- Client ID -->
          <div class="flex items-center gap-1.5">
            <p class="text-xs font-mono text-muted-foreground truncate" :title="client.client_id">
              {{ truncateClientId(client.client_id) }}
            </p>
            <button
              class="inline-flex items-center justify-center w-5 h-5 rounded text-muted-foreground hover:text-foreground hover:bg-accent transition-colors cursor-pointer shrink-0"
              title="Copy Client ID"
              @click="copyClientId(client.client_id)"
            >
              <Copy class="w-3 h-3" />
            </button>
          </div>

          <!-- Grant Type Badges -->
          <div class="flex items-center gap-2 flex-wrap">
            <span
              v-for="gt in client.grant_types"
              :key="gt"
              class="inline-flex items-center px-2 py-0.5 text-xs rounded-full font-medium bg-blue-100 text-blue-700"
            >
              {{ gt === 'client_credentials' ? 'Client Credentials' : 'Authorization Code' }}
            </span>
          </div>

          <!-- Scope Tags -->
          <div v-if="client.scopes.length > 0" class="flex items-center gap-1.5 flex-wrap">
            <Badge v-for="scope in client.scopes" :key="scope" variant="secondary" class="text-[11px]">
              {{ scope }}
            </Badge>
          </div>

          <!-- Created date -->
          <div class="text-xs text-muted-foreground">
            Created: <RelativeTime :date="client.created_at" />
          </div>
        </div>

        <!-- Card Actions -->
        <div class="flex items-center border-t border-border px-4 py-2 gap-1">
          <Button variant="ghost" size="sm" class="h-7 text-xs" @click="openEditDialog(client)">
            <template #icon>
              <Pencil class="w-3.5 h-3.5" />
            </template>
            Edit
          </Button>
          <Button variant="ghost" size="sm" class="h-7 text-xs" @click="openRegenerateDialog(client)">
            <template #icon>
              <RefreshCw class="w-3.5 h-3.5" />
            </template>
            Regenerate
          </Button>
          <div class="flex-1" />
          <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive hover:text-destructive hover:bg-destructive/10" @click="openDeleteDialog(client)">
            <Trash2 class="w-3.5 h-3.5" />
          </Button>
        </div>
      </Card>
    </div>

    <!-- Create Dialog -->
    <Teleport to="body">
      <div v-if="createDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-md p-6 border border-border">
          <h2 class="text-lg font-semibold text-foreground mb-4">Create OAuth2 Application</h2>
          <div class="space-y-4">
            <!-- Name -->
            <div>
              <label class="block text-sm font-medium text-foreground mb-1">Name</label>
              <input
                v-model="createForm.name"
                type="text"
                placeholder="Enter application name"
                class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>

            <!-- Redirect URIs -->
            <div>
              <label class="block text-sm font-medium text-foreground mb-1">Redirect URIs</label>
              <div class="space-y-2">
                <div v-for="(uri, index) in createForm.redirect_uris" :key="index" class="flex items-center gap-2">
                  <input
                    v-model="createForm.redirect_uris[index]"
                    type="text"
                    placeholder="https://example.com/callback"
                    class="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  />
                  <button
                    v-if="createForm.redirect_uris.length > 1"
                    class="inline-flex items-center justify-center w-8 h-8 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                    @click="removeRedirectUri(index)"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
                <Button variant="outline" size="sm" @click="addRedirectUri">
                  <template #icon>
                    <Plus class="w-3.5 h-3.5" />
                  </template>
                  Add URI
                </Button>
              </div>
            </div>

            <!-- Scopes -->
            <div>
              <label class="block text-sm font-medium text-foreground mb-2">Scopes</label>
              <div class="flex flex-wrap gap-2">
                <label
                  v-for="scope in availableScopes"
                  :key="scope"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border text-xs cursor-pointer transition-colors"
                  :class="createForm.scopes.includes(scope) ? 'bg-primary/10 border-primary text-primary' : 'bg-background text-muted-foreground hover:bg-accent'"
                >
                  <input
                    type="checkbox"
                    :value="scope"
                    class="sr-only"
                    @change="(e: any) => {
                      if (e.target.checked) createForm.scopes.push(scope)
                      else createForm.scopes = createForm.scopes.filter(s => s !== scope)
                    }"
                  />
                  {{ scope }}
                </label>
              </div>
            </div>

            <!-- Grant Types -->
            <div>
              <label class="block text-sm font-medium text-foreground mb-2">Grant Types</label>
              <div class="space-y-2">
                <label
                  v-for="gt in availableGrantTypes"
                  :key="gt"
                  class="flex items-center gap-2 px-3 py-2 rounded-md border border-border text-sm cursor-pointer transition-colors"
                  :class="createForm.grant_types.includes(gt) ? 'bg-primary/10 border-primary text-primary' : 'bg-background text-muted-foreground hover:bg-accent'"
                >
                  <input
                    type="checkbox"
                    :value="gt"
                    class="sr-only"
                    @change="(e: any) => {
                      if (e.target.checked) createForm.grant_types.push(gt)
                      else createForm.grant_types = createForm.grant_types.filter(g => g !== gt)
                    }"
                  />
                  <span class="w-4 h-4 rounded border flex items-center justify-center shrink-0" :class="createForm.grant_types.includes(gt) ? 'bg-primary border-primary' : 'border-border'">
                    <Check v-if="createForm.grant_types.includes(gt)" class="w-3 h-3 text-primary-foreground" />
                  </span>
                  {{ gt === 'client_credentials' ? 'Client Credentials' : 'Authorization Code' }}
                </label>
              </div>
            </div>
          </div>

          <div class="flex justify-end gap-2 mt-6">
            <Button variant="outline" size="sm" @click="createDialogOpen = false">Cancel</Button>
            <Button size="sm" :loading="creating" :disabled="!createForm.name.trim()" @click="handleCreate">
              <template v-if="!creating" #icon>
                <Plus class="w-3.5 h-3.5" />
              </template>
              Create
            </Button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Edit Dialog -->
    <Teleport to="body">
      <div v-if="editDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-md p-6 border border-border">
          <h2 class="text-lg font-semibold text-foreground mb-4">Edit OAuth2 Application</h2>
          <div class="space-y-4">
            <!-- Name -->
            <div>
              <label class="block text-sm font-medium text-foreground mb-1">Name</label>
              <input
                v-model="editForm.name"
                type="text"
                placeholder="Enter application name"
                class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
              />
            </div>

            <!-- Redirect URIs -->
            <div>
              <label class="block text-sm font-medium text-foreground mb-1">Redirect URIs</label>
              <div class="space-y-2">
                <div v-for="(uri, index) in editForm.redirect_uris" :key="index" class="flex items-center gap-2">
                  <input
                    v-model="editForm.redirect_uris[index]"
                    type="text"
                    placeholder="https://example.com/callback"
                    class="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                  />
                  <button
                    v-if="editForm.redirect_uris.length > 1"
                    class="inline-flex items-center justify-center w-8 h-8 rounded-md text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors cursor-pointer"
                    @click="removeEditRedirectUri(index)"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
                <Button variant="outline" size="sm" @click="addEditRedirectUri">
                  <template #icon>
                    <Plus class="w-3.5 h-3.5" />
                  </template>
                  Add URI
                </Button>
              </div>
            </div>

            <!-- Scopes -->
            <div>
              <label class="block text-sm font-medium text-foreground mb-2">Scopes</label>
              <div class="flex flex-wrap gap-2">
                <label
                  v-for="scope in availableScopes"
                  :key="scope"
                  class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border text-xs cursor-pointer transition-colors"
                  :class="editForm.scopes.includes(scope) ? 'bg-primary/10 border-primary text-primary' : 'bg-background text-muted-foreground hover:bg-accent'"
                >
                  <input
                    type="checkbox"
                    :value="scope"
                    class="sr-only"
                    @change="(e: any) => {
                      if (e.target.checked) editForm.scopes.push(scope)
                      else editForm.scopes = editForm.scopes.filter(s => s !== scope)
                    }"
                  />
                  {{ scope }}
                </label>
              </div>
            </div>

            <!-- Grant Types -->
            <div>
              <label class="block text-sm font-medium text-foreground mb-2">Grant Types</label>
              <div class="space-y-2">
                <label
                  v-for="gt in availableGrantTypes"
                  :key="gt"
                  class="flex items-center gap-2 px-3 py-2 rounded-md border border-border text-sm cursor-pointer transition-colors"
                  :class="editForm.grant_types.includes(gt) ? 'bg-primary/10 border-primary text-primary' : 'bg-background text-muted-foreground hover:bg-accent'"
                >
                  <input
                    type="checkbox"
                    :value="gt"
                    class="sr-only"
                    @change="(e: any) => {
                      if (e.target.checked) editForm.grant_types.push(gt)
                      else editForm.grant_types = editForm.grant_types.filter(g => g !== gt)
                    }"
                  />
                  <span class="w-4 h-4 rounded border flex items-center justify-center shrink-0" :class="editForm.grant_types.includes(gt) ? 'bg-primary border-primary' : 'border-border'">
                    <Check v-if="editForm.grant_types.includes(gt)" class="w-3 h-3 text-primary-foreground" />
                  </span>
                  {{ gt === 'client_credentials' ? 'Client Credentials' : 'Authorization Code' }}
                </label>
              </div>
            </div>

            <!-- Enabled Toggle -->
            <div class="flex items-center justify-between">
              <label class="text-sm font-medium text-foreground">Enabled</label>
              <button
                class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
                :class="editForm.enabled ? 'bg-primary' : 'bg-muted'"
                @click="editForm.enabled = !editForm.enabled"
              >
                <span
                  class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
                  :class="editForm.enabled ? 'translate-x-5' : 'translate-x-0'"
                />
              </button>
            </div>
          </div>

          <div class="flex justify-end gap-2 mt-6">
            <Button variant="outline" size="sm" @click="editDialogOpen = false">Cancel</Button>
            <Button size="sm" :loading="editing" :disabled="!editForm.name.trim()" @click="handleEdit">
              <template v-if="!editing" #icon>
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
          <h2 class="text-lg font-semibold text-foreground mb-2">Delete OAuth2 Application</h2>
          <p class="text-sm text-muted-foreground mb-4">
            Are you sure you want to delete "{{ deletingClient?.name }}"? This action cannot be undone. All tokens associated with this application will be revoked.
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

    <!-- Regenerate Secret Confirmation Dialog -->
    <Teleport to="body">
      <div v-if="regenerateDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-sm p-6 border border-border">
          <h2 class="text-lg font-semibold text-foreground mb-2">Regenerate Client Secret</h2>
          <p class="text-sm text-muted-foreground mb-4">
            Are you sure you want to regenerate the client secret for "{{ regeneratingClient?.name }}"? The current secret will be immediately invalidated.
          </p>
          <div class="flex justify-end gap-2">
            <Button variant="outline" size="sm" @click="regenerateDialogOpen = false">Cancel</Button>
            <Button variant="destructive" size="sm" :loading="regenerating" @click="confirmRegenerate">
              <template v-if="!regenerating" #icon>
                <RefreshCw class="w-3.5 h-3.5" />
              </template>
              Regenerate
            </Button>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Secret Reveal Dialog -->
    <Teleport to="body">
      <div v-if="secretDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-md p-6 border border-border">
          <h2 class="text-lg font-semibold text-foreground mb-2">Client Secret Created</h2>
          <p class="text-sm text-destructive mb-3 font-medium">
            Copy this secret now. It will not be shown again!
          </p>
          <div class="flex items-center gap-2 rounded-md border border-border bg-muted/50 p-3">
            <code class="flex-1 text-xs font-mono text-foreground break-all select-all">{{ revealedSecret }}</code>
            <Button variant="ghost" size="icon" class="h-8 w-8 shrink-0" @click="copySecret">
              <Check v-if="clientSecretCopied" class="w-4 h-4 text-success" />
              <Copy v-else class="w-4 h-4" />
            </Button>
          </div>
          <div class="flex justify-end mt-4">
            <Button size="sm" @click="closeSecretDialog">Done</Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
