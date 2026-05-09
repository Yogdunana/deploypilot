import { ref, onUnmounted } from 'vue'

export interface UseWebSocketOptions {
  /** WebSocket 路径，如 /ws/logs/1 */
  path: string
  /** 最大重连次数，默认 10 */
  maxRetries?: number
  /** 最大重连间隔（毫秒），默认 30000 */
  maxRetryInterval?: number
  /** 初始重连间隔（毫秒），默认 1000 */
  initialRetryInterval?: number
  /** 收到消息时的回调 */
  onMessage?: (data: any) => void
  /** 连接打开时的回调 */
  onOpen?: () => void
  /** 连接关闭时的回调 */
  onClose?: (event: CloseEvent) => void
  /** 连接错误时的回调 */
  onError?: (event: Event) => void
  /** 是否自动连接，默认 true */
  autoConnect?: boolean
}

/** Fetch a one-time WebSocket ticket from the API. */
async function fetchWSTicket(): Promise<string> {
  const token = localStorage.getItem('deploypilot_token') || ''
  const resp = await fetch('/api/v1/auth/ws-ticket', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })
  if (!resp.ok) {
    throw new Error(`Failed to fetch WebSocket ticket: ${resp.status}`)
  }
  const body = await resp.json()
  if (body.status !== 'success' || !body.data?.ticket) {
    throw new Error('Invalid ticket response')
  }
  return body.data.ticket as string
}

export function useWebSocket(options: UseWebSocketOptions) {
  const {
    path,
    maxRetries = 10,
    maxRetryInterval = 30000,
    initialRetryInterval = 1000,
    onMessage,
    onOpen,
    onClose,
    onError,
    autoConnect = true,
  } = options

  const connected = ref(false)
  const reconnecting = ref(false)
  const retryCount = ref(0)

  let ws: WebSocket | null = null
  let retryTimer: ReturnType<typeof setTimeout> | null = null
  let manuallyClosed = false

  function buildUrl(ticket: string): string {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    return `${protocol}//${host}${path}?ticket=${encodeURIComponent(ticket)}`
  }

  async function connect() {
    if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
      return
    }

    manuallyClosed = false

    // Fetch a one-time ticket before connecting
    let ticket: string
    try {
      ticket = await fetchWSTicket()
    } catch (err) {
      if (import.meta.env.DEV) {
        console.error('[WebSocket] Failed to fetch ticket:', err)
      }
      scheduleReconnect()
      return
    }

    const url = buildUrl(ticket)

    try {
      ws = new WebSocket(url)
    } catch (err) {
      if (import.meta.env.DEV) {
        console.error('[WebSocket] 创建连接失败:', err)
      }
      scheduleReconnect()
      return
    }

    ws.onopen = () => {
      connected.value = true
      reconnecting.value = false
      retryCount.value = 0
      onOpen?.()
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        onMessage?.(data)
      } catch {
        // 非 JSON 消息，直接传递原始数据
        onMessage?.(event.data)
      }
    }

    ws.onclose = (event) => {
      connected.value = false
      onClose?.(event)

      if (!manuallyClosed) {
        scheduleReconnect()
      }
    }

    ws.onerror = (event) => {
      onError?.(event)
    }
  }

  function scheduleReconnect() {
    if (retryCount.value >= maxRetries) {
      reconnecting.value = false
      return
    }

    reconnecting.value = true
    // 指数退避：interval = min(initialInterval * 2^retryCount, maxInterval)
    const interval = Math.min(
      initialRetryInterval * Math.pow(2, retryCount.value),
      maxRetryInterval
    )
    retryCount.value++

    retryTimer = setTimeout(() => {
      connect()
    }, interval)
  }

  function send(data: any) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      const payload = typeof data === 'string' ? data : JSON.stringify(data)
      ws.send(payload)
    } else {
      if (import.meta.env.DEV) {
        console.warn('[WebSocket] 未连接，无法发送消息')
      }
    }
  }

  function disconnect() {
    manuallyClosed = true
    reconnecting.value = false
    retryCount.value = 0

    if (retryTimer) {
      clearTimeout(retryTimer)
      retryTimer = null
    }

    if (ws) {
      ws.onopen = null
      ws.onmessage = null
      ws.onclose = null
      ws.onerror = null
      ws.close()
      ws = null
    }

    connected.value = false
  }

  onUnmounted(() => {
    disconnect()
  })

  if (autoConnect) {
    connect()
  }

  return {
    connected,
    reconnecting,
    retryCount,
    connect,
    send,
    disconnect,
  }
}
