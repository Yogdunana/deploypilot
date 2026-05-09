<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { useRouter } from 'vue-router'
import {
  Webhook,
  Plus,
  Pencil,
  Trash2,
  Send,
  Loader2,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as webhookApi from '@/api/modules/outbound_webhook'
import type { OutboundWebhook } from '@/api/modules/outbound_webhook'

const router = useRouter()
const { toast } = useToast()

const webhooks = ref<OutboundWebhook[]>([])
const loading = ref(false)
const error = ref('')

// Delete dialog state
const deleteDialogOpen = ref(false)
const deletingWebhook = ref<OutboundWebhook | null>(null)
const deleting = ref(false)

const formatColors: Record<string, string> = {
  json: 'bg-blue-100 text-blue-700',
  slack: 'bg-purple-100 text-purple-700',
  discord: 'bg-indigo-100 text-indigo-700',
  teams: 'bg-green-100 text-green-700',
}

function truncateUrl(url: string, maxLen = 40): string {
  if (url.length <= maxLen) return url
  return url.substring(0, maxLen) + '...'
}

async function fetchWebhooks() {
  loading.value = true
  error.value = ''
  try {
    const res = await webhookApi.listWebhooks()
    if (res.data.status === 'success') {
      webhooks.value = res.data.data.data
    }
  } catch (e: any) {
    error.value = e?.message || 'Failed to load webhooks'
  } finally {
    loading.value = false
  }
}

function openDeleteDialog(webhook: OutboundWebhook) {
  deletingWebhook.value = webhook
  deleteDialogOpen.value = true
}

async function confirmDelete() {
  if (!deletingWebhook.value) return
  deleting.value = true
  try {
    await webhookApi.deleteWebhook(deletingWebhook.value.id)
    toast('Webhook deleted', 'success')
    deleteDialogOpen.value = false
    deletingWebhook.value = null
    fetchWebhooks()
  } catch (e: any) {
    toast(e?.message || 'Failed to delete webhook', 'destructive')
  } finally {
    deleting.value = false
  }
}

onMounted(fetchWebhooks)
</script>

<template>
  <div class="space-y-4">
    <PageHeader title="Outbound Webhooks" description="Manage outbound webhook integrations">
      <template #actions>
        <Button @click="router.push('/webhooks/new')">
          <template #icon>
            <Plus class="w-4 h-4" />
          </template>
          Create Webhook
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
      v-else-if="webhooks.length === 0"
      :icon="Webhook"
      title="No webhooks configured"
      description="Create your first outbound webhook to receive event notifications."
      action-text="Create Webhook"
      @action="router.push('/webhooks/new')"
    />

    <!-- Webhook Cards -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <Card v-for="webhook in webhooks" :key="webhook.id" class="p-0">
        <div class="p-6 space-y-3">
          <!-- Header: Name + Enabled indicator -->
          <div class="flex items-start justify-between">
            <h3 class="text-sm font-semibold text-foreground truncate">{{ webhook.name }}</h3>
            <span
              class="inline-flex items-center gap-1.5 px-2 py-0.5 text-xs rounded-full shrink-0 ml-2"
              :class="webhook.enabled ? 'bg-success/15 text-success' : 'bg-accent text-muted-foreground'"
            >
              <span class="w-1.5 h-1.5 rounded-full" :class="webhook.enabled ? 'bg-success' : 'bg-muted-foreground'" />
              {{ webhook.enabled ? 'Enabled' : 'Disabled' }}
            </span>
          </div>

          <!-- URL -->
          <p class="text-xs font-mono text-muted-foreground truncate" :title="webhook.url">
            {{ truncateUrl(webhook.url) }}
          </p>

          <!-- Badges: Format + Last delivery status -->
          <div class="flex items-center gap-2 flex-wrap">
            <span
              class="inline-flex items-center px-2 py-0.5 text-xs rounded-full font-medium"
              :class="formatColors[webhook.format] || 'bg-gray-100 text-gray-700'"
            >
              {{ webhook.format.toUpperCase() }}
            </span>
            <span
              v-if="webhook.last_status"
              class="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-full"
              :class="webhook.last_status === 'success' ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'"
            >
              <span class="w-1.5 h-1.5 rounded-full" :class="webhook.last_status === 'success' ? 'bg-green-500' : 'bg-red-500'" />
              {{ webhook.last_status === 'success' ? 'Delivered' : 'Failed' }}
            </span>
          </div>

          <!-- Last delivery time -->
          <div v-if="webhook.last_delivery_at" class="text-xs text-muted-foreground">
            Last delivery: <RelativeTime :date="webhook.last_delivery_at" />
          </div>
        </div>

        <!-- Card Actions -->
        <div class="flex items-center border-t border-border px-4 py-2 gap-1">
          <Button variant="ghost" size="sm" class="h-7 text-xs" @click="router.push(`/webhooks/${webhook.id}/edit`)">
            <template #icon>
              <Pencil class="w-3.5 h-3.5" />
            </template>
            Edit
          </Button>
          <Button variant="ghost" size="sm" class="h-7 text-xs" @click="router.push(`/webhooks/${webhook.id}/deliveries`)">
            <template #icon>
              <Send class="w-3.5 h-3.5" />
            </template>
            Deliveries
          </Button>
          <div class="flex-1" />
          <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive hover:text-destructive hover:bg-destructive/10" @click="openDeleteDialog(webhook)">
            <Trash2 class="w-3.5 h-3.5" />
          </Button>
        </div>
      </Card>
    </div>

    <!-- Delete Confirmation Dialog -->
    <Teleport to="body">
      <div v-if="deleteDialogOpen" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div class="bg-card rounded-lg shadow-xl w-full max-w-sm p-6 border border-border">
          <h2 class="text-lg font-semibold text-foreground mb-2">Delete Webhook</h2>
          <p class="text-sm text-muted-foreground mb-4">
            Are you sure you want to delete "{{ deletingWebhook?.name }}"? This action cannot be undone.
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
