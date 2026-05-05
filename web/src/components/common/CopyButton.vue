<script setup lang="ts">
import { ref } from 'vue'
import { cn } from '@/lib/utils'
import { Check, Copy } from 'lucide-vue-next'

interface Props {
  text?: string
  class?: string
}

const props = defineProps<Props>()

const copied = ref(false)

async function copy() {
  if (!props.text) return
  try {
    await navigator.clipboard.writeText(props.text)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch {
    // fallback
    const textarea = document.createElement('textarea')
    textarea.value = props.text
    document.body.appendChild(textarea)
    textarea.select()
    document.execCommand('copy')
    document.body.removeChild(textarea)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  }
}
</script>

<template>
  <button
    type="button"
    :class="cn(
      'inline-flex items-center justify-center rounded-md p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors duration-150 cursor-pointer',
      props.class
    )"
    title="复制"
    @click="copy"
  >
    <Check v-if="copied" class="w-3.5 h-3.5 text-success" />
    <Copy v-else class="w-3.5 h-3.5" />
  </button>
</template>
