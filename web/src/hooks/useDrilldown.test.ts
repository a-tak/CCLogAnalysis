import { renderHook, act } from '@testing-library/react'
import { describe, test, expect, vi } from 'vitest'
import { useDrilldown } from './useDrilldown'

describe('useDrilldown', () => {
  test('初期状態は全て null/false', () => {
    const fetchData = vi.fn()

    const { result } = renderHook(() => useDrilldown({ fetchData }))

    expect(result.current.selectedDate).toBeNull()
    expect(result.current.data).toBeNull()
    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  test('日付をクリックするとデータを取得する', async () => {
    const mockData = { sessions: [{ id: '1' }] }
    const fetchData = vi.fn().mockResolvedValue(mockData)

    const { result } = renderHook(() => useDrilldown({ fetchData }))

    // 日付をクリック
    act(() => {
      result.current.handleDateClick('2026-01-29')
    })

    // selectedDate が更新される
    expect(result.current.selectedDate).toBe('2026-01-29')

    // ローディング状態になる
    expect(result.current.loading).toBe(true)

    // fetchData が呼ばれる
    expect(fetchData).toHaveBeenCalledWith('2026-01-29')

    // データが取得されるまで待つ
    await new Promise(resolve => setTimeout(resolve, 50))

    expect(result.current.data).toEqual(mockData)
    expect(result.current.loading).toBe(false)
    expect(result.current.error).toBeNull()
  })

  test('同じ日付を2回クリックするとドリルダウンパネルが閉じる', async () => {
    const mockData = { sessions: [{ id: '1' }] }
    const fetchData = vi.fn().mockResolvedValue(mockData)

    const { result } = renderHook(() => useDrilldown({ fetchData }))

    // 最初のクリック
    act(() => {
      result.current.handleDateClick('2026-01-29')
    })

    expect(result.current.selectedDate).toBe('2026-01-29')

    // データが取得されるまで待つ
    await new Promise(resolve => setTimeout(resolve, 50))

    expect(result.current.data).toEqual(mockData)

    // 2回目のクリック（同じ日付）
    act(() => {
      result.current.handleDateClick('2026-01-29')
    })

    expect(result.current.selectedDate).toBeNull()
    expect(result.current.data).toBeNull()
  })

  test('異なる日付をクリックすると新しいデータを取得する', async () => {
    const mockData1 = { sessions: [{ id: '1' }] }
    const mockData2 = { sessions: [{ id: '2' }] }
    const fetchData = vi.fn()
      .mockResolvedValueOnce(mockData1)
      .mockResolvedValueOnce(mockData2)

    const { result } = renderHook(() => useDrilldown({ fetchData }))

    // 最初の日付をクリック
    act(() => {
      result.current.handleDateClick('2026-01-29')
    })

    await new Promise(resolve => setTimeout(resolve, 50))

    expect(result.current.data).toEqual(mockData1)
    expect(fetchData).toHaveBeenCalledWith('2026-01-29')

    // 異なる日付をクリック
    act(() => {
      result.current.handleDateClick('2026-01-28')
    })

    await new Promise(resolve => setTimeout(resolve, 50))

    expect(result.current.selectedDate).toBe('2026-01-28')
    expect(result.current.data).toEqual(mockData2)
    expect(fetchData).toHaveBeenCalledWith('2026-01-28')
    expect(fetchData).toHaveBeenCalledTimes(2)
  })

  test('データ取得に失敗するとエラーメッセージが表示される', async () => {
    const fetchData = vi.fn().mockRejectedValue(new Error('Failed to fetch data'))

    const { result } = renderHook(() => useDrilldown({ fetchData }))

    act(() => {
      result.current.handleDateClick('2026-01-29')
    })

    await new Promise(resolve => setTimeout(resolve, 50))

    expect(result.current.error).toBe('データの取得に失敗しました')
    expect(result.current.data).toBeNull()
    expect(result.current.loading).toBe(false)
  })

  test('API エラーの場合はエラーメッセージが表示される', async () => {
    // ApiError は lib/api/client から直接インポートできないため、
    // instanceof チェック自体をテストするのではなく、
    // エラーメッセージを含むエラーオブジェクトをテストする
    const mockError = new Error('API Error: Not Found')
    mockError.name = 'ApiError'

    const fetchData = vi.fn().mockRejectedValue(mockError)

    const { result } = renderHook(() => useDrilldown({ fetchData }))

    act(() => {
      result.current.handleDateClick('2026-01-29')
    })

    await new Promise(resolve => setTimeout(resolve, 50))

    // ApiError ではないため、一般的なエラーメッセージが使われる
    expect(result.current.error).toBe('データの取得に失敗しました')
    expect(result.current.data).toBeNull()
  })

  test('無効な日付フォーマットでエラーを返す', () => {
    const fetchData = vi.fn()

    const { result } = renderHook(() => useDrilldown({ fetchData }))

    act(() => {
      result.current.handleDateClick('invalid-date')
    })

    expect(result.current.error).toBe('無効な日付フォーマットです')
    expect(result.current.data).toBeNull()
    expect(fetchData).not.toHaveBeenCalled()
  })

  test('close() を呼ぶとドリルダウンパネルが閉じる', async () => {
    const mockData = { sessions: [{ id: '1' }] }
    const fetchData = vi.fn().mockResolvedValue(mockData)

    const { result } = renderHook(() => useDrilldown({ fetchData }))

    act(() => {
      result.current.handleDateClick('2026-01-29')
    })

    await new Promise(resolve => setTimeout(resolve, 50))

    expect(result.current.selectedDate).toBe('2026-01-29')
    expect(result.current.data).toEqual(mockData)

    // close() を呼ぶ
    act(() => {
      result.current.close()
    })

    expect(result.current.selectedDate).toBeNull()
    expect(result.current.data).toBeNull()
  })

  test('selectedDate が null の場合、fetchData は呼ばれない', () => {
    const fetchData = vi.fn()

    renderHook(() => useDrilldown({ fetchData }))

    expect(fetchData).not.toHaveBeenCalled()
  })

  test('fetchData が新しい関数に変わると、再度フェッチする', async () => {
    const mockData1 = { sessions: [{ id: '1' }] }
    const mockData2 = { sessions: [{ id: '2' }] }

    const fetchData1 = vi.fn().mockResolvedValue(mockData1)
    const fetchData = fetchData1

    const { result, rerender } = renderHook(
      ({ fetchData: fd }) => useDrilldown({ fetchData: fd }),
      { initialProps: { fetchData } }
    )

    // 日付をクリック
    act(() => {
      result.current.handleDateClick('2026-01-29')
    })

    await new Promise(resolve => setTimeout(resolve, 50))

    expect(result.current.data).toEqual(mockData1)
    expect(fetchData1).toHaveBeenCalledTimes(1)

    // fetchData を新しい関数に変更
    const fetchData2 = vi.fn().mockResolvedValue(mockData2)
    rerender({ fetchData: fetchData2 })

    await new Promise(resolve => setTimeout(resolve, 50))

    // 新しい fetchData が呼ばれる
    expect(fetchData2).toHaveBeenCalledWith('2026-01-29')
  })
})
