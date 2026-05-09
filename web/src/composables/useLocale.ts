import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

export function useLocale() {
  const { locale, availableLocales } = useI18n()

  const currentLocale = computed({
    get: () => locale.value,
    set: (val: string) => {
      locale.value = val
      localStorage.setItem('locale', val)
      document.documentElement.lang = val
    }
  })

  const toggleLocale = () => {
    currentLocale.value = currentLocale.value === 'zh' ? 'en' : 'zh'
  }

  const localeName = computed(() =>
    currentLocale.value === 'zh' ? '中文' : 'EN'
  )

  return { currentLocale, toggleLocale, localeName, availableLocales }
}
