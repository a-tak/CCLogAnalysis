import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import type { SessionSummary } from '@/lib/api/types'

interface SessionListTabProps {
  sessions: SessionSummary[]
  loading: boolean
}

export function SessionListTab({ sessions, loading }: SessionListTabProps) {
  const [currentPage, setCurrentPage] = useState(1)
  const pageSize = 20

  // ページネーション計算
  const totalPages = Math.ceil(sessions.length / pageSize)
  const startIndex = (currentPage - 1) * pageSize
  const endIndex = startIndex + pageSize
  const displayedSessions = useMemo(
    () => sessions.slice(startIndex, endIndex),
    [sessions, startIndex, endIndex]
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
      <Card>
        <CardHeader>
          <CardTitle>
            セッション一覧 ({sessions.length.toLocaleString()}件)
          </CardTitle>
        </CardHeader>
        <CardContent>
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
        </CardContent>
      </Card>

      {/* ページネーション */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">
            {startIndex + 1} - {Math.min(endIndex, sessions.length)} / {sessions.length}件
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
