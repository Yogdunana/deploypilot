import { ref, onUnmounted, onMounted } from 'vue'

export interface UsePollingOptions<T> {
  /** 数据获取函数 */
  fetchFn: () => Promise<T>
  /** 轮询间隔（毫秒），默认 5000 */
  interval?: number
  /** 是否自动开始，默认 false */
  autoStart?: boolean
  /** 页面不可见时是否暂停，默认 true */
  pauseWhenHidden?: boolean
}

export function usePolling<T>(options: UsePollingOptions<T>) {
  const {
    fetchFn,
    interval = 5000,
    autoStart = false,
    pauseWhenHidden = true,
  } = options

  const data = ref<T | null>(null) as { value: T | null }
  const loading = ref(false)
  const error = ref<Error | null>(null)

  let timer: ReturnType<typeof setInterval> | null = null
  let isRunning = false
  let visibilityHandler: (() => void) | null = null

  async function refresh() {
    loading.value = true
    error.value = null
    try {
      data.value = await fetchFn()
    } catch (err: any) {
      error.value = err instanceof Error ? err : new Error(err?.message || '请求失败')
    } finally {
      loading.value = false
    }
  }

  function start() {
    if (isRunning) return
    isRunning = true

    // 立即执行一次
    refresh()

    // 设置定时器
    timer = setInterval(() => {
      if (pauseWhenHidden && document.hidden) return
      refresh()
    }, interval)

    // 监听页面可见性
    if (pauseWhenHidden && !visibilityHandler) {
      visibilityHandler = () => {
        if (!document.hidden && isRunning) {
          refresh()
        }
      }
      document.addEventListener('visibilitychange', visibilityHandler)
    }
  }

  function stop() {
    isRunning = false
    if (timer) {
      clearInterval(timer)
      timer = null
    }
    if (visibilityHandler) {
      document.removeEventListener('visibilitychange', visibilityHandler)
      visibilityHandler = null
    }
  }

  onUnmounted(() => {
    stop()
  })

  if (autoStart) {
    onMounted(() => {
      start()
    })
  }

  return {
    data,
    loading,
    error,
    start,
    stop,
    refresh,
  }
}
