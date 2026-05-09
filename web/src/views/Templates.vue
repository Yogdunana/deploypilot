<script setup lang="ts">
import { ref, inject, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Plus, Pencil, Trash2, Rocket, FileCode } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Badge from '@/components/ui/Badge.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Select from '@/components/ui/Select.vue'
import Card from '@/components/ui/Card.vue'
import Dialog from '@/components/ui/Dialog.vue'
import AlertDialog from '@/components/ui/AlertDialog.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as templatesApi from '@/api/modules/templates'
import type { Template } from '@/types/models'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const { t } = useI18n()
const { toast } = inject<any>('toast')!

// State
const templates = ref<Template[]>([])
const loading = ref(true)

// Dialog
const dialogOpen = ref(false)
const dialogTitle = ref(t('templates.createTitle'))
const editingId = ref<number | null>(null)
const formName = ref('')
const formType = ref('')
const formDescription = ref('')
const formBuildCmd = ref('')
const formRunCmd = ref('')
const formPort = ref('')
const submitting = ref(false)

// Delete dialog
const deleteDialogOpen = ref(false)
const deletingItem = ref<Template | null>(null)
const deleting = ref(false)

// Type options
const typeOptions = [
  { label: 'Docker', value: 'docker' },
  { label: 'Node.js', value: 'node' },
  { label: 'Python', value: 'python' },
  { label: 'Go', value: 'go' },
  { label: 'Java', value: 'java' },
  { label: 'PHP', value: 'php' },
  { label: 'Ruby', value: 'ruby' },
  { label: 'Rust', value: 'rust' },
  { label: 'Static', value: 'static' },
]

// Type badge mapping
function getTypeBadge(type: string) {
  const map: Record<string, { variant: 'default' | 'secondary' | 'outline' | 'success' | 'warning'; label: string }> = {
    docker: { variant: 'default', label: 'Docker' },
    node: { variant: 'success', label: 'Node.js' },
    python: { variant: 'warning', label: 'Python' },
    go: { variant: 'outline', label: 'Go' },
    java: { variant: 'secondary', label: 'Java' },
    php: { variant: 'secondary', label: 'PHP' },
    ruby: { variant: 'secondary', label: 'Ruby' },
    rust: { variant: 'warning', label: 'Rust' },
    static: { variant: 'outline', label: 'Static' },
  }
  return map[type] || { variant: 'secondary' as const, label: type }
}

// Fetch templates
async function fetchTemplates() {
  loading.value = true
  try {
    const res = await templatesApi.list()
    if (res.data.status === 'success') {
      templates.value = res.data.data
    }
  } catch (err: any) {
    toast(err.response?.data?.message || t('templates.fetchFailed'), 'destructive')
  } finally {
    loading.value = false
  }
}

// Open create dialog
function openCreateDialog() {
  editingId.value = null
  dialogTitle.value = t('templates.createTitle')
  formName.value = ''
  formType.value = ''
  formDescription.value = ''
  formBuildCmd.value = ''
  formRunCmd.value = ''
  formPort.value = ''
  dialogOpen.value = true
}

// Open edit dialog
function openEditDialog(item: Template) {
  editingId.value = item.id
  dialogTitle.value = t('templates.editTitle')
  formName.value = item.name
  formType.value = item.tech_stack
  formDescription.value = item.description
  formBuildCmd.value = (item.config && item.config.build_cmd) || ''
  formRunCmd.value = (item.config && item.config.run_cmd) || ''
  formPort.value = (item.config && item.config.port) ? String(item.config.port) : ''
  dialogOpen.value = true
}

// Submit form
async function handleSubmit() {
  if (!formName.value || !formType.value) {
    toast(t('templates.nameTypeRequired'), 'destructive')
    return
  }
  submitting.value = true
  try {
    const config: Record<string, any> = {}
    if (formBuildCmd.value) config.build_cmd = formBuildCmd.value
    if (formRunCmd.value) config.run_cmd = formRunCmd.value
    if (formPort.value) config.port = parseInt(formPort.value)
    const data = {
      name: formName.value,
      description: formDescription.value,
      tech_stack: formType.value,
      deploy_mode: 'docker',
      config,
    }
    if (editingId.value) {
      await templatesApi.update(editingId.value, data)
      toast(t('templates.updated'), 'success')
    } else {
      await templatesApi.create(data)
      toast(t('templates.created'), 'success')
    }
    dialogOpen.value = false
    fetchTemplates()
  } catch (err: any) {
    toast(err.response?.data?.message || t('common.operationFailed'), 'destructive')
  } finally {
    submitting.value = false
  }
}

// Use template
function handleUseTemplate(item: Template) {
  router.push(`/apps/create?template=${item.id}`)
}

// Delete
function openDeleteDialog(item: Template) {
  deletingItem.value = item
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingItem.value) return
  deleting.value = true
  try {
    await templatesApi.deleteTemplate(deletingItem.value.id)
    toast(t('templates.deleted', { name: deletingItem.value.name }), 'success')
    fetchTemplates()
  } catch (err: any) {
    toast(err.response?.data?.message || t('templates.deleteFailed'), 'destructive')
  } finally {
    deleting.value = false
    deletingItem.value = null
  }
}

onMounted(fetchTemplates)
</script>

