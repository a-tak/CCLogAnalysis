import { renderHook, waitFor } from '@testing-library/react'
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest'
import { useSessionsPolling } from './useSessionsPolling'
import { api } from '@/lib/api/client'
import * as usePollingModule from './usePolling'

// usePolling をモック
vi.mock('./usePolling', () => ({
  usePolling: vi.fn(),
}))

vi.mock('@/lib/api/client', () => ({
  api: {
    getSessions: vi.fn(),
  },
  ApiError: class ApiError extends Error {
    status: number
    constructor(message: string, status: number) {
      super(message)
      this.status = status
    }
  },
}))

describe('useSessionsPolling', () => {
  const mockSessionsResponse = {
    sessions: [
      {
        sessionId: 'session-1',
        projectName: 'TestProject',
        firstMessage: 'Test message',
        branch: 'main',
        startTime: '2026-01-26T00:00:00Z',
      },
      {
        sessionId: 'session-2',
        projectName: 'TestProject',
        firstMessage: 'Another message',
        branch: 'develop',
        startTime: '2026-01-26T01:00:00Z',
      },
    ],
  }

  let capturedCallback: (() => Promise<void>) | null = null

  beforeEach(() => {
    capturedCallback = null

    // usePolling が呼ばれたときに callback をキャプチャのみ（初回実行はしない）
    vi.mocked(usePollingModule.usePolling).mockImplementation((callback: any) => {
      capturedCallback = callback
    })

    vi.mocked(api.getSessions).mockResolvedValue(mockSessionsResponse)
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  test('初回ロード時にセッションデータを取得する', async () => {
    const { result } = renderHook(() => useSessionsPolling('TestProject'))

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
    expect(api.getSessions).toHaveBeenCalledWith('TestProject')

    // データが正しく設定されている
    expect(result.current.sessions).toEqual(mockSessionsResponse.sessions)
    expect(result.current.error).toBe(null)
  })

  test('usePollingに正しいパラメータを渡す', () => {
    renderHook(() => useSessionsPolling('TestProject'))

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
      ({ projectName }) => useSessionsPolling(projectName),
      { initialProps: { projectName: 'Project1' } }
    )

    // 初回callback実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(api.getSessions).toHaveBeenCalledWith('Project1')
    })

    vi.clearAllMocks()

    // projectNameを変更
    rerender({ projectName: 'Project2' })

    // rerenderで新しいcallbackがキャプチャされるので、それを実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(api.getSessions).toHaveBeenCalledWith('Project2')
    })
  })

  test('APIエラー時もエラー状態を更新する', async () => {
    vi.mocked(api.getSessions).mockRejectedValueOnce(new Error('Network error'))

    const { result } = renderHook(() => useSessionsPolling('TestProject'))

    // キャプチャしたcallbackを実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(result.current.error).toBe('Network error')
    expect(result.current.sessions).toEqual([])
  })

  test('手動でcallbackを実行してポーリング動作をシミュレート', async () => {
    const { result } = renderHook(() => useSessionsPolling('TestProject'))

    // 初回callback実行
    if (capturedCallback) {
      await capturedCallback()
    }

    await waitFor(() => {
      expect(result.current.loading).toBe(false)
    })

    expect(api.getSessions).toHaveBeenCalledTimes(1)

    // キャプチャしたcallbackを手動で実行（ポーリングをシミュレート）
    if (capturedCallback) {
      await capturedCallback()
    }

    // 2回目の呼び出しを確認
    expect(api.getSessions).toHaveBeenCalledTimes(2)
  })
})
