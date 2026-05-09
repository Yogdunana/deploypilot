<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useToast } from '@/composables/useToast'
import { useRouter, useRoute } from 'vue-router'
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Loader2,
  Send,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Button from '@/components/ui/Button.vue'
import Badge from '@/components/ui/Badge.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import * as webhookApi from '@/api/modules/outbound_webhook'
import type { WebhookDelivery } from '@/api/modules/outbound_webhook'

const router = useRouter()
const route = useRoute()
const { toast } = useToast()

const webhookId = route.params.webhookId as string

const deliveries = ref<WebhookDelivery[]>([])
const loading = ref(false)
const error = ref('')
const page = ref(1)
const total = ref(0)
const pageSize = 20
const expandedId = ref<string | null>(null)

function toggleExpand(id: string) {
  expandedId.value = expandedId.value === id ? null : id
}

function formatRequestBody(body: string): string {
  try {
    return JSON.stringify(JSON.parse(body), null, 2)
  } catch {
    return body
  }
}

function formatTime(date: string): string {
  return new Date(date).toLocaleString()
}

const totalPages = ref(1)

async function fetchDeliveries() {
  loading.value = true
  error.value = ''
  try {
    const res = await webhookApi.listDeliveries(webhookId, page.value, pageSize)
    if (res.data.status === 'success') {
      deliveries.value = res.data.data.data
      total.value = res.data.data.total
      totalPages.value = Math.ceil(total.value / pageSize)
    }
  } catch (e: any) {
    error.value = e?.message || 'Failed to load deliveries'
  } finally {
    loading.value = false
  }
}

function prevPage() {
  if (page.value > 1) {
    page.value--
    expandedId.value = null
    fetchDeliveries()
  }
}

function nextPage() {
  if (page.value < totalPages.value) {
    page.value++
    expandedId.value = null
    fetchDeliveries()
  }
}

onMounted(fetchDeliveries)
</script>

<template>
  <div class="space-y-4">
    <PageHeader title="Delivery Log" :description="`Showing delivery history for webhook ${webhookId}`">
      <template #actions>
        <Button variant="ghost" size="sm" @click="router.push('/webhooks')">
          <template #icon>
            <ArrowLeft class="w-4 h-4" />
          </template>
          Back to Webhooks
        </Button>
      </template>
    </PageHeader>

    <!-- Loading State -->
    <div v-if="loading" class="space-y-3">
      <Skeleton v-for="i in 5" :key="i" class="h-12 w-full" />
    </div>

    <!-- Error State -->
    <div v-else-if="error" class="rounded-lg border border-destructive/30 bg-destructive/10 p-4 text-sm text-destructive">
      {{ error }}
    </div>

    <!-- Empty State -->
    <div v-else-if="deliveries.length === 0" class="flex flex-col items-center justify-center py-12 text-center">
      <Send class="w-12 h-12 mx-auto mb-3 text-muted-foreground" />
      <h3 class="text-sm font-medium text-foreground">No deliveries yet</h3>
      <p class="mt-1 text-sm text-muted-foreground">Delivery records will appear here when webhooks are triggered.</p>
    </div>

    <!-- Delivery Table -->
    <div v-else class="border border-border rounded-lg overflow-hidden">
      <table class="w-full text-sm">
        <thead class="bg-accent/50 border-b border-border">
          <tr>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider w-6"></th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Time</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Event Type</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Status Code</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Latency</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Attempt</th>
            <th class="px-4 py-3 text-left text-xs font-medium text-muted-foreground uppercase tracking-wider">Status</th>
          </tr>
        </thead>
        <tbody>
          <template v-for="delivery in deliveries" :key="delivery.id">
            <!-- Row -->
            <tr
              class="border-b border-border last:border-0 hover:bg-accent/30 cursor-pointer transition-colors"
              @click="toggleExpand(delivery.id)"
            >
              <td class="px-4 py-3 text-muted-foreground">
                <ChevronRight v-if="expandedId !== delivery.id" class="w-4 h-4" />
                <ChevronDown v-else class="w-4 h-4" />
              </td>
              <td class="px-4 py-3 text-muted-foreground whitespace-nowrap">
                {{ formatTime(delivery.created_at) }}
              </td>
              <td class="px-4 py-3">
                <Badge variant="secondary">{{ delivery.event_type }}</Badge>
              </td>
              <td class="px-4 py-3 font-mono">
                {{ delivery.status_code || '-' }}
              </td>
              <td class="px-4 py-3 text-muted-foreground">
                {{ delivery.latency_ms != null ? `${delivery.latency_ms}ms` : '-' }}
              </td>
              <td class="px-4 py-3 text-muted-foreground">
                {{ delivery.attempt }}
              </td>
              <td class="px-4 py-3">
                <Badge :variant="delivery.success ? 'success' : 'destructive'">
                  {{ delivery.success ? 'Success' : 'Failed' }}
                </Badge>
              </td>
            </tr>

            <!-- Expanded Detail -->
            <tr v-if="expandedId === delivery.id">
              <td colspan="7" class="px-4 py-3 bg-accent/20 border-b border-border">
                <div class="space-y-3">
                  <!-- Request Body -->
                  <div>
                    <h4 class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1.5">Request Body</h4>
                    <pre class="bg-card border border-border rounded-md p-3 text-xs font-mono text-foreground overflow-auto max-h-48 whitespace-pre-wrap">{{ formatRequestBody(delivery.request_body || '{}') }}</pre>
                  </div>

                  <!-- Error Response -->
                  <div v-if="!delivery.success && delivery.error_response">
                    <h4 class="text-xs font-medium text-destructive uppercase tracking-wider mb-1.5">Error Response</h4>
                    <pre class="bg-destructive/5 border border-destructive/20 rounded-md p-3 text-xs font-mono text-destructive overflow-auto max-h-32 whitespace-pre-wrap">{{ delivery.error_response }}</pre>
                  </div>
                </div>
              </td>
            </tr>
          </template>
        </tbody>
      </table>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="flex items-center justify-between px-4 py-3 border-t border-border bg-accent/30">
        <p class="text-xs text-muted-foreground">
          Page {{ page }} of {{ totalPages }} ({{ total }} total)
        </p>
        <div class="flex gap-2">
          <Button variant="outline" size="sm" :disabled="page <= 1" @click="prevPage">
            Previous
          </Button>
          <Button variant="outline" size="sm" :disabled="page >= totalPages" @click="nextPage">
            Next
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
