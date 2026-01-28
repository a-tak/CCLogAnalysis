import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import type { SessionSummary } from '@/lib/api/types'

interface DateRangeFilter {
  startDate: string | null  // YYYY-MM-DD形式
  endDate: string | null    // YYYY-MM-DD形式
}

interface SessionListTabProps {
  sessions: SessionSummary[]
  loading: boolean
}

export function SessionListTab({ sessions, loading }: SessionListTabProps) {
  const [currentPage, setCurrentPage] = useState(1)
  const [dateFilter, setDateFilter] = useState<DateRangeFilter>({
    startDate: null,
    endDate: null
  })
  const pageSize = 20

  // ハンドラ関数
  function handleDateChange(field: keyof DateRangeFilter) {
    return (e: React.ChangeEvent<HTMLInputElement>) => {
      setDateFilter(prev => ({
        ...prev,
        [field]: e.target.value || null
      }))
      setCurrentPage(1)
    }
  }

  function handleClearFilter(): void {
    setDateFilter({ startDate: null, endDate: null })
    setCurrentPage(1)
  }

  // フィルタリング
  const filteredSessions = useMemo(() => {
    return sessions.filter(session => {
      // 日付範囲チェック
      const sessionDate = new Date(session.startTime).toISOString().split('T')[0]

      if (dateFilter.startDate && sessionDate < dateFilter.startDate) {
        return false
      }

      if (dateFilter.endDate && sessionDate > dateFilter.endDate) {
        return false
      }

      return true
    })
  }, [sessions, dateFilter])

  // ページネーション計算（フィルタリング後）
  const totalPages = Math.ceil(filteredSessions.length / pageSize)
  const startIndex = (currentPage - 1) * pageSize
  const endIndex = startIndex + pageSize
  const displayedSessions = useMemo(
    () => filteredSessions.slice(startIndex, endIndex),
    [filteredSessions, startIndex, endIndex]
  )

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    )
  }

  if (sessions.length === 0) {
    return (
      <Card>
        <CardContent className="py-12 text-center text-muted-foreground">
          セッションが見つかりません
        </CardContent>
      </Card>
    )
  }

  return (
    <div className="space-y-4">
      {/* フィルタカード */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>フィルタ</CardTitle>
            {(dateFilter.startDate || dateFilter.endDate) && (
              <Button
                variant="outline"
                size="sm"
                onClick={handleClearFilter}
              >
                クリア
              </Button>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid gap-4">
            <div>
              <label className="text-sm font-medium mb-2 block">期間</label>
              <div className="flex gap-2 items-center">
                <input
                  type="date"
                  value={dateFilter.startDate || ''}
                  onChange={handleDateChange('startDate')}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                />
                <span className="text-muted-foreground">～</span>
                <input
                  type="date"
                  value={dateFilter.endDate || ''}
                  onChange={handleDateChange('endDate')}
                  className="flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                />
              </div>
              {(dateFilter.startDate || dateFilter.endDate) && (
                <p className="text-sm text-muted-foreground mt-2">
                  {dateFilter.startDate && `${new Date(dateFilter.startDate).toLocaleDateString('ja-JP')} から`}
                  {dateFilter.endDate && ` ${new Date(dateFilter.endDate).toLocaleDateString('ja-JP')} まで`}
                </p>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* セッション一覧カード */}
      <Card>
        <CardHeader>
          <CardTitle>
            セッション一覧 ({filteredSessions.length.toLocaleString()}件
            {filteredSessions.length !== sessions.length &&
              ` / ${sessions.length.toLocaleString()}件中`})
          </CardTitle>
        </CardHeader>
        <CardContent>
          {filteredSessions.length === 0 ? (
            <div className="py-12 text-center text-muted-foreground">
              条件に一致するセッションが見つかりません
            </div>
          ) : (
            <Table>
            <TableHeader>
              <TableRow>
                <TableHead>セッションID</TableHead>
                <TableHead>ブランチ</TableHead>
                <TableHead>開始時刻</TableHead>
                <TableHead>最初のメッセージ</TableHead>
                <TableHead className="text-right">トークン</TableHead>
                <TableHead className="text-right">エラー</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {displayedSessions.map((session) => (
                <TableRow key={session.id}>
                  <TableCell className="font-medium">
                    <Link
                      to={`/projects/${encodeURIComponent(session.projectName)}/sessions/${encodeURIComponent(session.id)}`}
                      className="text-primary hover:underline"
                    >
                      {session.id.substring(0, 8)}...
                    </Link>
                  </TableCell>
                  <TableCell>{session.gitBranch}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {new Date(session.startTime).toLocaleString('ja-JP')}
                  </TableCell>
                  <TableCell className="max-w-md truncate" title={session.firstUserMessage}>
                    {session.firstUserMessage || '-'}
                  </TableCell>
                  <TableCell className="text-right">
                    {session.totalTokens.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {session.errorCount > 0 ? (
                      <span className="text-destructive font-medium">
                        {session.errorCount}
                      </span>
                    ) : (
                      <span className="text-muted-foreground">0</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          )}
        </CardContent>
      </Card>

      {/* ページネーション */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            {startIndex + 1} - {Math.min(endIndex, filteredSessions.length)} / {filteredSessions.length}件
          </p>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentPage(currentPage - 1)}
              disabled={currentPage === 1}
            >
              <ChevronLeft className="h-4 w-4 mr-1" />
              前へ
            </Button>
            <span className="flex items-center px-4 text-sm text-muted-foreground">
              {currentPage} / {totalPages}
            </span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setCurrentPage(currentPage + 1)}
              disabled={currentPage === totalPages}
            >
              次へ
              <ChevronRight className="h-4 w-4 ml-1" />
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
