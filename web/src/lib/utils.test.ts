import { describe, it, expect } from 'vitest'
import { cn, formatDate, formatRelativeTime } from '@/lib/utils'

describe('cn', () => {
  it('merges class names', () => {
    expect(cn('foo', 'bar')).toBe('foo bar')
  })

  it('handles conditional classes', () => {
    expect(cn('base', false && 'hidden', 'visible')).toBe('base visible')
  })

  it('merges tailwind conflicts (later wins)', () => {
    expect(cn('px-2', 'px-4')).toBe('px-4')
  })

  it('handles empty input', () => {
    expect(cn()).toBe('')
  })
})

describe('formatDate', () => {
  it('formats a Date object', () => {
    const date = new Date('2024-01-15T10:30:00')
    const result = formatDate(date)
    expect(result).toContain('2024')
    expect(result).toContain('01')
    expect(result).toContain('15')
  })

  it('formats a date string', () => {
    const result = formatDate('2024-06-01T08:00:00')
    expect(result).toContain('2024')
    expect(result).toContain('06')
  })
})

describe('formatRelativeTime', () => {
  it('returns "刚刚" for recent times', () => {
    const now = new Date()
    expect(formatRelativeTime(now)).toBe('刚刚')
  })

  it('returns minutes ago', () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000)
    expect(formatRelativeTime(fiveMinutesAgo)).toContain('5')
    expect(formatRelativeTime(fiveMinutesAgo)).toContain('分钟前')
  })

  it('returns hours ago', () => {
    const threeHoursAgo = new Date(Date.now() - 3 * 60 * 60 * 1000)
    expect(formatRelativeTime(threeHoursAgo)).toContain('3')
    expect(formatRelativeTime(threeHoursAgo)).toContain('小时前')
  })

  it('returns days ago', () => {
    const tenDaysAgo = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000)
    expect(formatRelativeTime(tenDaysAgo)).toContain('10')
    expect(formatRelativeTime(tenDaysAgo)).toContain('天前')
  })

  it('falls back to formatDate for old dates', () => {
    const oldDate = new Date(Date.now() - 60 * 24 * 60 * 60 * 1000)
    const result = formatRelativeTime(oldDate)
    // Should contain year (from formatDate)
    expect(result).toMatch(/\d{4}/)
  })
})
