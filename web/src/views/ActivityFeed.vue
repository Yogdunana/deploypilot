<script setup lang="ts">
import { ref } from 'vue'
import { Activity, Clock, Filter } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import RelativeTime from '@/components/common/RelativeTime.vue'
import StatusBadge from '@/components/common/StatusBadge.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)

// TODO: Implement activity feed when backend API is ready
// import * as activityApi from '@/api/modules/activity'
</script>

<template>
  <div>
    <PageHeader
      :title="t('activity.title')"
      :description="t('activity.description')"
    />

    <EmptyState
      :icon="Activity"
      :title="t('activity.emptyTitle')"
      :description="t('activity.emptyDescription')"
    >
      <template #action>
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <Clock class="w-4 h-4" />
          <span>{{ t('activity.comingSoon') }}</span>
        </div>
      </template>
    </EmptyState>
  </div>
</template>
