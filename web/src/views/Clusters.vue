<script setup lang="ts">
import { ref } from 'vue'
import { Server, Globe, Plus } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)

// TODO: Implement cluster management when backend API is ready
// import * as clustersApi from '@/api/modules/clusters'
</script>

<template>
  <div>
    <PageHeader
      :title="t('clusters.title')"
      :description="t('clusters.description')"
    >
      <Button disabled>
        <Plus class="w-4 h-4 mr-2" />
        {{ t('clusters.create') }}
      </Button>
    </PageHeader>

    <EmptyState
      :icon="Server"
      :title="t('clusters.emptyTitle')"
      :description="t('clusters.emptyDescription')"
    >
      <template #action>
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <Globe class="w-4 h-4" />
          <span>{{ t('clusters.comingSoon') }}</span>
        </div>
      </template>
    </EmptyState>
  </div>
</template>
