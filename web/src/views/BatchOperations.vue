<script setup lang="ts">
import { ref } from 'vue'
import { PlaySquare, Layers, CheckCircle, XCircle } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Progress from '@/components/ui/Progress.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)

// TODO: Implement batch operations when backend API is ready
// import * as batchApi from '@/api/modules/batch'
</script>

<template>
  <div>
    <PageHeader
      :title="t('batch.title')"
      :description="t('batch.description')"
    />

    <EmptyState
      :icon="Layers"
      :title="t('batch.emptyTitle')"
      :description="t('batch.emptyDescription')"
    >
      <template #action>
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <PlaySquare class="w-4 h-4" />
          <span>{{ t('batch.comingSoon') }}</span>
        </div>
      </template>
    </EmptyState>
  </div>
</template>
