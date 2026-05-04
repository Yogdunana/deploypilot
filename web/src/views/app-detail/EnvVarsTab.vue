<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { Eye, EyeOff, Trash2, Plus, Save } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'
import Input from '@/components/ui/Input.vue'
import Skeleton from '@/components/ui/Skeleton.vue'

const { t } = useI18n()

export interface EnvItem {
  key: string
  value: string
  visible: boolean
  isNew?: boolean
}

const props = defineProps<{
  envList: EnvItem[]
  loading: boolean
  saving: boolean
}>()

const emit = defineEmits<{
  add: []
  remove: [index: number]
  toggleVisibility: [index: number]
  save: []
}>()
</script>

<template>
  <div class="space-y-3">
    <div class="flex items-center justify-between">
      <p class="text-sm text-muted-foreground">
        {{ t('appDetail.totalVars', { count: envList.length }) }}
      </p>
      <div class="flex items-center gap-2">
        <Button :loading="saving" size="sm" @click="emit('save')">
          <template #icon><Save class="w-4 h-4" /></template>
          {{ t('appDetail.save') }}
        </Button>
      </div>
    </div>

    <div v-if="loading" class="space-y-3">
      <Skeleton v-for="i in 5" :key="i" class="h-10 w-full" />
    </div>

    <div v-else class="space-y-2">
      <div class="grid grid-cols-[1fr_1fr_80px] gap-3 px-1">
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('appDetail.key') }}</span>
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider">{{ t('appDetail.value') }}</span>
        <span class="text-xs font-medium text-muted-foreground uppercase tracking-wider text-center">{{ t('appDetail.actions') }}</span>
      </div>

      <div
        v-for="(item, index) in envList"
        :key="index"
        class="grid grid-cols-[1fr_1fr_80px] gap-3 items-center group"
      >
        <Input
          v-model="item.key"
          :placeholder="t('appDetail.varNamePlaceholder')"
          :class="item.isNew ? 'border-primary/50' : ''"
        />
        <div class="relative">
          <Input
            v-model="item.value"
            :type="item.visible ? 'text' : 'password'"
            :placeholder="t('appDetail.varValuePlaceholder')"
          />
          <button
            class="absolute right-2 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
            @click="emit('toggleVisibility', index)"
          >
            <Eye v-if="!item.visible" class="w-4 h-4" />
            <EyeOff v-else class="w-4 h-4" />
          </button>
        </div>
        <div class="flex justify-center">
          <Button
            variant="ghost"
            size="icon"
            class="h-8 w-8 text-muted-foreground hover:text-destructive"
            @click="emit('remove', index)"
          >
            <Trash2 class="w-4 h-4" />
          </Button>
        </div>
      </div>

      <Button variant="outline" size="sm" class="mt-2" @click="emit('add')">
        <template #icon><Plus class="w-4 h-4" /></template>
        {{ t('appDetail.addVar') }}
      </Button>
    </div>
  </div>
</template>
