import { inject } from 'vue'

type ToastVariant = 'default' | 'success' | 'destructive' | 'error' | 'warning' | 'info'

/**
 * Safely get the toast function from the Toast component's provide.
 * Returns a no-op function if toast is not available (e.g., during testing
 * or if Toast component hasn't mounted yet).
 */
export function useToast() {
  const ctx = inject<{ toast: (message: string, variant?: ToastVariant) => void }>('toast')
  return {
    toast: ctx?.toast ?? ((message: string) => {
      console.warn('[useToast] Toast not available:', message)
    }),
  }
}
