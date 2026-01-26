import { useState, useCallback } from 'react'
import { api, ApiError } from '@/lib/api/client'
import type {
  Project,
  ProjectGroup,
  TotalStats,
  TimeSeriesResponse,
} from '@/lib/api/types'
import { usePolling } from './usePolling'

interface UseProjectsPollingResult {
  projects: Project[]
  groups: ProjectGroup[]
  totalStats: TotalStats | null
  timeline: TimeSeriesResponse | null
  loading: boolean
  error: string | null
}

/**
 * プロジェクト一覧ページ用のポーリングフック
 * 15秒ごとに全データを自動更新する
 *
 * @param period - タイムライン表示期間
 * @returns プロジェクトデータと状態
 */
export function useProjectsPolling(
  period: 'day' | 'week' | 'month'
): UseProjectsPollingResult {
  const [projects, setProjects] = useState<Project[]>([])
  const [groups, setGroups] = useState<ProjectGroup[]>([])
  const [totalStats, setTotalStats] = useState<TotalStats | null>(null)
  const [timeline, setTimeline] = useState<TimeSeriesResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setError(null)

      const [projectsRes, groupsRes, statsRes, timelineRes] = await Promise.all([
        api.getProjects(),
        api.getProjectGroups(),
        api.getTotalStats(),
        api.getTotalTimeline(period, 30),
      ])

      setProjects(projectsRes.projects)
      setGroups(groupsRes.groups)
      setTotalStats(statsRes)
      setTimeline(timelineRes)
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
  }, [period])

  // 15秒ごとにポーリング
  usePolling(fetchData, 15000, true)

  return {
    projects,
    groups,
    totalStats,
    timeline,
    loading,
    error,
  }
}
