import { useState, useCallback } from 'react'
import { api, ApiError } from '@/lib/api/client'
import type { ProjectStats, TimeSeriesResponse } from '@/lib/api/types'
import { usePolling } from './usePolling'

interface UseProjectDetailPollingResult {
  stats: ProjectStats | null
  timeline: TimeSeriesResponse | null
  loading: boolean
  error: string | null
}

/**
 * プロジェクト詳細ページ用のポーリングフック
 * 15秒ごとにプロジェクトの統計とタイムラインを自動更新する
 *
 * @param projectName - プロジェクト名
 * @param period - タイムライン表示期間
 * @returns プロジェクト詳細データと状態
 */
export function useProjectDetailPolling(
  projectName: string,
  period: 'day' | 'week' | 'month'
): UseProjectDetailPollingResult {
  const [stats, setStats] = useState<ProjectStats | null>(null)
  const [timeline, setTimeline] = useState<TimeSeriesResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setError(null)

      const [statsData, timelineData] = await Promise.all([
        api.getProjectStats(projectName),
        api.getProjectTimeline(projectName, period, 30),
      ])

      setStats(statsData)
      setTimeline(timelineData)
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message)
      } else if (err instanceof Error) {
        setError(err.message)
      } else {
        setError('Failed to load data')
      }
    } finally {
      setLoading(false)
    }
  }, [projectName, period])

  // 15秒ごとにポーリング
  usePolling(fetchData, 15000, true)

  return {
    stats,
    timeline,
    loading,
    error,
  }
}
