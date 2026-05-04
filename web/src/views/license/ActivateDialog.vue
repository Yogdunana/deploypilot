<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Key } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Dialog from '@/components/ui/Dialog.vue'
import Input from '@/components/ui/Input.vue'
import Switch from '@/components/ui/Switch.vue'

const { t } = useI18n()

const props = defineProps<{
  open: boolean
  activating: boolean
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  activate: [data: { license_key?: string; use_type?: string; agree_terms?: boolean }]
}>()

const licenseKey = ref('')
const useType = ref('non-commercial')
const agreeTerms = ref(false)

function handleActivate() {
  const data: { license_key?: string; use_type?: string; agree_terms?: boolean } = {}
  if (licenseKey.value.trim()) {
    data.license_key = licenseKey.value.trim()
  } else {
    data.use_type = useType.value
    data.agree_terms = agreeTerms.value
  }
  emit('activate', data)
}
</script>

<template>
  <Dialog
    :open="open"
    :title="t('license.activateTitle')"
    :description="t('license.activateDesc')"
    @update:open="emit('update:open', $event)"
  >
    <div class="space-y-4">
      <!-- License key input -->
      <div class="space-y-2">
        <label class="text-sm font-medium text-foreground">{{ t('license.licenseKey') }}</label>
        <Input
          v-model="licenseKey"
          :placeholder="t('license.licenseKeyPlaceholder')"
        />
      </div>

      <!-- Divider -->
      <div class="relative">
        <div class="absolute inset-0 flex items-center">
          <span class="w-full border-t border-border" />
        </div>
        <div class="relative flex justify-center text-xs">
          <span class="bg-card px-2 text-muted-foreground">{{ t('license.orCommunity') }}</span>
        </div>
      </div>

      <!-- Community license -->
      <div class="rounded-lg border border-border p-4 space-y-3">
        <h4 class="text-sm font-medium text-foreground">{{ t('license.communityTitle') }}</h4>
        <p class="text-xs text-muted-foreground">{{ t('license.communityDesc') }}</p>

        <!-- Legal notice -->
        <div class="rounded-md bg-accent/50 p-3 space-y-2">
          <h5 class="text-xs font-medium text-foreground">{{ t('license.legalNotice') }}</h5>
          <p class="text-xs text-muted-foreground leading-relaxed">{{ t('license.legalText') }}</p>
        </div>

        <label class="flex items-start gap-2 cursor-pointer">
          <Switch v-model="agreeTerms" class="mt-0.5" />
          <span class="text-xs text-foreground">{{ t('license.agreeTerms') }}</span>
        </label>
      </div>

      <div class="flex justify-end gap-2 pt-2">
        <Button variant="outline" @click="emit('update:open', false)">{{ t('common.cancel') }}</Button>
        <Button
          :loading="activating"
          :disabled="!licenseKey.trim() && !agreeTerms"
          @click="handleActivate"
        >
          <template #icon><Key class="w-4 h-4" /></template>
          {{ activating ? t('license.activating') : t('license.activate') }}
        </Button>
      </div>
    </div>
  </Dialog>
</template>
