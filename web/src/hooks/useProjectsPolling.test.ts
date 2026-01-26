import { renderHook, waitFor } from '@testing-library/react'
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { useProjectsPolling } from './useProjectsPolling'
import { api } from '@/lib/api/client'
import * as usePollingModule from './usePolling'

// usePolling をモック
vi.mock('./usePolling', () => ({
  usePolling: vi.fn(),
}))

vi.mock('@/lib/api/client', () => ({
  api: {
    getProjects: vi.fn(),
    getProjectGroups: vi.fn(),
    getTotalStats: vi.fn(),
    getTotalTimeline: vi.fn(),
  },
  ApiError: class ApiError extends Error {
    status: number
    constructor(message: string, status: number) {
      super(message)
      this.status = status
    }
  },
}))

describe('useProjectsPolling', () => {
  const mockProjectsResponse = {
    projects: [
      { name: 'Project1', decodedPath: '/path/to/project1', sessionCount: 5 },
    ],
  }

  const mockGroupsResponse = {
    groups: [
      { id: 1, name: 'Group1', gitRoot: '/git/root', updatedAt: '2026-01-26T00:00:00Z' },
    ],
  }

  const mockTotalStats = {
    totalGroups: 1,
    totalSessions: 5,
    totalTokens: 10000,
    totalInputTokens: 4000,
    totalOutputTokens: 6000,
    errorRate: 0.05,
  }

  const mockTimelineResponse = {
    data: [
      {
        periodStart: '2026-01-26T00:00:00Z',
        totalInputTokens: 4000,
        totalOutputTokens: 6000,
        totalTokens: 10000,
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

    vi.mocked(api.getProjects).mockResolvedValue(mockProjectsResponse)
    vi.mocked(api.getProjectGroups).mockResolvedValue(mockGroupsResponse)
    vi.mocked(api.getTotalStats).mockResolvedValue(mockTotalStats)
    vi.mocked(api.getTotalTimeline).mockResolvedValue(mockTimelineResponse)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  test('初回ロード時に全データを取得する', async () => {
    const { result } = renderHook(() => useProjectsPolling('day'))

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
    expect(api.getProjects).toHaveBeenCalledTimes(1)
    expect(api.getProjectGroups).toHaveBeenCalledTimes(1)
    expect(api.getTotalStats).toHaveBeenCalledTimes(1)
    expect(api.getTotalTimeline).toHaveBeenCalledWith('day', 30)

    // データが正しく設定されている
    expect(result.current.projects).toEqual(mockProjectsResponse.projects)
    expect(result.current.groups).toEqual(mockGroupsResponse.groups)
    expect(result.current.totalStats).toEqual(mockTotalStats)
    expect(result.current.timeline).toEqual(mockTimelineResponse)
    expect(result.current.error).toBe(null)
  })

  test('usePollingに正しいパラメータを渡す', () => {
    renderHook(() => useProjectsPolling('day'))

    // usePolling が呼ばれたことを確認
    expect(usePollingModule.usePolling).toHaveBeenCalledTimes(1)

    // パラメータを確認
    const callArgs = vi.mocked(usePollingModule.usePolling).mock.calls[0]
    expect(typeof callArgs[0]).toBe('function') // callback
    expect(callArgs[1]).toBe(15000) // interval: 15秒
    expect(callArgs[2]).toBe(true) // enabled: true
  })

  test('periodが変更されたらfetchDataが再実行される', async () => {
    const { rerender } = renderHook(
      ({ period }) => useProjectsPolling(period),
      { initialProps: { period: 'day' as const } }
    )

    // 初回callback実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(api.getTotalTimeline).toHaveBeenCalledWith('day', 30)
    })

    vi.clearAllMocks()

    // periodを変更
    rerender({ period: 'week' as const })

    // rerenderで新しいcallbackがキャプチャされるので、それを実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(api.getTotalTimeline).toHaveBeenCalledWith('week', 30)
    })
  })

  test('APIエラー時もエラー状態を更新する', async () => {
    vi.mocked(api.getProjects).mockRejectedValueOnce(new Error('Network error'))

    const { result } = renderHook(() => useProjectsPolling('day'))

    // キャプチャしたcallbackを実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('Network error')
    expect(result.current.projects).toEqual([])
  })

  test('手動でcallbackを実行してポーリング動作をシミュレート', async () => {
    const { result } = renderHook(() => useProjectsPolling('day'))

    // 初回callback実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(api.getProjects).toHaveBeenCalledTimes(1)

    // キャプチャしたcallbackを手動で実行（ポーリングをシミュレート）
    if (capturedCallback) {
      await capturedCallback()
    }

    // 2回目の呼び出しを確認
    expect(api.getProjects).toHaveBeenCalledTimes(2)
  })
})
