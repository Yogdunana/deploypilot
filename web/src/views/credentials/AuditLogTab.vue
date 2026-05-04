<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Badge from '@/components/ui/Badge.vue'
import Skeleton from '@/components/ui/Skeleton.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'

const { t } = useI18n()

const props = defineProps<{
  auditLogs: any[]
  loading: boolean
}>()
</script>

<template>
  <div class="space-y-3">
    <div v-if="loading" class="space-y-2">
      <Skeleton v-for="i in 3" :key="i" class="h-4 w-full" />
    </div>
    <div v-else-if="auditLogs.length > 0" class="space-y-2">
      <div v-for="log in auditLogs" :key="log.id" class="flex items-start gap-3 rounded-md border border-border p-3 text-sm">
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2">
            <Badge variant="outline" class="text-xs">{{ log.action }}</Badge>
            <span class="text-muted-foreground">{{ log.username }}</span>
          </div>
          <p class="text-xs text-muted-foreground mt-1">{{ log.detail }}</p>
        </div>
        <RelativeTime :date="log.created_at" class="text-xs text-muted-foreground shrink-0" />
      </div>
    </div>
    <p v-else class="text-sm text-muted-foreground text-center py-4">{{ t('credentials.noAuditLogs') }}</p>
  </div>
</template>
