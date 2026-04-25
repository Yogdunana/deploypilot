<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useWebSocket } from '@/composables/useWebSocket'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

interface Props {
  serverId: string
}

const props = defineProps<Props>()

const terminalRef = ref<HTMLElement | null>(null)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null

const { connected, reconnecting, send, disconnect, connect } = useWebSocket({
  path: `/ws/terminal/${props.serverId}`,
  onMessage(data: any) {
    if (!terminal) return

    if (typeof data === 'string') {
      terminal.write(data)
    } else if (data.type === 'output') {
      terminal.write(data.data || '')
    } else if (data.type === 'error') {
      terminal.write(`\x1b[31m${data.data || '错误'}\x1b[0m\r\n`)
    } else if (data.type === 'connected') {
      terminal.write('\x1b[32m已连接到服务器\x1b[0m\r\n')
    } else if (data.type === 'disconnected') {
      terminal.write('\x1b[31m已断开连接\x1b[0m\r\n')
    }
  },
  onOpen() {
    if (terminal) {
      terminal.write('\x1b[32m正在连接...\x1b[0m\r\n')
    }
  },
  onClose() {
    if (terminal) {
      terminal.write('\r\n\x1b[33m连接已关闭\x1b[0m\r\n')
    }
  },
})

function initTerminal() {
  if (!terminalRef.value || terminal) return

  terminal = new Terminal({
    theme: {
      background: '#0a0a0a',
      foreground: '#d4d4d4',
      cursor: '#ffffff',
      cursorAccent: '#0a0a0a',
      selectionBackground: '#264f78',
      selectionForeground: '#ffffff',
      black: '#0a0a0a',
      red: '#ef4444',
      green: '#22c55e',
      yellow: '#eab308',
      blue: '#3b82f6',
      magenta: '#a855f7',
      cyan: '#06b6d4',
      white: '#d4d4d4',
      brightBlack: '#555555',
      brightRed: '#f87171',
      brightGreen: '#4ade80',
      brightYellow: '#facc15',
      brightBlue: '#60a5fa',
      brightMagenta: '#c084fc',
      brightCyan: '#22d3ee',
      brightWhite: '#ffffff',
    },
    fontFamily: '"JetBrains Mono", "Fira Code", "Cascadia Code", Menlo, Monaco, "Courier New", monospace',
    fontSize: 14,
    lineHeight: 1.2,
    cursorBlink: true,
    cursorStyle: 'block',
    scrollback: 5000,
    allowProposedApi: true,
  })

  // 加载插件
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  // 用户输入发送到 WebSocket
  terminal.onData((data) => {
    send({ type: 'input', data })
  })

  // 挂载到 DOM
  terminal.open(terminalRef.value)

  // 适配大小
  nextTick(() => {
    fitAddon?.fit()
  })

  // 监听容器大小变化
  resizeObserver = new ResizeObserver(() => {
    fitAddon?.fit()
  })
  resizeObserver.observe(terminalRef.value)
}

function disposeTerminal() {
  if (resizeObserver) {
    resizeObserver.disconnect()
    resizeObserver = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
    fitAddon = null
  }
}

onMounted(() => {
  initTerminal()
})

onBeforeUnmount(() => {
  disposeTerminal()
  disconnect()
})

// 监听 serverId 变化
watch(() => props.serverId, (newId, oldId) => {
  if (newId !== oldId) {
    disconnect()
    disposeTerminal()
    nextTick(() => {
      initTerminal()
      connect()
    })
  }
})

// 暴露方法
defineExpose({
  disconnect,
  connect,
  connected,
  reconnecting,
})
</script>

<template>
  <div class="relative w-full h-full bg-[#0a0a0a] rounded-lg overflow-hidden">
    <div ref="terminalRef" class="w-full h-full" />
  </div>
</template>
