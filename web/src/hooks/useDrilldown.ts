/**
 * Drilldown state management hook
 *
 * 日付ドリルダウン機能の共通状態管理ロジックを提供します。
 * ProjectsPage、GroupDetailPage、ProjectDetailPage で共通して使用されます。
 */

import { useState, useEffect, useCallback } from 'react'
import { ApiError } from '@/lib/api/client'
import { isValidDateFormat } from '@/lib/utils/formatters'

interface UseDrilldownOptions<T> {
  /**
   * 選択された日付のデータを取得する関数
   * @param date - YYYY-MM-DD 形式の日付文字列
   */
  fetchData: (date: string) => Promise<T>
}

interface UseDrilldownResult<T> {
  /** 選択された日付 */
  selectedDate: string | null
  /** 取得されたデータ */
  data: T | null
  /** ローディング状態 */
  loading: boolean
  /** エラーメッセージ */
  error: string | null
  /**
   * 日付バッジクリック時のハンドラー（トグル機能）
   * 同じ日付をクリックすると選択解除される
   */
  handleDateClick: (dateStr: string) => void
  /** ドリルダウンパネルを閉じる */
  close: () => void
}

/**
 * Drilldown state management hook
 *
 * @example
 * ```tsx
 * const drilldown = useDrilldown({
 *   fetchData: (date) => api.getDailyStats(date)
 * })
 *
 * // 日付バッジのクリック
 * <button onClick={() => drilldown.handleDateClick('2026-01-27')}>
 *   1/27
 * </button>
 *
 * // ドリルダウンパネル
 * {drilldown.selectedDate && (
 *   <Card>
 *     {drilldown.loading && <LoadingSpinner />}
 *     {drilldown.error && <p>{drilldown.error}</p>}
 *     {drilldown.data && <DataDisplay data={drilldown.data} />}
 *   </Card>
 * )}
 * ```
 */
export function useDrilldown<T>({
  fetchData,
}: UseDrilldownOptions<T>): UseDrilldownResult<T> {
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const [data, setData] = useState<T | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!selectedDate) {
      setData(null)
      return
    }

    // 日付フォーマット検証
    if (!isValidDateFormat(selectedDate)) {
      setError('無効な日付フォーマットです')
      setData(null)
      return
    }

    const loadData = async () => {
      try {
        setLoading(true)
        setError(null)
        const result = await fetchData(selectedDate)
        setData(result)
      } catch (err) {
        const errorMsg =
          err instanceof ApiError ? err.message : 'データの取得に失敗しました'
        setError(errorMsg)
        setData(null)
      } finally {
        setLoading(false)
      }
    }

    loadData()
  }, [selectedDate, fetchData])

  const handleDateClick = useCallback((dateStr: string) => {
    setSelectedDate((prev) => (prev === dateStr ? null : dateStr))
  }, [])

  const close = useCallback(() => {
    setSelectedDate(null)
    setData(null)
  }, [])

  return {
    selectedDate,
    data,
    loading,
    error,
    handleDateClick,
    close,
  }
}
