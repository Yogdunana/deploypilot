<script setup lang="ts">
import { ref } from 'vue'
import { Check, Copy } from 'lucide-vue-next'
import Button from '@/components/ui/Button.vue'

const props = defineProps<{
  open: boolean
  secret: string
}>()

const emit = defineEmits<{
  'update:open': [value: boolean]
  copy: []
}>()

const clientSecretCopied = ref(false)
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div class="bg-card rounded-lg shadow-xl w-full max-w-md p-6 border border-border">
        <h2 class="text-lg font-semibold text-foreground mb-2">Client Secret Created</h2>
        <p class="text-sm text-destructive mb-3 font-medium">
          Copy this secret now. It will not be shown again!
        </p>
        <div class="flex items-center gap-2 rounded-md border border-border bg-muted/50 p-3">
          <code class="flex-1 text-xs font-mono text-foreground break-all select-all">{{ secret }}</code>
          <Button variant="ghost" size="icon" class="h-8 w-8 shrink-0" @click="emit('copy')">
            <Check v-if="clientSecretCopied" class="w-4 h-4 text-success" />
            <Copy v-else class="w-4 h-4" />
          </Button>
        </div>
        <div class="flex justify-end mt-4">
          <Button size="sm" @click="emit('update:open', false)">Done</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>
