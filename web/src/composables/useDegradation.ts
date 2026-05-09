import { ref, type Ref } from 'vue'
import { getDegradationStatus } from '@/api/modules/degradation'

interface DegradationInfo {
  level: 'none' | 'partial' | 'readonly'
  license_status: string
  trial_status: string
  tier: string
  gated_features: string[]
  read_only_reason: string
  expires_at: string
  grace_days_left: number
}

const degradationInfo: Ref<DegradationInfo | null> = ref(null)
const loaded = ref(false)

export function useDegradation() {
  async function loadStatus() {
    try {
      const res = await getDegradationStatus()
      if (res.data.status === 'success') {
        degradationInfo.value = res.data.data as DegradationInfo
        loaded.value = true
      }
    } catch {
      loaded.value = true
      degradationInfo.value = null
    }
  }

  function isReadOnly(): boolean {
    return degradationInfo.value?.level === 'readonly'
  }

  function isPartial(): boolean {
    return degradationInfo.value?.level === 'partial'
  }

  function isFeatureGated(feature: string): boolean {
    if (!degradationInfo.value) return false
    return degradationInfo.value.gated_features.includes(feature)
  }

  function getBannerMessage(): string {
    if (!degradationInfo.value) return ''
    if (degradationInfo.value.level === 'readonly') {
      return degradationInfo.value.read_only_reason || 'Instance is in read-only mode.'
    }
    if (degradationInfo.value.level === 'partial' && degradationInfo.value.read_only_reason) {
      return degradationInfo.value.read_only_reason
    }
    return ''
  }

  return {
    degradationInfo,
    loaded,
    loadStatus,
    isReadOnly,
    isPartial,
    isFeatureGated,
    getBannerMessage,
  }
}
