<script setup lang="ts">
import { Copy, Pencil, RefreshCw, Trash2 } from 'lucide-vue-next'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import type { OAuth2Client } from '@/api/modules/oauth2'

const props = defineProps<{
  clients: OAuth2Client[]
}>()

const emit = defineEmits<{
  edit: [client: OAuth2Client]
  regenerate: [client: OAuth2Client]
  delete: [client: OAuth2Client]
  copyClientId: [clientId: string]
}>()

function truncateClientId(id: string, maxLen = 24): string {
  if (id.length <= maxLen) return id
  return id.substring(0, maxLen) + '...'
}
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
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
            @click="emit('copyClientId', client.client_id)"
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
        <Button variant="ghost" size="sm" class="h-7 text-xs" @click="emit('edit', client)">
          <template #icon>
            <Pencil class="w-3.5 h-3.5" />
          </template>
          Edit
        </Button>
        <Button variant="ghost" size="sm" class="h-7 text-xs" @click="emit('regenerate', client)">
          <template #icon>
            <RefreshCw class="w-3.5 h-3.5" />
          </template>
          Regenerate
        </Button>
        <div class="flex-1" />
        <Button variant="ghost" size="icon" class="h-7 w-7 text-destructive hover:text-destructive hover:bg-destructive/10" @click="emit('delete', client)">
          <Trash2 class="w-3.5 h-3.5" />
        </Button>
      </div>
    </Card>
  </div>
</template>
