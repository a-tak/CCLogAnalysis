import { renderHook, waitFor } from '@testing-library/react'
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { useProjectDetailPolling } from './useProjectDetailPolling'
import { api } from '@/lib/api/client'
import * as usePollingModule from './usePolling'

// usePolling をモック
vi.mock('./usePolling', () => ({
  usePolling: vi.fn(),
}))

vi.mock('@/lib/api/client', () => ({
  api: {
    getProjectStats: vi.fn(),
    getProjectTimeline: vi.fn(),
  },
  ApiError: class ApiError extends Error {
    status: number
    constructor(message: string, status: number) {
      super(message)
      this.status = status
    }
  },
}))

describe('useProjectDetailPolling', () => {
  const mockProjectStats = {
    totalSessions: 10,
    totalTokens: 50000,
    totalInputTokens: 20000,
    totalOutputTokens: 30000,
    errorRate: 0.1,
  }

  const mockTimelineResponse = {
    data: [
      {
        periodStart: '2026-01-26T00:00:00Z',
        totalInputTokens: 20000,
        totalOutputTokens: 30000,
        totalTokens: 50000,
      },
    ],
  }

  let capturedCallback: (() => Promise<void>) | null = null

  beforeEach(() => {
    capturedCallback = null

    // usePolling が呼ばれたときに callback をキャプチャのみ（初回実行はしない）
    vi.mocked(usePollingModule.usePolling).mockImplementation((callback: () => Promise<void>) => {
      capturedCallback = callback
    })

    vi.mocked(api.getProjectStats).mockResolvedValue(mockProjectStats)
    vi.mocked(api.getProjectTimeline).mockResolvedValue(mockTimelineResponse)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  test('初回ロード時に全データを取得する', async () => {
    const { result } = renderHook(() => useProjectDetailPolling('TestProject', 'day'))

    // 初期状態はローディング中
    expect(result.current.loading).toBe(true)
    expect(result.current.error).toBe(null)

    // キャプチャしたcallbackを実行（usePollingの初回実行をシミュレート）
    if (capturedCallback) {
      await capturedCallback()
    }

    // データ取得完了まで待つ
    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    // API呼び出しを確認
    expect(api.getProjectStats).toHaveBeenCalledWith('TestProject')
    expect(api.getProjectTimeline).toHaveBeenCalledWith('TestProject', 'day', 30)

    // データが正しく設定されている
    expect(result.current.stats).toEqual(mockProjectStats)
    expect(result.current.timeline).toEqual(mockTimelineResponse)
    expect(result.current.error).toBe(null)
  })

  test('usePollingに正しいパラメータを渡す', () => {
    renderHook(() => useProjectDetailPolling('TestProject', 'day'))

    // usePolling が呼ばれたことを確認
    expect(usePollingModule.usePolling).toHaveBeenCalledTimes(1)

    // パラメータを確認
    const callArgs = vi.mocked(usePollingModule.usePolling).mock.calls[0]
    expect(typeof callArgs[0]).toBe('function') // callback
    expect(callArgs[1]).toBe(15000) // interval: 15秒
    expect(callArgs[2]).toBe(true) // enabled: true
  })

  test('projectNameが変更されたらfetchDataが再実行される', async () => {
    const { rerender } = renderHook(
      ({ projectName, period }) => useProjectDetailPolling(projectName, period),
      { initialProps: { projectName: 'Project1', period: 'day' as const } }
    )

    // 初回callback実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(api.getProjectStats).toHaveBeenCalledWith('Project1')
    })

    vi.clearAllMocks()

    // projectNameを変更
    rerender({ projectName: 'Project2', period: 'day' as const })

    // rerenderで新しいcallbackがキャプチャされるので、それを実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(api.getProjectStats).toHaveBeenCalledWith('Project2')
    })
  })

  test('periodが変更されたらfetchDataが再実行される', async () => {
    const { rerender } = renderHook(
      ({ projectName, period }) => useProjectDetailPolling(projectName, period),
      { initialProps: { projectName: 'TestProject', period: 'day' as const } }
    )

    // 初回callback実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(api.getProjectTimeline).toHaveBeenCalledWith('TestProject', 'day', 30)
    })

    vi.clearAllMocks()

    // periodを変更
    rerender({ projectName: 'TestProject', period: 'week' as const })

    // rerenderで新しいcallbackがキャプチャされるので、それを実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(api.getProjectTimeline).toHaveBeenCalledWith('TestProject', 'week', 30)
    })
  })

  test('APIエラー時もエラー状態を更新する', async () => {
    vi.mocked(api.getProjectStats).mockRejectedValueOnce(new Error('Network error'))

    const { result } = renderHook(() => useProjectDetailPolling('TestProject', 'day'))

    // キャプチャしたcallbackを実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('Network error')
    expect(result.current.stats).toBe(null)
  })

  test('手動でcallbackを実行してポーリング動作をシミュレート', async () => {
    const { result } = renderHook(() => useProjectDetailPolling('TestProject', 'day'))

    // 初回callback実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(api.getProjectStats).toHaveBeenCalledTimes(1)

    // キャプチャしたcallbackを手動で実行（ポーリングをシミュレート）
    if (capturedCallback) {
      await capturedCallback()
    }

    // 2回目の呼び出しを確認
    expect(api.getProjectStats).toHaveBeenCalledTimes(2)
  })
})
