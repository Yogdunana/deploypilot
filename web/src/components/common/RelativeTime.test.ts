import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import RelativeTime from '@/components/common/RelativeTime.vue'

describe('RelativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('渲染"刚刚"表示当前时间', async () => {
    const now = new Date()
    const wrapper = mount(RelativeTime, {
      props: { date: now.toISOString() },
    })
    await flushPromises()
    expect(wrapper.text()).toBe('刚刚')
  })

  it('渲染"X 分钟前"表示几分钟前的时间', async () => {
    const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000)
    const wrapper = mount(RelativeTime, {
      props: { date: fiveMinutesAgo.toISOString() },
    })
    await flushPromises()
    expect(wrapper.text()).toBe('5 分钟前')
  })

  it('渲染"X 小时前"表示几小时前的时间', async () => {
    const twoHoursAgo = new Date(Date.now() - 2 * 60 * 60 * 1000)
    const wrapper = mount(RelativeTime, {
      props: { date: twoHoursAgo.toISOString() },
    })
    await flushPromises()
    expect(wrapper.text()).toBe('2 小时前')
  })

  it('渲染"X 天前"表示几天前的时间', async () => {
    const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000)
    const wrapper = mount(RelativeTime, {
      props: { date: threeDaysAgo.toISOString() },
    })
    await flushPromises()
    expect(wrapper.text()).toBe('3 天前')
  })

  it('不传 date 时不渲染文本', async () => {
    const wrapper = mount(RelativeTime)
    await flushPromises()
    expect(wrapper.text()).toBe('')
  })

  it('date 为 undefined 时不渲染文本', async () => {
    const wrapper = mount(RelativeTime, {
      props: { date: undefined },
    })
    await flushPromises()
    expect(wrapper.text()).toBe('')
  })

  it('设置正确的 datetime 属性', async () => {
    const date = '2026-05-04T12:00:00Z'
    const wrapper = mount(RelativeTime, {
      props: { date },
    })
    await flushPromises()
    expect(wrapper.find('time').attributes('datetime')).toBe('2026-05-04T12:00:00.000Z')
  })

  it('不传 date 时 datetime 属性为 undefined', async () => {
    const wrapper = mount(RelativeTime)
    await flushPromises()
    expect(wrapper.find('time').attributes('datetime')).toBeUndefined()
  })

  it('每 60 秒自动更新显示', async () => {
    const now = new Date()
    const wrapper = mount(RelativeTime, {
      props: { date: now.toISOString() },
    })
    await flushPromises()
    expect(wrapper.text()).toBe('刚刚')

    // 快进 60 秒
    vi.advanceTimersByTime(60000)
    await flushPromises()

    // 60 秒后应显示 "1 分钟前"
    expect(wrapper.text()).toBe('1 分钟前')
  })

  it('组件卸载时清除定时器', async () => {
    const clearIntervalSpy = vi.spyOn(globalThis, 'clearInterval')
    const wrapper = mount(RelativeTime, {
      props: { date: new Date().toISOString() },
    })
    await flushPromises()

    wrapper.unmount()
    expect(clearIntervalSpy).toHaveBeenCalled()
    clearIntervalSpy.mockRestore()
  })

  it('支持传入自定义 class', async () => {
    const wrapper = mount(RelativeTime, {
      props: { date: new Date().toISOString(), class: 'custom-time' },
    })
    await flushPromises()
    expect(wrapper.find('time').classes()).toContain('custom-time')
  })
})
