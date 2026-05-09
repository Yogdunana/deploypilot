<script setup lang="ts">
import { ref, onMounted, inject } from 'vue'
import {
  Puzzle,
  Play,
  Square,
  Pencil,
  Loader2,
  RefreshCw,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import Switch from '@/components/ui/Switch.vue'
import * as pluginApi from '@/api/modules/plugin'
import type { PluginInfo } from '@/api/modules/plugin'

const { toast } = inject<any>('toast')!

const plugins = ref<PluginInfo[]>([])
const loading = ref(false)
const error = ref('')

// Config dialog state
const configDialogOpen = ref(false)
const editingPlugin = ref<PluginInfo | null>(null)
const configText = ref('')
const configError = ref('')
const saving = ref(false)

// Action loading states
const actionLoading = ref<Record<string, boolean>>({})

const statusColors: Record<string, string> = {
  running: 'bg-green-100 text-green-700',
  stopped: 'bg-gray-100 text-gray-700',
  error: 'bg-red-100 text-red-700',
  initialized: 'bg-yellow-100 text-yellow-700',
  registered: 'bg-blue-100 text-blue-700',
}

const statusDotColors: Record<string, string> = {
  running: 'bg-green-500',
  stopped: 'bg-gray-500',
  error: 'bg-red-500',
  initialized: 'bg-yellow-500',
  registered: 'bg-blue-500',
}

function isActionLoading(name: string) {
  return actionLoading.value[name] || false
}

async function fetchPlugins() {
  loading.value = true
  error.value = ''
  try {
    const res = await pluginApi.listPlugins()
    if (res.data.status === 'success') {
      plugins.value = res.data.data
    }
  } catch (e: any) {
    error.value = e?.message || 'Failed to load plugins'
  } finally {
    loading.value = false
  }
}

async function toggleEnabled(plugin: PluginInfo) {
  actionLoading.value[plugin.name] = true
  try {
    await pluginApi.updatePlugin(plugin.name, { enabled: !plugin.enabled })
    toast(plugin.enabled ? 'Plugin disabled' : 'Plugin enabled', 'success')
    fetchPlugins()
  } catch (e: any) {
    toast(e?.message || 'Failed to update plugin', 'destructive')
  } finally {
    actionLoading.value[plugin.name] = false
  }
}

async function startPlugin(plugin: PluginInfo) {
  actionLoading.value[plugin.name] = true
  try {
    await pluginApi.startPlugin(plugin.name)
    toast('Plugin started', 'success')
    fetchPlugins()
  } catch (e: any) {
    toast(e?.message || 'Failed to start plugin', 'destructive')
  } finally {
    actionLoading.value[plugin.name] = false
  }
}

async function stopPlugin(plugin: PluginInfo) {
  actionLoading.value[plugin.name] = true
  try {
    await pluginApi.stopPlugin(plugin.name)
    toast('Plugin stopped', 'success')
    fetchPlugins()
  } catch (e: any) {
    toast(e?.message || 'Failed to stop plugin', 'destructive')
  } finally {
    actionLoading.value[plugin.name] = false
  }
}

function openConfigDialog(plugin: PluginInfo) {
  editingPlugin.value = plugin
  configText.value = JSON.stringify(plugin.config || {}, null, 2)
  configError.value = ''
  configDialogOpen.value = true
}

async function saveConfig() {
  if (!editingPlugin.value) return
  configError.value = ''
  try {
    const parsed = JSON.parse(configText.value)
    saving.value = true
    await pluginApi.updatePlugin(editingPlugin.value.name, { config: parsed })
    toast('Plugin config updated', 'success')
    configDialogOpen.value = false
    editingPlugin.value = null
    fetchPlugins()
  } catch (e: any) {
    if (e instanceof SyntaxError) {
      configError.value = 'Invalid JSON: ' + e.message
    } else {
      toast(e?.message || 'Failed to update config', 'destructive')
    }
  } finally {
    saving.value = false
  }
}

onMounted(fetchPlugins)
</script>

<template>
  <div class="space-y-4">
    <PageHeader title="Event Plugins" description="Manage event-driven plugins">
      <template #actions>
        <Button variant="outline" size="sm" :loading="loading" @click="fetchPlugins">
          <template #icon>
            <RefreshCw class="w-4 h-4" />
          </template>
          Refresh
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
      v-else-if="plugins.length === 0"
      :icon="Puzzle"
      title="No plugins found"
      description="No event plugins are currently registered in the system."
    />

    <!-- Plugin Cards -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card v-for="plugin in plugins" :key="plugin.name" class="p-0">
        <div class="p-6 space-y-3">
          <!-- Header: Name + Enabled toggle -->
          <div class="flex items-start justify-between">
            <h3 class="text-sm font-semibold text-foreground truncate">{{ plugin.name }}</h3>
            <Switch
              :model-value="plugin.enabled"
              :disabled="isActionLoading(plugin.name)"
              class="shrink-0 ml-2"
              @update:model-value="toggleEnabled(plugin)"
            />
          </div>

          <!-- Description -->
          <p v-if="plugin.description" class="text-xs text-muted-foreground line-clamp-2">
            {{ plugin.description }}
          </p>

          <!-- Badges: Version + Status -->
          <div class="flex items-center gap-2 flex-wrap">
            <span class="inline-flex items-center px-2 py-0.5 text-xs rounded-full bg-accent text-muted-foreground font-mono">
              v{{ plugin.version }}
            </span>
            <span
              class="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full font-medium"
              :class="statusColors[plugin.status] || 'bg-gray-100 text-gray-700'"
            >
              <span class="w-1.5 h-1.5 rounded-full" :class="statusDotColors[plugin.status] || 'bg-gray-500'" />
              {{ plugin.status }}
            </span>
          </div>

          <!-- Error message -->
          <div v-if="plugin.status === 'error' && plugin.error" class="rounded-md bg-red-50 border border-red-200 p-2 text-xs text-red-700">
            {{ plugin.error }}
          </div>
        </div>

        <!-- Card Actions -->
        <div class="flex items-center border-t border-border px-4 py-2 gap-1">
          <Button
            v-if="plugin.status !== 'running'"
            variant="ghost"
            size="sm"
            class="h-7 text-xs"
            :loading="isActionLoading(plugin.name)"
            @click="startPlugin(plugin)"
          >
            <template #icon>
              <Play class="w-3.5 h-3.5" />
            </template>
            Start
          </Button>
          <Button
            v-if="plugin.status === 'running'"
            variant="ghost"
            size="sm"
            class="h-7 text-xs"
            :loading="isActionLoading(plugin.name)"
            @click="stopPlugin(plugin)"
          >
            <template #icon>
              <Square class="w-3.5 h-3.5" />
            </template>
            Stop
          </Button>
          <Button variant="ghost" size="sm" class="h-7 text-xs" @click="openConfigDialog(plugin)">
            <template #icon>
              <Pencil class="w-3.5 h-3.5" />
            </template>
            Config
          </Button>
        </div>
      </Card>
    </div>

    <!-- Config Edit Dialog -->
    <Teleport to="body">
      <div v-if="configDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-lg p-6 border border-border">
          <h2 class="text-lg font-semibold text-foreground mb-1">Edit Plugin Config</h2>
          <p class="text-sm text-muted-foreground mb-4">
            Editing config for <span class="font-mono text-foreground">{{ editingPlugin?.name }}</span>
          </p>
          <textarea
            v-model="configText"
            class="w-full h-48 rounded-md border border-border bg-background px-3 py-2 text-sm font-mono text-foreground focus:outline-none focus:ring-2 focus:ring-ring resize-none"
            placeholder="{}"
          />
          <p v-if="configError" class="mt-2 text-xs text-destructive">{{ configError }}</p>
          <div class="flex justify-end gap-2 mt-4">
            <Button variant="outline" size="sm" @click="configDialogOpen = false">Cancel</Button>
            <Button variant="default" size="sm" :loading="saving" @click="saveConfig">
              <template v-if="!saving" #icon>
                <Pencil class="w-3.5 h-3.5" />
              </template>
              Save
            </Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>
