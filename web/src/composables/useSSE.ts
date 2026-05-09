import { ref, onUnmounted } from 'vue'

export interface UseSSEOptions {
  /** SSE 端点路径，如 /api/v1/sse/deploy/1 */
  url: string
  /** 收到事件时的回调 */
  onEvent?: (event: string, data: any) => void
  /** 连接成功时的回调 */
  onOpen?: () => void
  /** 连接错误时的回调 */
  onError?: (error: Error) => void
  /** 连接关闭时的回调 */
  onClose?: () => void
  /** 是否自动连接，默认 true */
  autoConnect?: boolean
}

export type SSEStatus = 'connecting' | 'connected' | 'closed' | 'error'

export function useSSE(options: UseSSEOptions) {
  const {
    url,
    onEvent,
    onOpen,
    onError,
    onClose,
    autoConnect = true,
  } = options

  const status = ref<SSEStatus>('connecting')
  let abortController: AbortController | null = null

  function getToken(): string {
    return localStorage.getItem('deploypilot_token') || ''
  }

  function buildFullUrl(): string {
    // 确保 URL 是完整的
    if (url.startsWith('http://') || url.startsWith('https://')) {
      return url
    }
    const base = window.location.origin
    return `${base}${url}`
  }

  async function connect() {
    if (abortController) {
      return
    }

    abortController = new AbortController()
    status.value = 'connecting'

    const fullUrl = buildFullUrl()
    const token = getToken()

    try {
      const response = await fetch(fullUrl, {
        method: 'GET',
        headers: {
          'Accept': 'text/event-stream',
          'Authorization': `Bearer ${token}`,
          'Cache-Control': 'no-cache',
        },
        signal: abortController.signal,
      })

      if (!response.ok) {
        throw new Error(`SSE 连接失败: ${response.status} ${response.statusText}`)
      }

      status.value = 'connected'
      onOpen?.()

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('无法获取响应流')
      }

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })

        // 解析 SSE 格式：event: xxx\ndata: xxx\n\n
        const lines = buffer.split('\n')
        buffer = ''

        let currentEvent = ''
        let currentData = ''

        for (const line of lines) {
          if (line.startsWith('event:')) {
            currentEvent = line.slice(6).trim()
          } else if (line.startsWith('data:')) {
            currentData = line.slice(5).trim()
          } else if (line === '' && currentData) {
            // 空行表示事件结束
            try {
              const parsed = JSON.parse(currentData)
              onEvent?.(currentEvent || 'message', parsed)

              // step=done 时自动关闭
              if (parsed.step === 'done') {
                close()
                return
              }
            } catch {
              onEvent?.(currentEvent || 'message', currentData)
            }
            currentEvent = ''
            currentData = ''
          } else if (line !== '') {
            // 未完成的行，放回 buffer
            buffer = line + '\n'
          }
        }
      }

      // 流正常结束
      status.value = 'closed'
      onClose?.()
    } catch (err: any) {
      if (err.name === 'AbortError') {
        status.value = 'closed'
        onClose?.()
      } else {
        status.value = 'error'
        onError?.(err)
      }
    } finally {
      abortController = null
    }
  }

  function close() {
    if (abortController) {
      abortController.abort()
      abortController = null
    }
    status.value = 'closed'
  }

  onUnmounted(() => {
    close()
  })

  if (autoConnect) {
    connect()
  }

  return {
    status,
    connect,
    close,
  }
}
