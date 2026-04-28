import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useThemeStore } from '@/stores/theme'

describe('useThemeStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    document.documentElement.classList.remove('dark')
  })

  it('defaults to dark theme', () => {
    const store = useThemeStore()
    expect(store.isDark).toBe(true)
  })

  it('toggles theme', () => {
    const store = useThemeStore()
    expect(store.isDark).toBe(true)
    store.toggleTheme()
    expect(store.isDark).toBe(false)
    store.toggleTheme()
    expect(store.isDark).toBe(true)
  })

  it('updates document class on toggle', () => {
    const store = useThemeStore()
    // Initial state: dark class should be present (default isDark=true)
    store.toggleTheme() // isDark = false
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    store.toggleTheme() // isDark = true
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })
})