<template>
  <div class="p-6 space-y-4">
    <!-- Header -->
    <PageHeader :title="t('templates.title')">
      <template #actions>
        <Button @click="openCreateDialog">
          <template #icon><Plus class="w-4 h-4" /></template>
          {{ t('templates.createTemplate') }}
        </Button>
      </template>
    </PageHeader>

    <!-- Loading skeleton -->
    <div v-if="loading" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="i in 6" :key="i" class="rounded-lg border border-border bg-card p-5 space-y-3">
        <Skeleton class="h-5 w-32" />
        <Skeleton class="h-4 w-16" />
        <Skeleton class="h-4 w-full" />
        <Skeleton class="h-4 w-3/4" />
        <div class="flex gap-2 pt-2">
          <Skeleton class="h-4 w-20" />
          <Skeleton class="h-4 w-20" />
        </div>
      </div>
    </div>

    <!-- Card grid -->
    <div v-else-if="templates.length > 0" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card v-for="item in templates" :key="item.id" class="hover:border-primary/30 transition-colors">
        <div class="p-5 space-y-3">
          <!-- Header -->
          <div class="flex items-start justify-between">
            <div class="space-y-1">
              <h3 class="text-sm font-semibold text-foreground">{{ item.name }}</h3>
              <Badge :variant="getTypeBadge(item.tech_stack).variant" class="text-xs">
                {{ getTypeBadge(item.tech_stack).label }}
              </Badge>
            </div>
            <div class="flex items-center gap-1">
              <Button variant="ghost" size="icon" class="h-8 w-8" @click="openEditDialog(item)">
                <Pencil class="w-3.5 h-3.5" />
              </Button>
              <Button variant="ghost" size="icon" class="h-8 w-8" @click="openDeleteDialog(item)">
                <Trash2 class="w-3.5 h-3.5 text-destructive" />
              </Button>
            </div>
          </div>

          <!-- Description -->
          <p v-if="item.description" class="text-sm text-muted-foreground line-clamp-2">
            {{ item.description }}
          </p>
          <p v-else class="text-sm text-muted-foreground italic">{{ t('templates.noDescription') }}</p>

          <!-- Config details -->
          <div class="space-y-1.5 text-xs text-muted-foreground">
            <div v-if="item.config?.build_cmd" class="flex items-center gap-2">
              <span class="text-foreground/60 font-medium">{{ t('templates.build') }}:</span>
              <code class="font-mono bg-accent/50 px-1.5 py-0.5 rounded text-foreground">{{ item.config.build_cmd }}</code>
            </div>
            <div v-if="item.config?.run_cmd" class="flex items-center gap-2">
              <span class="text-foreground/60 font-medium">{{ t('templates.run') }}:</span>
              <code class="font-mono bg-accent/50 px-1.5 py-0.5 rounded text-foreground">{{ item.config.run_cmd }}</code>
            </div>
            <div v-if="item.config?.port" class="flex items-center gap-2">
              <span class="text-foreground/60 font-medium">{{ t('templates.port') }}:</span>
              <code class="font-mono bg-accent/50 px-1.5 py-0.5 rounded text-foreground">{{ item.config.port }}</code>
            </div>
          </div>

          <!-- Use button -->
          <Button variant="outline" size="sm" class="w-full mt-2" @click="handleUseTemplate(item)">
            <template #icon><Rocket class="w-3.5 h-3.5" /></template>
            {{ t('templates.useTemplate') }}
          </Button>
        </div>
      </Card>
    </div>

    <!-- Empty state -->
    <EmptyState
      v-else
      :icon="FileCode"
      :title="t('templates.noTemplates')"
      :description="t('templates.noTemplatesDesc')"
      :action-text="t('templates.createTemplate')"
      @action="openCreateDialog"
    />

    <!-- Create/Edit Dialog -->
    <Dialog
      v-model:open="dialogOpen"
      :title="dialogTitle"
      :description="t('templates.configDesc')"
    >
      <div class="space-y-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('templates.name') }}</label>
          <Input v-model="formName" :placeholder="t('templates.namePlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('templates.type') }}</label>
          <Select v-model="formType" :options="typeOptions" :placeholder="t('templates.typePlaceholder')" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('templates.description') }}</label>
          <Textarea v-model="formDescription" :placeholder="t('templates.descriptionPlaceholder')" :rows="3" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('templates.buildCmd') }}</label>
          <Input v-model="formBuildCmd" :placeholder="t('templates.buildCmdPlaceholder')" class="font-mono text-xs" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('templates.runCmd') }}</label>
          <Input v-model="formRunCmd" :placeholder="t('templates.runCmdPlaceholder')" class="font-mono text-xs" />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('templates.port') }}</label>
          <Input v-model="formPort" type="number" :placeholder="t('templates.portPlaceholder')" class="font-mono text-xs" />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <Button variant="outline" @click="dialogOpen = false">{{ t('common.cancel') }}</Button>
          <Button :loading="submitting" @click="handleSubmit">
            {{ editingId ? t('common.saveText') : t('common.createText') }}
          </Button>
        </div>
      </div>
    </Dialog>

    <!-- Delete AlertDialog -->
    <AlertDialog
      v-model:open="deleteDialogOpen"
      :title="t('templates.deleteConfirm')"
      :description="t('templates.deleteConfirmDesc', { name: deletingItem?.name || '' })"
      :confirm-text="t('templates.delete')"
      :cancel-text="t('common.cancel')"
      variant="destructive"
      @confirm="confirmDelete"
    />
  </div>
</template>
