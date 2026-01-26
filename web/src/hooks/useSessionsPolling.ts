import { useState, useCallback } from 'react'
import { api, ApiError } from '@/lib/api/client'
import type { SessionSummary } from '@/lib/api/types'
import { usePolling } from './usePolling'

interface UseSessionsPollingResult {
  sessions: SessionSummary[]
  loading: boolean
  error: string | null
}

/**
 * セッション一覧ページ用のポーリングフック
 * 15秒ごとにセッションデータを自動更新する
 *
 * @param projectName - プロジェクト名
 * @returns セッションデータと状態
 */
export function useSessionsPolling(projectName: string): UseSessionsPollingResult {
  const [sessions, setSessions] = useState<SessionSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isInitialLoad, setIsInitialLoad] = useState(true)

  const fetchData = useCallback(async () => {
    try {
      setError(null)

      const response = await api.getSessions(projectName)
      setSessions(response.sessions)

      if (isInitialLoad) {
        setIsInitialLoad(false)
      }
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Failed to load sessions')
      }

      if (isInitialLoad) {
        setIsInitialLoad(false)
      }
    } finally {
      if (isInitialLoad) {
        setLoading(false)
      }
    }
  }, [projectName, isInitialLoad])

  // 15秒ごとにポーリング
  usePolling(fetchData, 15000, true)

  return {
    sessions,
    loading,
    error,
  }
}
