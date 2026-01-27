import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api, ApiError } from '@/lib/api/client'
import type { ProjectStats, TimeSeriesResponse, SessionSummary, ProjectDailyStatsResponse } from '@/lib/api/types'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { LoadingSpinner } from '@/components/ui/loading-spinner'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { Activity, Zap, TrendingUp, AlertCircle, X, GitBranch } from 'lucide-react'
import { SessionListTab } from '@/components/sessions/SessionListTab'
import { Breadcrumb } from '@/components/navigation/Breadcrumb'
import { useProjectDetailPolling } from '@/hooks/useProjectDetailPolling'

export default function ProjectDetailPage() {
  const { name } = useParams<{ name: string }>()
  const [period, setPeriod] = useState<'day' | 'week' | 'month'>('day')

  // 15秒ごとにポーリングしてデータを自動更新
  const { stats, timeline, loading, error } = useProjectDetailPolling(name || '', period)

  const [sessions, setSessions] = useState<SessionSummary[]>([])
  const [sessionsLoading, setSessionsLoading] = useState(false)
  const [activeTab, setActiveTab] = useState('stats')

  // Drilldown state
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const [dailyStats, setDailyStats] = useState<ProjectDailyStatsResponse | null>(null)
  const [dailyLoading, setDailyLoading] = useState(false)
  const [dailyError, setDailyError] = useState<string | null>(null)

  useEffect(() => {
    if (!name || activeTab !== 'sessions') return

    const fetchSessions = async () => {
      setSessionsLoading(true)
      try {
        const data = await api.getSessions(name)
        setSessions(data.sessions)
      } catch (err) {
        console.error('Failed to fetch sessions:', err)
      } finally {
        setSessionsLoading(false)
      }
    }

    fetchSessions()
  }, [name, activeTab])

  // Fetch daily stats when date is selected
  useEffect(() => {
    if (!selectedDate || !name) {
      setDailyStats(null)
      return
    }

    const dateToFetch = selectedDate
    const projectName = name

    async function loadDailyStats() {
      // 日付フォーマット検証
      if (!isValidDateFormat(dateToFetch)) {
        setDailyError('無効な日付フォーマットです')
        setDailyStats(null)
        return
      }

      try {
        setDailyLoading(true)
        setDailyError(null)
        const stats = await api.getProjectDailyStats(projectName, dateToFetch)
        setDailyStats(stats)
      } catch (err) {
        const errorMsg = err instanceof ApiError ? err.message : 'データの取得に失敗しました'
        setDailyError(errorMsg)
        setDailyStats(null)
      } finally {
        setDailyLoading(false)
      }
    }

    loadDailyStats()
  }, [selectedDate, name])

  const isValidDateFormat = (dateStr: string): boolean => {
    // YYYY-MM-DD 形式の検証
    const datePattern = /^\d{4}-\d{2}-\d{2}$/
    if (!datePattern.test(dateStr)) {
      return false
    }
    const date = new Date(dateStr)
    return date instanceof Date && !isNaN(date.getTime())
  }

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('ja-JP', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  // Handle date badge click (toggle)
  const handleDateClick = (dateStr: string) => {
    if (selectedDate === dateStr) {
      setSelectedDate(null)
    } else {
      setSelectedDate(dateStr)
    }
  }

  const closeDrilldown = () => {
    setSelectedDate(null)
    setDailyStats(null)
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <LoadingSpinner />
          <p className="text-muted-foreground mt-4">読み込み中...</p>
        </div>
      </div>
    )
  }

  if (error || !stats) {
    return (
      <div className="container mx-auto py-8">
        <Breadcrumb items={[{ label: 'エラー' }]} />
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">エラー</CardTitle>
          </CardHeader>
          <CardContent>
            <p>{error || '統計の取得に失敗しました'}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const formatNumber = (num: number) => num.toLocaleString('ja-JP')
  const formatPercent = (num: number) => (num * 100).toFixed(1) + '%'

  return (
    <div className="container mx-auto py-8">
      {/* Breadcrumb */}
      <Breadcrumb items={[{ label: name || 'プロジェクト' }]} />

      <h1 className="text-3xl font-bold mb-6">{name}</h1>

      {/* タブUI */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList className="mb-6">
          <TabsTrigger value="stats">統計・グラフ</TabsTrigger>
          <TabsTrigger value="sessions">セッション一覧</TabsTrigger>
        </TabsList>

        {/* 統計タブ */}
        <TabsContent value="stats">
          {/* Summary Cards */}
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">総セッション数</CardTitle>
                <Activity className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{formatNumber(stats.totalSessions)}</div>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">総トークン数</CardTitle>
                <Zap className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{formatNumber(stats.totalTokens)}</div>
                <p className="text-xs text-muted-foreground">
                  入力: {formatNumber(stats.totalInputTokens)} / 出力: {formatNumber(stats.totalOutputTokens)}
                </p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">平均トークン数</CardTitle>
                <TrendingUp className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{formatNumber(stats.avgTokens)}</div>
                <p className="text-xs text-muted-foreground">セッションあたり</p>
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">エラー率</CardTitle>
                <AlertCircle className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{formatPercent(stats.errorRate)}</div>
              </CardContent>
            </Card>
          </div>

          {/* Timeline Chart */}
          <Card className="mb-6">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>トークン使用量推移</CardTitle>
                  <CardDescription>時系列でのトークン使用状況</CardDescription>
                </div>
                <Tabs value={period} onValueChange={(v: string) => setPeriod(v as 'day' | 'week' | 'month')}>
                  <TabsList>
                    <TabsTrigger value="day">日別</TabsTrigger>
                    <TabsTrigger value="week">週別</TabsTrigger>
                    <TabsTrigger value="month">月別</TabsTrigger>
                  </TabsList>
                </Tabs>
              </div>
            </CardHeader>
            <CardContent>
              {timeline && timeline.data.length > 0 ? (
                <ResponsiveContainer width="100%" height={300}>
                  <LineChart data={timeline.data}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis
                      dataKey="periodStart"
                      tickFormatter={(value) => {
                        const date = new Date(value)
                        if (period === 'day') return `${date.getMonth() + 1}/${date.getDate()}`
                        if (period === 'week') return `${date.getMonth() + 1}/${date.getDate()}`
                        return `${date.getFullYear()}/${date.getMonth() + 1}`
                      }}
                    />
                    <YAxis />
                    <Tooltip
                      labelFormatter={(value) => {
                        const date = new Date(value as string)
                        return date.toLocaleDateString('ja-JP')
                      }}
                    />
                    <Legend />
                    <Line
                      type="monotone"
                      dataKey="totalInputTokens"
                      stroke="#8884d8"
                      name="入力トークン"
                      strokeWidth={2}
                    />
                    <Line
                      type="monotone"
                      dataKey="totalOutputTokens"
                      stroke="#82ca9d"
                      name="出力トークン"
                      strokeWidth={2}
                    />
                  </LineChart>
                </ResponsiveContainer>
              ) : (
                <div className="text-center py-8 text-muted-foreground">
                  データがありません
                </div>
              )}
            </CardContent>
            {period === 'day' && !loading && timeline && timeline.data.length > 0 && (
              <CardContent className="pt-0">
                <div className="text-sm text-muted-foreground mb-2">日付を選択してドリルダウン:</div>
                <div className="flex flex-wrap gap-2">
                  {[...timeline.data].reverse().map((item) => {
                    const date = new Date(item.periodStart)
                    const dateStr = date.toISOString().split('T')[0]
                    const displayDate = `${date.getMonth() + 1}/${date.getDate()}`
                    const isSelected = selectedDate === dateStr
                    return (
                      <button
                        key={dateStr}
                        type="button"
                        onClick={() => handleDateClick(dateStr)}
                        className={isSelected ?
                          "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 border-transparent bg-primary text-primary-foreground hover:bg-primary/80 cursor-pointer" :
                          "inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2 border-border text-foreground cursor-pointer hover:bg-primary hover:text-primary-foreground"
                        }
                      >
                        {displayDate}
                      </button>
                    )
                  })}
                </div>
              </CardContent>
            )}
          </Card>

          {/* Drilldown Panel */}
          {selectedDate && (
            <Card className="border-primary">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      <Zap className="h-5 w-5" />
                      {formatDate(selectedDate)} のセッション別トークン使用量
                    </CardTitle>
                    <CardDescription>
                      セッションをクリックすると詳細ページに移動します
                    </CardDescription>
                  </div>
                  <Button variant="ghost" size="icon" onClick={closeDrilldown}>
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                {dailyLoading && <LoadingSpinner />}
                {!dailyLoading && dailyError && (
                  <div className="flex items-center justify-center py-8">
                    <p className="text-sm text-destructive">{dailyError}</p>
                  </div>
                )}
                {!dailyLoading && !dailyError && dailyStats && dailyStats.sessions.length > 0 && (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>セッションID</TableHead>
                        <TableHead>ブランチ</TableHead>
                        <TableHead>開始時刻</TableHead>
                        <TableHead className="text-right">入力トークン</TableHead>
                        <TableHead className="text-right">出力トークン</TableHead>
                        <TableHead className="text-right">合計トークン</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {dailyStats.sessions.map((session) => (
                        <TableRow key={session.id} className="cursor-pointer hover:bg-accent">
                          <TableCell className="font-medium">
                            <Link
                              to={`/projects/${encodeURIComponent(name!)}/sessions/${encodeURIComponent(session.id)}`}
                              className="text-primary hover:underline flex items-center gap-2"
                            >
                              <Activity className="h-4 w-4" />
                              {session.id.substring(0, 8)}...
                            </Link>
                          </TableCell>
                          <TableCell>
                            <Badge variant="outline" className="flex items-center gap-1">
                              <GitBranch className="h-3 w-3" />
                              {session.gitBranch}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-muted-foreground text-sm">
                            {new Date(session.startTime).toLocaleTimeString('ja-JP')}
                          </TableCell>
                          <TableCell className="text-right text-muted-foreground">
                            {formatNumber(session.totalInputTokens)}
                          </TableCell>
                          <TableCell className="text-right text-muted-foreground">
                            {formatNumber(session.totalOutputTokens)}
                          </TableCell>
                          <TableCell className="text-right font-semibold">
                            {formatNumber(session.totalTokens)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
                {!dailyLoading && !dailyError && dailyStats && dailyStats.sessions.length === 0 && (
                  <div className="text-center py-8 text-muted-foreground">
                    この日のデータはありません
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Cache Statistics */}
          {(stats.totalCacheCreationTokens > 0 || stats.totalCacheReadTokens > 0) && (
            <Card>
              <CardHeader>
                <CardTitle>キャッシュ統計</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid gap-4 md:grid-cols-2">
                  <div>
                    <p className="text-sm text-muted-foreground">キャッシュ作成トークン</p>
                    <p className="text-2xl font-bold">{formatNumber(stats.totalCacheCreationTokens)}</p>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">キャッシュ読み取りトークン</p>
                    <p className="text-2xl font-bold">{formatNumber(stats.totalCacheReadTokens)}</p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        {/* セッション一覧タブ */}
        <TabsContent value="sessions">
          <SessionListTab sessions={sessions} loading={sessionsLoading} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
