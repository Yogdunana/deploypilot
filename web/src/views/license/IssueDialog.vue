<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Button from '@/components/ui/Button.vue'
import Dialog from '@/components/ui/Dialog.vue'
import Input from '@/components/ui/Input.vue'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  issuing: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  issue: [data: { tenant: string; tier: string; use_type: string; duration_days: number }]
}>()

const tenant = ref('')
const tier = ref('community')
const useType = ref('non-commercial')
const durationDays = ref(365)

function handleIssue() {
  emit('issue', {
    tenant: tenant.value,
    tier: tier.value,
    use_type: useType.value,
    duration_days: durationDays.value,
  })
}
</script>

<template>
  <Dialog
    :open="open"
    :title="t('license.issueLicense')"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-4">
      <div class="space-y-2">
        <label class="text-sm font-medium text-foreground">{{ t('license.tenant') }}</label>
        <Input
          v-model="tenant"
          :placeholder="t('common.namePlaceholder')"
        />
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('license.tier') }}</label>
          <Input
            v-model="tier"
            placeholder="community"
          />
        </div>
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground">{{ t('license.useType') }}</label>
          <Input
            v-model="useType"
            placeholder="non-commercial"
          />
        </div>
      </div>
      <div class="space-y-2">
        <label class="text-sm font-medium text-foreground">{{ t('common.days', { count: '' }) }}</label>
        <Input
          v-model.number="durationDays"
          type="number"
          placeholder="365"
        />
      </div>
      <div class="flex justify-end gap-2 pt-2">
        <Button variant="outline" @click="emit('update:open', false)">{{ t('common.cancel') }}</Button>
        <Button :loading="issuing" @click="handleIssue">
          {{ t('license.issueLicense') }}
        </Button>
      </div>
    </div>
  </Dialog>
</template>
