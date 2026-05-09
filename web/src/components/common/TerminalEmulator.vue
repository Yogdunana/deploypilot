<script setup lang="ts">
import { ref, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { useWebSocket } from '@/composables/useWebSocket'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

interface Props {
  serverId: string
  onStatusChange?: (status: 'connecting' | 'connected' | 'disconnected' | 'error') => void
}

const props = defineProps<Props>()

const terminalRef = ref<HTMLElement | null>(null)
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null

function decodeBase64(data: string): string {
  try {
    return atob(data)
  } catch {
    return data
  }
}

const { connected, reconnecting, send, disconnect, connect } = useWebSocket({
  path: `/ws/terminal/${props.serverId}`,
  onMessage(data: any) {
    if (!terminal) return

    if (typeof data === 'string') {
      terminal.write(data)
    } else if (data.type === 'output') {
      // Output is base64 encoded from backend
      terminal.write(decodeBase64(data.data || ''))
    } else if (data.type === 'error') {
      terminal.write(`\x1b[31m${data.data || 'Error'}\x1b[0m\r\n`)
      props.onStatusChange?.('error')
    } else if (data.type === 'connected') {
      terminal.clear()
      terminal.write('\x1b[32m✓ Connected to server\x1b[0m\r\n')
      props.onStatusChange?.('connected')
    } else if (data.type === 'disconnected') {
      terminal.write(`\r\n\x1b[33m⚠ Disconnected: ${data.data || 'unknown'}\x1b[0m\r\n`)
      props.onStatusChange?.('disconnected')
    }
  },
  onOpen() {
    props.onStatusChange?.('connecting')
  },
  onClose() {
    if (terminal) {
      terminal.write('\r\n\x1b[33mConnection closed\x1b[0m\r\n')
    }
    props.onStatusChange?.('disconnected')
  },
})

function sendResize() {
  if (!terminal || !fitAddon || !connected.value) return
  try {
    send({
      type: 'resize',
      data: {
        rows: terminal.rows,
        cols: terminal.cols,
      },
    })
  } catch {
    // ignore send errors
  }
}

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

  // Load addons
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  // Send keystrokes to WebSocket
  terminal.onData((data) => {
    send({ type: 'input', data })
  })

  // Send resize on terminal size change
  terminal.onResize(() => {
    sendResize()
  })

  // Mount to DOM
  terminal.open(terminalRef.value)

  // Fit to container
  nextTick(() => {
    fitAddon?.fit()
  })

  // Watch container size changes
  resizeObserver = new ResizeObserver(() => {
    fitAddon?.fit()
    // Debounce resize message
    setTimeout(sendResize, 100)
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

// Watch serverId changes
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

// Expose methods
defineExpose({
  disconnect,
  connect,
  connected,
  reconnecting,
  fit: () => fitAddon?.fit(),
})
</script>

<template>
  <div class="relative w-full h-full bg-[#0a0a0a] rounded-lg overflow-hidden">
    <div ref="terminalRef" class="w-full h-full" />
  </div>
</template>
