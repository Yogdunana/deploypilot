import { describe, it, expect } from 'vitest'
import { cn, formatDate, formatRelativeTime } from '@/lib/utils'

describe('cn', () => {
  it('合并多个类名', () => {
    const result = cn('foo', 'bar')
    expect(result).toBe('foo bar')
  })

  it('处理条件类名', () => {
    const result = cn('base', false && 'hidden', 'visible')
    expect(result).toBe('base visible')
  })

  it('合并 Tailwind 冲突类名（后者优先）', () => {
    const result = cn('px-2', 'px-4')
    expect(result).toBe('px-4')
  })

  it('处理空输入', () => {
    const result = cn()
    expect(result).toBe('')
  })

  it('处理 undefined 和 null', () => {
    const result = cn('base', undefined, null, 'extra')
    expect(result).toBe('base extra')
  })
})

describe('formatDate', () => {
  it('格式化有效的日期字符串', () => {
    const result = formatDate('2026-05-04T12:00:00Z')
    expect(result).toBeTruthy()
    expect(typeof result).toBe('string')
  })

  it('格式化 Date 对象', () => {
    const result = formatDate(new Date('2026-05-04T12:00:00Z'))
    expect(result).toBeTruthy()
    expect(typeof result).toBe('string')
  })

  it('处理空字符串', () => {
    const result = formatDate('')
    expect(result).toBe('Invalid Date')
  })

  it('处理无效日期', () => {
    const result = formatDate('not-a-date')
    expect(result).toBe('Invalid Date')
  })

  it('格式化结果包含年份', () => {
    const result = formatDate('2026-01-01T00:00:00Z')
    expect(result).toContain('2026')
  })
})

describe('formatRelativeTime', () => {
  it('返回"刚刚"表示当前时间', () => {
    const now = new Date()
    const result = formatRelativeTime(now.toISOString())
    expect(result).toBe('刚刚')
  })

  it('返回"X 分钟前"表示几分钟前', () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000)
    const result = formatRelativeTime(fiveMinutesAgo.toISOString())
    expect(result).toBe('5 分钟前')
  })

  it('返回"X 小时前"表示几小时前', () => {
    const threeHoursAgo = new Date(Date.now() - 3 * 60 * 60 * 1000)
    const result = formatRelativeTime(threeHoursAgo.toISOString())
    expect(result).toBe('3 小时前')
  })

  it('返回"X 天前"表示几天前', () => {
    const tenDaysAgo = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000)
    const result = formatRelativeTime(tenDaysAgo.toISOString())
    expect(result).toBe('10 天前')
  })

  it('超过 30 天时返回格式化日期', () => {
    const sixtyDaysAgo = new Date(Date.now() - 60 * 24 * 60 * 60 * 1000)
    const result = formatRelativeTime(sixtyDaysAgo.toISOString())
    // 超过 30 天应返回 formatDate 的结果
    expect(result).toBeTruthy()
    expect(result).not.toContain('天前')
  })

  it('处理空字符串', () => {
    const result = formatRelativeTime('')
    expect(result).toBe('Invalid Date')
  })

  it('处理 Date 对象', () => {
    const now = new Date()
    const result = formatRelativeTime(now)
    expect(result).toBe('刚刚')
  })

  it('返回字符串类型', () => {
    const result = formatRelativeTime(new Date().toISOString())
    expect(typeof result).toBe('string')
  })
})
