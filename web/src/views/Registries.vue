<script setup lang="ts">
import { ref } from 'vue'
import { Container, Plus, Wifi } from 'lucide-vue-next'
import PageHeader from '@/components/common/PageHeader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import Button from '@/components/ui/Button.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const loading = ref(false)

// TODO: Implement registry management when backend API is ready
// import * as registriesApi from '@/api/modules/registries'
</script>

<template>
  <div>
    <PageHeader
      :title="t('registries.title')"
      :description="t('registries.description')"
    >
      <Button disabled>
        <Plus class="w-4 h-4 mr-2" />
        {{ t('registries.create') }}
      </Button>
    </PageHeader>

    <EmptyState
      :icon="Container"
      :title="t('registries.emptyTitle')"
      :description="t('registries.emptyDescription')"
    >
      <template #action>
        <div class="flex items-center gap-2 text-sm text-muted-foreground">
          <Wifi class="w-4 h-4" />
          <span>{{ t('registries.comingSoon') }}</span>
        </div>
      </template>
    </EmptyState>
  </div>
</template>
