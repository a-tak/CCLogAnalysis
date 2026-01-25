import { useEffect, useState } from 'react'
import { api } from '@/lib/api/client'
import type { ScanStatus } from '@/lib/api/types'

export function ScanStatusIndicator() {
  const [scanStatus, setScanStatus] = useState<ScanStatus | null>(null)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    let intervalId: number | null = null

    async function checkScanStatus() {
      try {
        const status = await api.getScanStatus()
        setScanStatus(status)

        // スキャン実行中の場合のみ表示
        if (status.status === 'running') {
          setVisible(true)
          // ポーリング間隔: 2秒
          if (!intervalId) {
            intervalId = window.setInterval(checkScanStatus, 2000)
          }
        } else {
          setVisible(false)
          if (intervalId) {
            clearInterval(intervalId)
            intervalId = null
          }
        }
      } catch (error) {
        console.error('Failed to fetch scan status:', error)
      }
    }

    checkScanStatus()

    return () => {
      if (intervalId) {
        clearInterval(intervalId)
      }
    }
  }, [])

  if (!visible || !scanStatus) {
    return null
  }

  // 処理済みセッション数（保存 + スキップ）
  const processedSessions = scanStatus.sessionsSynced + scanStatus.sessionsSkipped

  return (
    <div className="fixed bottom-4 right-4 bg-primary text-primary-foreground p-4 rounded-lg shadow-lg max-w-sm z-50">
      <div className="flex items-center gap-3">
        <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-white"></div>
        <div>
          <div className="font-semibold">初期スキャン実行中...</div>
          <div className="text-sm opacity-90">
            {processedSessions.toLocaleString()} セッション処理中
          </div>
        </div>
      </div>
    </div>
  )
}
