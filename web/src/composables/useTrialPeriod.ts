import { ref, onMounted, type Ref } from 'vue'
import { getTrialStatus } from '@/api/modules/trial'

interface TrialInfo {
  status: string
  days_remaining: number
  is_active: boolean
  is_expired: boolean
  is_converted: boolean
  expires_at: string
}

const trialInfo: Ref<TrialInfo | null> = ref(null)
const loaded = ref(false)

export function useTrialPeriod() {
  async function loadTrialStatus() {
    try {
      const res = await getTrialStatus()
      if (res.data.status === 'success') {
        trialInfo.value = res.data.data as TrialInfo
        loaded.value = true
      }
    } catch {
      // On error, hide trial banner (fail-open)
      loaded.value = true
      trialInfo.value = null
    }
  }

  function showBanner(): boolean {
    if (!loaded.value || !trialInfo.value) return false
    return trialInfo.value.is_active || trialInfo.value.is_expired
  }

  function isUrgent(): boolean {
    if (!trialInfo.value?.is_active) return false
    return trialInfo.value.days_remaining <= 7
  }

  return {
    trialInfo,
    loaded,
    loadTrialStatus,
    showBanner,
    isUrgent,
  }
}
