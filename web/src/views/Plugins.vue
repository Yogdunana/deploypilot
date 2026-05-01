<script setup lang="ts">
import { ref } from 'vue'
import { Puzzle, Plus, Download, Power, Settings } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import Card from '@/components/ui/Card.vue'
import Badge from '@/components/ui/Badge.vue'
import Switch from '@/components/ui/Switch.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)

// TODO: Implement plugin management when backend API is ready
// import * as pluginsApi from '@/api/modules/plugins'
</script>

<template>
  <div>
    <PageHeader
      :title="t('plugins.title')"
      :description="t('plugins.description')"
    >
      <Button disabled>
        <Plus class="w-4 h-4 mr-2" />
        {{ t('plugins.install') }}
      </Button>
    </PageHeader>

    <EmptyState
      :icon="Puzzle"
      :title="t('plugins.emptyTitle')"
      :description="t('plugins.emptyDescription')"
    >
      <template #action>
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <Settings class="w-4 h-4" />
          <span>{{ t('plugins.comingSoon') }}</span>
        </div>
      </template>
    </EmptyState>
  </div>
</template>
