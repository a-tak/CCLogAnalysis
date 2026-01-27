import { useState, useEffect, useMemo } from 'react'
import { useParams } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { api, ApiError } from '@/lib/api/client'
import type { SessionDetail } from '@/lib/api/types'
import { TokenBreakdownChart } from '@/components/charts/TokenBreakdownChart'
import { ModelUsageChart } from '@/components/charts/ModelUsageChart'
import { ConversationHistory } from '@/components/conversation/ConversationHistory'
import { Breadcrumb } from '@/components/navigation/Breadcrumb'
import { Calendar } from 'lucide-react'

function formatDate(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleString()
}

export function SessionDetailPage() {
  const { projectName, sessionId } = useParams<{ projectName: string; sessionId: string }>()
  const [session, setSession] = useState<SessionDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Date filter state
  const [selectedDateFilter, setSelectedDateFilter] = useState<string | null>(null)

  useEffect(() => {
    async function loadSessionDetail() {
      if (!projectName || !sessionId) return

      try {
        setLoading(true)
        setError(null)
        const data = await api.getSessionDetail(projectName, sessionId)
        setSession(data)
      } catch (err) {
        if (err instanceof ApiError) {
          setError(err.message)
        } else {
          setError('Failed to load session details')
        }
      } finally {
        setLoading(false)
      }
    }

    loadSessionDetail()
  }, [projectName, sessionId])

  // Helper to build breadcrumb items
  const buildBreadcrumbItems = (sessionData?: SessionDetail | null) => {
    const items = []
    if (projectName) {
      items.push({
        label: projectName,
        href: `/projects/${encodeURIComponent(projectName)}`,
      })
    }
    if (sessionData) {
      items.push({ label: `セッション ${sessionData.id.substring(0, 8)}...` })
    } else if (sessionId) {
      items.push({ label: `セッション ${sessionId.substring(0, 8)}...` })
    }
    return items
  }

  // Generate available dates from session data
  const availableDates = useMemo(() => {
    if (!session) return []

    const dates = new Set<string>()

    // Extract dates from tool calls
    session.toolCalls.forEach((call) => {
      const date = new Date(call.timestamp).toISOString().split('T')[0]
      dates.add(date)
    })

    // Extract dates from messages
    session.messages.forEach((msg) => {
      const date = new Date(msg.timestamp).toISOString().split('T')[0]
      dates.add(date)
    })

    return Array.from(dates).sort().reverse() // 降順
  }, [session])

  // Filter tool calls by date
  const filteredToolCalls = useMemo(() => {
    if (!session || !selectedDateFilter) return session?.toolCalls || []

    return session.toolCalls.filter((call) => {
      const callDate = new Date(call.timestamp).toISOString().split('T')[0]
      return callDate === selectedDateFilter
    })
  }, [session, selectedDateFilter])

  // Filter messages by date
  const filteredMessages = useMemo(() => {
    if (!session || !selectedDateFilter) return session?.messages || []

    return session.messages.filter((msg) => {
      const msgDate = new Date(msg.timestamp).toISOString().split('T')[0]
      return msgDate === selectedDateFilter
    })
  }, [session, selectedDateFilter])

  if (loading) {
    return (
      <div className="space-y-4">
        <Breadcrumb items={buildBreadcrumbItems()} />
        <div>
          <h1 className="text-3xl font-bold tracking-tight">セッション詳細</h1>
          <p className="text-muted-foreground">読み込み中...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-4">
        <Breadcrumb items={buildBreadcrumbItems()} />
        <div>
          <h1 className="text-3xl font-bold tracking-tight">セッション詳細</h1>
          <p className="text-destructive">{error}</p>
        </div>
      </div>
    )
  }

  if (!session) {
    return (
      <div className="space-y-4">
        <Breadcrumb items={buildBreadcrumbItems()} />
        <div>
          <h1 className="text-3xl font-bold tracking-tight">セッション詳細</h1>
          <p className="text-muted-foreground">セッションが見つかりません</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <Breadcrumb items={buildBreadcrumbItems(session)} />
      <div>
        <h1 className="text-3xl font-bold tracking-tight">セッション詳細</h1>
        <p className="text-muted-foreground">
          {session.projectPath} ({session.gitBranch})
        </p>
      </div>

      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle>Basic Information</CardTitle>
        </CardHeader>
        <CardContent>
          <dl className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <dt className="font-medium text-muted-foreground">Session ID</dt>
              <dd className="mt-1">{session.id}</dd>
            </div>
            <div>
              <dt className="font-medium text-muted-foreground">Duration</dt>
              <dd className="mt-1">{session.duration}</dd>
            </div>
            <div>
              <dt className="font-medium text-muted-foreground">Start Time</dt>
              <dd className="mt-1">{formatDate(session.startTime)}</dd>
            </div>
            <div>
              <dt className="font-medium text-muted-foreground">End Time</dt>
              <dd className="mt-1">{formatDate(session.endTime)}</dd>
            </div>
            <div>
              <dt className="font-medium text-muted-foreground">Errors</dt>
              <dd className="mt-1">
                {session.errorCount > 0 ? (
                  <span className="text-destructive font-medium">{session.errorCount}</span>
                ) : (
                  <span className="text-muted-foreground">0</span>
                )}
              </dd>
            </div>
          </dl>
        </CardContent>
      </Card>

      {/* Date Filter */}
      {availableDates.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Calendar className="h-5 w-5" />
              日付フィルタ
            </CardTitle>
            <CardDescription>
              特定の日付のデータのみを表示できます
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Select
              value={selectedDateFilter || 'all'}
              onValueChange={(value: string) => setSelectedDateFilter(value === 'all' ? null : value)}
            >
              <SelectTrigger className="w-full md:w-[300px]">
                <SelectValue placeholder="すべての日付" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">すべての日付</SelectItem>
                {availableDates.map((date) => {
                  const dateObj = new Date(date)
                  const displayDate = dateObj.toLocaleDateString('ja-JP', {
                    year: 'numeric',
                    month: 'short',
                    day: 'numeric',
                  })
                  return (
                    <SelectItem key={date} value={date}>
                      {displayDate}
                    </SelectItem>
                  )
                })}
              </SelectContent>
            </Select>
            {selectedDateFilter && (
              <p className="text-sm text-muted-foreground mt-2">
                フィルタ適用中: {new Date(selectedDateFilter).toLocaleDateString('ja-JP')}
              </p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Token Summary */}
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <CardTitle>Token Usage</CardTitle>
            <CardDescription>Total tokens used in this session</CardDescription>
          </CardHeader>
          <CardContent>
            <dl className="grid grid-cols-2 gap-4 text-sm">
              <div>
                <dt className="font-medium text-muted-foreground">Input Tokens</dt>
                <dd className="mt-1 text-lg font-semibold">
                  {session.totalTokens.inputTokens.toLocaleString()}
                </dd>
              </div>
              <div>
                <dt className="font-medium text-muted-foreground">Output Tokens</dt>
                <dd className="mt-1 text-lg font-semibold">
                  {session.totalTokens.outputTokens.toLocaleString()}
                </dd>
              </div>
              <div>
                <dt className="font-medium text-muted-foreground">Cache Creation</dt>
                <dd className="mt-1 text-lg font-semibold">
                  {session.totalTokens.cacheCreationInputTokens.toLocaleString()}
                </dd>
              </div>
              <div>
                <dt className="font-medium text-muted-foreground">Cache Read</dt>
                <dd className="mt-1 text-lg font-semibold">
                  {session.totalTokens.cacheReadInputTokens.toLocaleString()}
                </dd>
              </div>
              <div className="col-span-2">
                <dt className="font-medium text-muted-foreground">Total Tokens</dt>
                <dd className="mt-1 text-2xl font-bold text-primary">
                  {session.totalTokens.totalTokens.toLocaleString()}
                </dd>
              </div>
            </dl>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Token Breakdown</CardTitle>
            <CardDescription>Distribution of token types</CardDescription>
          </CardHeader>
          <CardContent>
            <TokenBreakdownChart tokens={session.totalTokens} />
          </CardContent>
        </Card>
      </div>

      {/* Model Usage */}
      <Card>
        <CardHeader>
          <CardTitle>Model Usage</CardTitle>
          <CardDescription>Tokens used per model</CardDescription>
        </CardHeader>
        <CardContent className="space-y-6">
          <ModelUsageChart modelUsage={session.modelUsage} />

          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Model</TableHead>
                <TableHead className="text-right">Input</TableHead>
                <TableHead className="text-right">Output</TableHead>
                <TableHead className="text-right">Cache Create</TableHead>
                <TableHead className="text-right">Cache Read</TableHead>
                <TableHead className="text-right">Total</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {session.modelUsage.map((usage) => (
                <TableRow key={usage.model}>
                  <TableCell className="font-medium">{usage.model}</TableCell>
                  <TableCell className="text-right">
                    {usage.tokens.inputTokens.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {usage.tokens.outputTokens.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {usage.tokens.cacheCreationInputTokens.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right">
                    {usage.tokens.cacheReadInputTokens.toLocaleString()}
                  </TableCell>
                  <TableCell className="text-right font-semibold">
                    {usage.tokens.totalTokens.toLocaleString()}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Tool Calls */}
      <Card>
        <CardHeader>
          <CardTitle>Tool Calls</CardTitle>
          <CardDescription>
            {filteredToolCalls.length} tool calls
            {selectedDateFilter && ` (filtered from ${session.toolCalls.length})`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead>
                <TableHead>Tool</TableHead>
                <TableHead>Input</TableHead>
                <TableHead>Status</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredToolCalls.slice(0, 20).map((call, index) => (
                <TableRow key={index}>
                  <TableCell className="text-muted-foreground">
                    {formatDate(call.timestamp)}
                  </TableCell>
                  <TableCell className="font-medium">{call.name}</TableCell>
                  <TableCell className="max-w-md truncate text-muted-foreground">
                    {JSON.stringify(call.input || {})}
                  </TableCell>
                  <TableCell>
                    {call.isError ? (
                      <span className="text-destructive">Error</span>
                    ) : (
                      <span className="text-muted-foreground">Success</span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
          {filteredToolCalls.length > 20 && (
            <p className="mt-4 text-sm text-muted-foreground">
              Showing 20 of {filteredToolCalls.length} tool calls
            </p>
          )}
        </CardContent>
      </Card>

      {/* Conversation History */}
      <Card>
        <CardHeader>
          <CardTitle>Conversation History</CardTitle>
          <CardDescription>
            {filteredMessages.length} messages
            {selectedDateFilter && ` (filtered from ${session.messages.length})`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ConversationHistory messages={filteredMessages.slice(0, 50)} />
          {filteredMessages.length > 50 && (
            <p className="mt-4 text-sm text-muted-foreground">
              Showing 50 of {filteredMessages.length} messages
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
