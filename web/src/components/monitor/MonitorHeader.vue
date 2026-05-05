<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { Maximize, Sun, Moon } from 'lucide-vue-next'

interface Props {
  connected: boolean
  monitorCount: number
  dark?: boolean
}

defineProps<Props>()

const emit = defineEmits<{
  toggleFullscreen: []
  toggleTheme: []
}>()

const clock = ref('')
let timer: ReturnType<typeof setInterval>

function updateClock() {
  clock.value = new Date().toLocaleTimeString(navigator.language || 'en-US', { hour12: false })
}

onMounted(() => {
  updateClock()
  timer = setInterval(updateClock, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="border-b border-gray-800 px-6 py-3">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-4">
        <h1 class="text-xl font-bold tracking-tight">DeployPilot Monitor</h1>
        <span class="text-xs px-2 py-1 rounded font-mono" :class="connected ? 'bg-green-900/50 text-green-400' : 'bg-red-900/50 text-red-400'">
          {{ connected ? '● LIVE' : '○ RECONNECTING' }}
        </span>
        <span class="text-xs text-gray-500">{{ monitorCount }} monitors</span>
      </div>
      <div class="flex items-center gap-2">
        <span class="text-lg font-mono text-gray-400 tabular-nums">{{ clock }}</span>
        <button class="p-1.5 rounded hover:bg-gray-800 text-gray-400 hover:text-white transition-colors" @click="emit('toggleTheme')">
          <Sun v-if="dark" class="w-4 h-4" />
          <Moon v-else class="w-4 h-4" />
        </button>
        <button class="p-1.5 rounded hover:bg-gray-800 text-gray-400 hover:text-white transition-colors" @click="emit('toggleFullscreen')">
          <Maximize class="w-4 h-4" />
        </button>
      </div>
    </div>
  </div>
</template>
