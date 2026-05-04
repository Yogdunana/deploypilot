<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { RefreshCw } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Dialog from '@/components/ui/Dialog.vue'
import type { Credential } from '@/types/models'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  rotatingItem: Credential | null
  rotateValue: string
  rotateSubmitting: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:rotateValue': [value: string]
  submit: []
}>()
</script>

<template>
  <Dialog
    :open="open"
    :title="t('credentials.rotateTitle')"
    :description="t('credentials.rotateDesc')"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-4">
      <div v-if="rotatingItem" class="rounded-lg bg-muted p-3 text-sm">
        <span class="font-medium">{{ rotatingItem.name }}</span>
        <span class="text-muted-foreground ml-2">({{ rotatingItem.type }})</span>
      </div>
      <div class="space-y-2">
        <label class="text-sm font-medium text-foreground">{{ t('credentials.value') }}</label>
        <Textarea
          :model-value="rotateValue"
          @update:model-value="emit('update:rotateValue', $event)"
          :placeholder="t('credentials.valuePlaceholder')"
          :rows="3"
        />
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <Button variant="outline" @click="emit('update:open', false)">{{ t('common.cancel') }}</Button>
        <Button :loading="rotateSubmitting" @click="emit('submit')">
          <template #icon><RefreshCw class="w-4 h-4" /></template>
          {{ t('credentials.rotate') }}
        </Button>
      </div>
    </div>
  </Dialog>
</template>
