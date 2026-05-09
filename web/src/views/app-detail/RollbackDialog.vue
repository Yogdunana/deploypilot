<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AlertDialog from '@/components/ui/AlertDialog.vue'

defineProps<{
  open: boolean
  appName: string
  currentVersion: string
}>()

defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'confirm'): void
}>()

const { t } = useI18n()
</script>

<template>
  <AlertDialog
    :model-value="open"
    :title="t('appDetail.rollbackConfirm')"
    :description="t('appDetail.rollbackConfirmDesc', { name: appName, version: currentVersion })"
    :confirm-text="t('appDetail.confirmRollback')"
    :cancel-text="t('common.cancel')"
    variant="destructive"
    @update:model-value="$emit('update:open', $event)"
    @confirm="$emit('confirm')"
  />
</template>
