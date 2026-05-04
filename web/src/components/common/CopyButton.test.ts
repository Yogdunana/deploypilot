import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import CopyButton from '@/components/common/CopyButton.vue'

// Mock lucide-vue-next 图标
vi.mock('lucide-vue-next', () => ({
  Check: { template: '<svg data-testid="check-icon" />' },
  Copy: { template: '<svg data-testid="copy-icon" />' },
}))

describe('CopyButton', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('渲染复制按钮，默认显示 Copy 图标', () => {
    const wrapper = mount(CopyButton, {
      props: { text: 'hello' },
    })
    expect(wrapper.find('[data-testid="copy-icon"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="check-icon"]').exists()).toBe(false)
  })

  it('点击后调用 clipboard API 并显示 Check 图标', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })

    const wrapper = mount(CopyButton, {
      props: { text: 'copy-me' },
    })

    await wrapper.find('button').trigger('click')

    expect(writeText).toHaveBeenCalledWith('copy-me')
    expect(wrapper.find('[data-testid="check-icon"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="copy-icon"]').exists()).toBe(false)
  })

  it('2 秒后恢复显示 Copy 图标', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })

    const wrapper = mount(CopyButton, {
      props: { text: 'copy-me' },
    })

    await wrapper.find('button').trigger('click')
    expect(wrapper.find('[data-testid="check-icon"]').exists()).toBe(true)

    // 快进 2 秒
    vi.advanceTimersByTime(2000)
    await wrapper.vm.$nextTick()

    expect(wrapper.find('[data-testid="copy-icon"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="check-icon"]').exists()).toBe(false)
  })

  it('text 为空时不调用 clipboard API', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })

    const wrapper = mount(CopyButton, {
      props: { text: '' },
    })

    await wrapper.find('button').trigger('click')
    expect(writeText).not.toHaveBeenCalled()
  })

  it('不传 text 时不调用 clipboard API', async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    Object.assign(navigator, { clipboard: { writeText } })

    const wrapper = mount(CopyButton)

    await wrapper.find('button').trigger('click')
    expect(writeText).not.toHaveBeenCalled()
  })

  it('clipboard API 失败时使用 fallback 方式复制', async () => {
    const writeText = vi.fn().mockRejectedValue(new Error('clipboard denied'))
    Object.assign(navigator, { clipboard: { writeText } })

    // mock document.execCommand
    const execCommandSpy = vi.fn()
    document.execCommand = execCommandSpy

    const wrapper = mount(CopyButton, {
      props: { text: 'fallback-copy' },
    })

    await wrapper.find('button').trigger('click')

    expect(execCommandSpy).toHaveBeenCalledWith('copy')
    expect(wrapper.find('[data-testid="check-icon"]').exists()).toBe(true)
  })

  it('按钮有正确的 title 属性', () => {
    const wrapper = mount(CopyButton, {
      props: { text: 'test' },
    })
    expect(wrapper.find('button').attributes('title')).toBe('复制')
  })

  it('支持传入自定义 class', () => {
    const wrapper = mount(CopyButton, {
      props: { text: 'test', class: 'custom-class' },
    })
    expect(wrapper.find('button').classes()).toContain('custom-class')
  })
})
