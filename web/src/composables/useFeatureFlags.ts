import { ref, type Ref } from 'vue'
import { getFeatureFlags, getFeatureFlagsForTenant } from '@/api/modules/featureFlags'

interface FeatureFlagInfo {
  key: string
  name: string
  enabled: boolean
  overridden?: boolean
  override_enabled?: boolean
  override_reason?: string
}

const featureFlags: Ref<Record<string, boolean>> = ref({})
const loaded = ref(false)

export function useFeatureFlags() {
  async function loadFlags(tenantId?: string) {
    try {
      const res = tenantId
        ? await getFeatureFlagsForTenant(tenantId)
        : await getFeatureFlags()
      if (res.data.status === 'success') {
        const flags = res.data.data.flags as FeatureFlagInfo[]
        const map: Record<string, boolean> = {}
        for (const f of flags) {
          map[f.key] = f.enabled
        }
        featureFlags.value = map
        loaded.value = true
      }
    } catch {
      // On error, enable all features (fail-open)
      loaded.value = true
    }
  }

  function isEnabled(key: string): boolean {
    if (!loaded.value) return true // fail-open before load
    return featureFlags.value[key] !== false
  }

  function isDisabled(key: string): boolean {
    return !isEnabled(key)
  }

  return {
    featureFlags,
    loaded,
    loadFlags,
    isEnabled,
    isDisabled,
  }
}
