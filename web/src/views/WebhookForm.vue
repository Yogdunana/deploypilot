<script setup lang="ts">
import { ref, computed, onMounted, inject } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  ArrowLeft,
  Save,
  Send,
  Eye,
  EyeOff,
  Loader2,
} from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Input from '@/components/ui/Input.vue'
import Textarea from '@/components/ui/Textarea.vue'
import * as webhookApi from '@/api/modules/outbound_webhook'
import type { OutboundWebhook } from '@/api/modules/outbound_webhook'

const router = useRouter()
const route = useRoute()
const { toast } = inject<any>('toast')!

const isEdit = computed(() => !!route.params.id)
const pageTitle = computed(() => (isEdit.value ? 'Edit Webhook' : 'Create Webhook'))

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const showSecret = ref(false)

// Form fields
const formName = ref('')
const formUrl = ref('')
const formSecret = ref('')
const formFormat = ref<OutboundWebhook['format']>('json')
const formEventTypes = ref<string[]>(['deploy', 'alert'])
const formSeverityFilter = ref<string[]>([])
const formAppFilter = ref('')
const formServerFilter = ref('')
const formMaxRetries = ref(5)
const formTimeout = ref(10)
const formDescription = ref('')
const formEnabled = ref(true)

const formatOptions = [
  { label: 'JSON', value: 'json' },
  { label: 'Slack', value: 'slack' },
  { label: 'Discord', value: 'discord' },
  { label: 'Teams', value: 'teams' },
]

const eventTypeOptions = [
  { label: 'Deploy', value: 'deploy' },
  { label: 'Alert', value: 'alert' },
  { label: 'Notify', value: 'notify' },
  { label: 'System', value: 'system' },
  { label: 'User', value: 'user' },
  { label: 'Server', value: 'server' },
  { label: 'Security', value: 'security' },
  { label: 'Audit', value: 'audit' },
  { label: 'Backup', value: 'backup' },
]

const severityOptions = [
  { label: 'Critical', value: 'critical' },
  { label: 'Warning', value: 'warning' },
  { label: 'Info', value: 'info' },
]

function toggleEventType(value: string) {
  const idx = formEventTypes.value.indexOf(value)
  if (idx >= 0) {
    formEventTypes.value.splice(idx, 1)
  } else {
    formEventTypes.value.push(value)
  }
}

function toggleSeverity(value: string) {
  const idx = formSeverityFilter.value.indexOf(value)
  if (idx >= 0) {
    formSeverityFilter.value.splice(idx, 1)
  } else {
    formSeverityFilter.value.push(value)
  }
}

async function loadWebhook() {
  if (!route.params.id) return
  loading.value = true
  try {
    const res = await webhookApi.getWebhook(route.params.id as string)
    if (res.data.status === 'success') {
      const w = res.data.data
      formName.value = w.name
      formUrl.value = w.url
      formSecret.value = ''
      formFormat.value = w.format
      formEventTypes.value = [...w.event_types]
      formSeverityFilter.value = [...w.severity_filter]
      formAppFilter.value = w.app_filter.join(', ')
      formServerFilter.value = w.server_filter.join(', ')
      formMaxRetries.value = w.max_retries
      formTimeout.value = w.timeout
      formDescription.value = w.description
      formEnabled.value = w.enabled
    }
  } catch (e: any) {
    toast(e?.message || 'Failed to load webhook', 'destructive')
    router.push('/webhooks')
  } finally {
    loading.value = false
  }
}

function buildPayload(): Partial<OutboundWebhook> {
  return {
    name: formName.value.trim(),
    url: formUrl.value.trim(),
    format: formFormat.value,
    event_types: formEventTypes.value,
    severity_filter: formSeverityFilter.value,
    app_filter: formAppFilter.value
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean),
    server_filter: formServerFilter.value
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean),
    max_retries: formMaxRetries.value,
    timeout: formTimeout.value,
    description: formDescription.value.trim(),
    enabled: formEnabled.value,
    ...(formSecret.value.trim() ? { secret: formSecret.value.trim() } : {}),
  }
}

async function handleSave() {
  if (!formName.value.trim() || !formUrl.value.trim()) {
    toast('Name and URL are required', 'destructive')
    return
  }
  saving.value = true
  try {
    const payload = buildPayload()
    if (isEdit.value) {
      await webhookApi.updateWebhook(route.params.id as string, payload)
      toast('Webhook updated', 'success')
    } else {
      await webhookApi.createWebhook(payload)
      toast('Webhook created', 'success')
    }
    router.push('/webhooks')
  } catch (e: any) {
    toast(e?.message || 'Failed to save webhook', 'destructive')
  } finally {
    saving.value = false
  }
}

async function handleTest() {
  if (!isEdit.value) return
  testing.value = true
  try {
    const res = await webhookApi.testWebhook(route.params.id as string)
    if (res.data.status === 'success') {
      const delivery = res.data.data
      if (delivery.success) {
        toast(`Test delivery succeeded (${delivery.status_code}, ${delivery.latency_ms}ms)`, 'success')
      } else {
        toast(`Test delivery failed: ${delivery.error_response || 'Unknown error'}`, 'destructive')
      }
    }
  } catch (e: any) {
    toast(e?.message || 'Test delivery failed', 'destructive')
  } finally {
    testing.value = false
  }
}

onMounted(loadWebhook)
</script>

