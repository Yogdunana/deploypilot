import { describe, it, expect } from 'vitest'
import { formatDate, formatRelativeTime } from '@/lib/utils'

describe('formatDate', () => {
  it('formats a valid date string', () => {
    const result = formatDate('2026-05-04T12:00:00Z')
    expect(result).toBeTruthy()
    expect(typeof result).toBe('string')
  })

  it('handles empty string', () => {
    const result = formatDate('')
    expect(result).toBe('Invalid Date')
  })

  it('handles invalid date', () => {
    const result = formatDate('not-a-date')
    expect(result).toBe('Invalid Date')
  })
})

describe('formatRelativeTime', () => {
  it('returns a string for recent dates', () => {
    const now = new Date()
    const result = formatRelativeTime(now.toISOString())
    expect(result).toBeTruthy()
    expect(typeof result).toBe('string')
  })

  it('returns a string for old dates', () => {
    const old = new Date('2020-01-01T00:00:00Z')
    const result = formatRelativeTime(old.toISOString())
    expect(result).toBeTruthy()
  })

  it('handles empty string', () => {
    const result = formatRelativeTime('')
    expect(result).toBe('Invalid Date')
  })
})
