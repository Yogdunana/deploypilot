import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as uptimeApi from '@/api/modules/uptime'
import * as monitorApi from '@/api/modules/monitor'
import * as queryApi from '@/api/modules/monitor_query'
import type { UptimeMonitor, HeartbeatMonitor } from '@/types/monitor'

export const useMonitorStore = defineStore('monitor', () => {
  const monitors = ref<UptimeMonitor[]>([])
  const heartbeats = ref<HeartbeatMonitor[]>([])
  const connected = ref(false)
  const loading = ref(false)
  const systemMetrics = ref<any>(null)
  const alertStats = ref<any>(null)

  const overallStatus = computed(() => {
    if (monitors.value.length === 0) return 'loading'
    return monitors.value.every(m => m.status === 'up') ? 'operational' : 'degraded'
  })

  const downCount = computed(() => monitors.value.filter(m => m.status === 'down').length)
  const upCount = computed(() => monitors.value.filter(m => m.status === 'up').length)
  const avgUptime = computed(() => {
    if (monitors.value.length === 0) return 0
    return monitors.value.reduce((sum, m) => sum + m.uptime, 0) / monitors.value.length
  })

  async function fetchMonitors() {
    loading.value = true
    try {
      const res = await uptimeApi.listMonitors()
      if (res.data.status === 'success') {
        monitors.value = res.data.data
      }
    } catch { /* silent */ } finally {
      loading.value = false
    }
  }

  async function fetchHeartbeats() {
    try {
      const res = await uptimeApi.listHeartbeats()
      if (res.data.status === 'success') {
        heartbeats.value = res.data.data
      }
    } catch { /* silent */ }
  }

  async function fetchSystemMetrics() {
    try {
      const res = await monitorApi.getSystemMetrics()
      if (res.data.status === 'success') {
        systemMetrics.value = res.data.data
      }
    } catch { /* silent */ }
  }

  async function fetchAlertStats(period?: string) {
    try {
      const res = await queryApi.getAlertStats(period)
      if (res.data.status === 'success') {
        alertStats.value = res.data.data
      }
    } catch { /* silent */ }
  }

  return {
    monitors, heartbeats, connected, loading, systemMetrics, alertStats,
    overallStatus, downCount, upCount, avgUptime,
    fetchMonitors, fetchHeartbeats, fetchSystemMetrics, fetchAlertStats,
  }
})
