<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Eye, EyeOff } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Select from '@/components/ui/Select.vue'
import Textarea from '@/components/ui/Textarea.vue'
import Dialog from '@/components/ui/Dialog.vue'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  title: string
  editingId: string | null
  formName: string
  formType: string
  formValue: string
  formExpiresInDays: number
  showValue: boolean
  submitting: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  'update:formName': [value: string]
  'update:formType': [value: string]
  'update:formValue': [value: string]
  'update:formExpiresInDays': [value: number]
  'update:showValue': [value: boolean]
  submit: []
}>()

const typeOptions = computed(() => [
  { label: t('credentials.ssh'), value: 'ssh' },
  { label: t('credentials.apiKey'), value: 'api_key' },
  { label: t('credentials.token'), value: 'token' },
])
</script>

<template>
  <Dialog
    :open="open"
    :title="title"
    :description="t('credentials.configDesc')"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-4">
      <div class="space-y-2">
        <label class="text-sm font-medium text-foreground">{{ t('credentials.name') }}</label>
        <Input :model-value="formName" @update:model-value="emit('update:formName', $event)" :placeholder="t('credentials.namePlaceholder')" />
      </div>
      <div class="space-y-2">
        <label class="text-sm font-medium text-foreground">{{ t('credentials.type') }}</label>
        <Select :model-value="formType" @update:model-value="emit('update:formType', String($event))" :options="typeOptions" :placeholder="t('credentials.typePlaceholder')" />
      </div>
      <div class="space-y-2">
        <label class="text-sm font-medium text-foreground">{{ t('credentials.value') }}</label>
        <div class="relative">
          <Textarea
            :model-value="formValue"
            @update:model-value="emit('update:formValue', $event)"
            :type="showValue ? 'text' : 'password'"
            :placeholder="t('credentials.valuePlaceholder')"
            :rows="3"
            class="pr-10"
          />
          <button
            type="button"
            class="absolute right-3 top-3 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            @click="emit('update:showValue', !showValue)"
          >
            <EyeOff v-if="showValue" class="w-4 h-4" />
            <Eye v-else class="w-4 h-4" />
          </button>
        </div>
      </div>
      <div v-if="!editingId" class="space-y-2">
        <label class="text-sm font-medium text-foreground">{{ t('credentials.expiresInDays') }}</label>
        <Input
          :model-value="String(formExpiresInDays)"
          @update:model-value="emit('update:formExpiresInDays', Number($event))"
          type="number"
          :min="0"
          :placeholder="t('credentials.expiresInDaysPlaceholder')"
        />
        <p class="text-xs text-muted-foreground">{{ t('credentials.expiresInDaysPlaceholder') }}</p>
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <Button variant="outline" @click="emit('update:open', false)">{{ t('common.cancel') }}</Button>
        <Button :loading="submitting" @click="emit('submit')">
          {{ editingId ? t('common.saveText') : t('common.createText') }}
        </Button>
      </div>
    </div>
  </Dialog>
</template>
