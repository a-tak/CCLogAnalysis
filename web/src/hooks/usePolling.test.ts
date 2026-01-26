import { renderHook } from '@testing-library/react'
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { usePolling } from './usePolling'

describe('usePolling', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.restoreAllMocks()
    vi.useRealTimers()
  })

  test('初回マウント時に即座にcallbackを実行する', () => {
    const callback = vi.fn()

    renderHook(() => usePolling(callback, 1000))

    expect(callback).toHaveBeenCalledTimes(1)
  })

  test('指定間隔でcallbackを繰り返し実行する', () => {
    const callback = vi.fn()

    renderHook(() => usePolling(callback, 1000))

    expect(callback).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(2)

    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(3)

    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(4)
  })

  test('アンマウント時にタイマーをクリアする', () => {
    const callback = vi.fn()

    const { unmount } = renderHook(() => usePolling(callback, 1000))

    expect(callback).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(2)

    unmount()

    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(2) // アンマウント後は増えない
  })

  test('enabled=falseの場合、ポーリングしない', () => {
    const callback = vi.fn()

    renderHook(() => usePolling(callback, 1000, false))

    expect(callback).not.toHaveBeenCalled()

    vi.advanceTimersByTime(1000)
    expect(callback).not.toHaveBeenCalled()
  })

  test('enabledがfalse→trueに変化したら、ポーリング開始', () => {
    const callback = vi.fn()

    const { rerender } = renderHook(
      ({ enabled }) => usePolling(callback, 1000, enabled),
      { initialProps: { enabled: false } }
    )

    expect(callback).not.toHaveBeenCalled()

    rerender({ enabled: true })
    expect(callback).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(2)
  })

  test('依存配列が変化したら、タイマーを再セットアップ', () => {
    const callback1 = vi.fn()
    const callback2 = vi.fn()

    const { rerender } = renderHook(
      ({ cb }) => usePolling(cb, 1000),
      { initialProps: { cb: callback1 } }
    )

    expect(callback1).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1000)
    expect(callback1).toHaveBeenCalledTimes(2)

    // コールバックを変更
    rerender({ cb: callback2 })
    expect(callback2).toHaveBeenCalledTimes(1)
    expect(callback1).toHaveBeenCalledTimes(2) // 増えない

    vi.advanceTimersByTime(1000)
    expect(callback2).toHaveBeenCalledTimes(2)
    expect(callback1).toHaveBeenCalledTimes(2) // 増えない
  })

  test('callbackがエラーを投げても、次回ポーリングは継続', async () => {
    let callCount = 0
    const callback = vi.fn(() => {
      callCount++
      if (callCount === 2) {
        throw new Error('Test error')
      }
    })

    renderHook(() => usePolling(callback, 1000))

    expect(callback).toHaveBeenCalledTimes(1)

    // 2回目でエラー
    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(2)

    // エラー後も続行
    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(3)
  })

  test('非同期callbackも正しく動作する', async () => {
    const callback = vi.fn(async () => {
      await new Promise(resolve => setTimeout(resolve, 100))
    })

    renderHook(() => usePolling(callback, 1000))

    expect(callback).toHaveBeenCalledTimes(1)

    vi.advanceTimersByTime(1000)
    expect(callback).toHaveBeenCalledTimes(2)
  })
})