<template>
  <div class="space-y-4">
    <PageHeader :title="pageTitle">
      <template #actions>
        <Button variant="ghost" size="sm" @click="router.push('/webhooks')">
          <template #icon>
            <ArrowLeft class="w-4 h-4" />
          </template>
          Back
        </Button>
      </template>
    </PageHeader>

    <!-- Loading State -->
    <div v-if="loading" class="flex items-center justify-center py-12 text-muted-foreground">
      <Loader2 class="w-6 h-6 animate-spin mr-2" />
      Loading webhook...
    </div>

    <!-- Form -->
    <Card v-else class="max-w-2xl">
      <div class="space-y-6">
        <!-- Name -->
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-foreground">Name <span class="text-destructive">*</span></label>
          <Input v-model="formName" placeholder="e.g. Slack Notifications" />
        </div>

        <!-- URL -->
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-foreground">URL <span class="text-destructive">*</span></label>
          <Input v-model="formUrl" placeholder="https://hooks.slack.com/services/..." />
        </div>

        <!-- Secret -->
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-foreground">Secret</label>
          <div class="relative">
            <Input
              v-model="formSecret"
              :type="showSecret ? 'text' : 'password'"
              placeholder="Optional signing secret"
              class="pr-10"
            />
            <button
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground cursor-pointer"
              @click="showSecret = !showSecret"
            >
              <EyeOff v-if="showSecret" class="w-4 h-4" />
              <Eye v-else class="w-4 h-4" />
            </button>
          </div>
          <p class="text-xs text-muted-foreground">Leave blank to keep existing secret unchanged when editing.</p>
        </div>

        <!-- Format -->
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-foreground">Format</label>
          <select
            v-model="formFormat"
            class="flex h-9 w-full rounded-md border border-border bg-card px-3 py-1 text-sm text-foreground shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:border-primary"
          >
            <option v-for="opt in formatOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
        </div>

        <!-- Event Types -->
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">Event Types</label>
          <div class="grid grid-cols-3 gap-2">
            <label
              v-for="opt in eventTypeOptions"
              :key="opt.value"
              class="flex items-center gap-2 px-3 py-2 rounded-md border border-border cursor-pointer hover:bg-accent transition-colors"
              :class="formEventTypes.includes(opt.value) ? 'bg-primary/10 border-primary/30' : ''"
            >
              <input
                type="checkbox"
                :checked="formEventTypes.includes(opt.value)"
                class="rounded border-border text-primary focus:ring-primary"
                @change="toggleEventType(opt.value)"
              />
              <span class="text-sm text-foreground">{{ opt.label }}</span>
            </label>
          </div>
        </div>

        <!-- Severity Filter -->
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">Severity Filter <span class="text-xs text-muted-foreground">(optional)</span></label>
          <div class="flex gap-2">
            <label
              v-for="opt in severityOptions"
              :key="opt.value"
              class="flex items-center gap-2 px-3 py-2 rounded-md border border-border cursor-pointer hover:bg-accent transition-colors"
              :class="formSeverityFilter.includes(opt.value) ? 'bg-primary/10 border-primary/30' : ''"
            >
              <input
                type="checkbox"
                :checked="formSeverityFilter.includes(opt.value)"
                class="rounded border-border text-primary focus:ring-primary"
                @change="toggleSeverity(opt.value)"
              />
              <span class="text-sm text-foreground">{{ opt.label }}</span>
            </label>
          </div>
        </div>

        <!-- App Filter -->
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-foreground">App Filter <span class="text-xs text-muted-foreground">(optional)</span></label>
          <Input v-model="formAppFilter" placeholder="app1, app2, app3" />
          <p class="text-xs text-muted-foreground">Comma-separated app names. Leave empty for all apps.</p>
        </div>

        <!-- Server Filter -->
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-foreground">Server Filter <span class="text-xs text-muted-foreground">(optional)</span></label>
          <Input v-model="formServerFilter" placeholder="server1, server2, server3" />
          <p class="text-xs text-muted-foreground">Comma-separated server names. Leave empty for all servers.</p>
        </div>

        <!-- Max Retries & Timeout -->
        <div class="grid grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label class="text-sm font-medium text-foreground">Max Retries</label>
            <Input v-model="formMaxRetries" type="number" placeholder="5" />
          </div>
          <div class="space-y-1.5">
            <label class="text-sm font-medium text-foreground">Timeout (seconds)</label>
            <Input v-model="formTimeout" type="number" placeholder="10" />
          </div>
        </div>

        <!-- Description -->
        <div class="space-y-1.5">
          <label class="text-sm font-medium text-foreground">Description <span class="text-xs text-muted-foreground">(optional)</span></label>
          <Textarea v-model="formDescription" placeholder="Brief description of this webhook..." :rows="3" />
        </div>

        <!-- Actions -->
        <div class="flex items-center justify-between pt-2">
          <Button
            v-if="isEdit"
            variant="outline"
            :loading="testing"
            @click="handleTest"
          >
            <template #icon>
              <Send class="w-4 h-4" />
            </template>
            Test Delivery
          </Button>
          <div v-else />
          <div class="flex gap-2">
            <Button variant="outline" @click="router.push('/webhooks')">Cancel</Button>
            <Button :loading="saving" @click="handleSave">
              <template #icon>
                <Save class="w-4 h-4" />
              </template>
              {{ isEdit ? 'Save Changes' : 'Create Webhook' }}
            </Button>
          </div>
        </div>
      </div>
    </Card>
  </div>
</template>
