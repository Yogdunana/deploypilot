<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AlertDialog from '@/components/ui/AlertDialog.vue'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  appName: string
  currentVersion: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  confirm: []
}>()
</script>

<template>
  <AlertDialog
    :open="open"
    :title="t('appDetail.rollbackConfirm')"
    :description="t('appDetail.rollbackConfirmDesc', { name: appName, version: currentVersion })"
    :confirm-text="t('appDetail.confirmRollback')"
    :cancel-text="t('common.cancel')"
    variant="destructive"
    @update:open="emit('update:open', $event)"
    @confirm="emit('confirm')"
  />
</template>
