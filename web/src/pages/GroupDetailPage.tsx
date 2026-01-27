import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api, ApiError } from '@/lib/api/client'
import type { ProjectGroupDetail, ProjectGroupStats, TimeSeriesResponse, GroupDailyStatsResponse } from '@/lib/api/types'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { LoadingSpinner } from '@/components/ui/loading-spinner'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { Folder, Activity, Zap, TrendingUp, AlertCircle, GitBranch, X } from 'lucide-react'
import { Breadcrumb } from '@/components/navigation/Breadcrumb'

export default function GroupDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [group, setGroup] = useState<ProjectGroupDetail | null>(null)
  const [stats, setStats] = useState<ProjectGroupStats | null>(null)
  const [timeline, setTimeline] = useState<TimeSeriesResponse | null>(null)
  const [period, setPeriod] = useState<'day' | 'week' | 'month'>('day')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Drilldown state
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const [dailyStats, setDailyStats] = useState<GroupDailyStatsResponse | null>(null)
  const [dailyLoading, setDailyLoading] = useState(false)
  const [dailyError, setDailyError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return

    const fetchData = async () => {
      setLoading(true)
      setError(null)

      try {
        const groupId = parseInt(id, 10)
        const [groupData, statsData, timelineData] = await Promise.all([
          api.getProjectGroup(groupId),
          api.getProjectGroupStats(groupId),
          api.getProjectGroupTimeline(groupId, period, 30),
        ])

        setGroup(groupData)
        setStats(statsData)
        setTimeline(timelineData)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'グループ情報の取得に失敗しました')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [id, period])

  // Fetch daily stats when date is selected
  useEffect(() => {
    if (!selectedDate || !id) {
      setDailyStats(null)
      return
    }

    const dateToFetch = selectedDate
    const groupId = parseInt(id, 10)

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
        const stats = await api.getGroupDailyStats(groupId, dateToFetch)
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
  }, [selectedDate, id])

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

  const formatNumber = (num: number) => num.toLocaleString('ja-JP')
  const formatPercent = (num: number) => (num * 100).toFixed(1) + '%'

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

  if (error || !group || !stats) {
    return (
      <div className="container mx-auto py-8">
        <Breadcrumb items={[{ label: 'エラー' }]} />
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">エラー</CardTitle>
          </CardHeader>
          <CardContent>
            <p>{error || 'グループ情報の取得に失敗しました'}</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="container mx-auto py-8">
      {/* Breadcrumb */}
      <Breadcrumb items={[{ label: group.name }]} />

      {/* Group Info */}
      <Card className="mb-6">
        <CardHeader>
          <div className="flex items-center gap-2">
            <GitBranch className="h-6 w-6" />
            <CardTitle className="text-2xl">{group.name}</CardTitle>
          </div>
          <CardDescription>
            <div className="flex flex-col gap-1 mt-2">
              <span className="text-sm">Git Root: {group.gitRoot}</span>
              <span className="text-xs text-muted-foreground">
                作成日: {formatDate(group.createdAt)} | 更新日: {formatDate(group.updatedAt)}
              </span>
            </div>
          </CardDescription>
        </CardHeader>
      </Card>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4 mb-6">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">プロジェクト数</CardTitle>
            <Folder className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{formatNumber(stats.totalProjects)}</div>
          </CardContent>
        </Card>

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
            <div className="text-2xl font-bold">{formatNumber(stats.totalInputTokens + stats.totalOutputTokens)}</div>
            <p className="text-xs text-muted-foreground">
              入力: {formatNumber(stats.totalInputTokens)} / 出力: {formatNumber(stats.totalOutputTokens)}
            </p>
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

      {/* Average Token Usage */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5" />
            平均トークン数
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-3xl font-bold">{formatNumber(stats.avgTokens)}</div>
          <p className="text-sm text-muted-foreground">セッションあたり</p>
        </CardContent>
      </Card>

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
        <Card className="border-primary mb-6">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Zap className="h-5 w-5" />
                  {formatDate(selectedDate)} のプロジェクト別トークン使用量
                </CardTitle>
                <CardDescription>
                  プロジェクトをクリックすると詳細ページに移動します
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
            {!dailyLoading && !dailyError && dailyStats && dailyStats.projects.length > 0 && (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>プロジェクト名</TableHead>
                    <TableHead className="text-right">セッション数</TableHead>
                    <TableHead className="text-right">入力トークン</TableHead>
                    <TableHead className="text-right">出力トークン</TableHead>
                    <TableHead className="text-right">合計トークン</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {dailyStats.projects.map((project) => (
                    <TableRow key={project.projectId} className="cursor-pointer hover:bg-accent">
                      <TableCell className="font-medium">
                        <Link
                          to={`/projects/${encodeURIComponent(project.projectName)}`}
                          className="text-primary hover:underline flex items-center gap-2"
                        >
                          <Folder className="h-4 w-4" />
                          {project.projectName}
                        </Link>
                      </TableCell>
                      <TableCell className="text-right">
                        <Badge variant="secondary">{project.sessionCount}</Badge>
                      </TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {formatNumber(project.totalInputTokens)}
                      </TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {formatNumber(project.totalOutputTokens)}
                      </TableCell>
                      <TableCell className="text-right font-semibold">
                        {formatNumber(project.totalTokens)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
            {!dailyLoading && !dailyError && dailyStats && dailyStats.projects.length === 0 && (
              <div className="text-center py-8 text-muted-foreground">
                この日のデータはありません
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Member Projects */}
      <Card>
        <CardHeader>
          <CardTitle>メンバープロジェクト ({group.projects.length})</CardTitle>
        </CardHeader>
        <CardContent>
          {group.projects.length > 0 ? (
            <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
              {group.projects.map((project) => (
                <Link key={project.name} to={`/projects/${encodeURIComponent(project.name)}`}>
                  <Card className="hover:bg-accent transition-colors cursor-pointer">
                    <CardHeader>
                      <CardTitle className="text-base flex items-center gap-2">
                        <Folder className="h-4 w-4" />
                        {project.name}
                      </CardTitle>
                      <CardDescription className="text-xs truncate">
                        {project.decodedPath}
                      </CardDescription>
                    </CardHeader>
                    <CardContent>
                      <div className="flex items-center gap-2">
                        <Badge variant="secondary">
                          {formatNumber(project.sessionCount)} sessions
                        </Badge>
                      </div>
                    </CardContent>
                  </Card>
                </Link>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              プロジェクトがありません
            </div>
          )}
        </CardContent>
      </Card>

      {/* Cache Statistics */}
      {(stats.totalCacheCreationTokens > 0 || stats.totalCacheReadTokens > 0) && (
        <Card className="mt-6">
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
    </div>
  )
}
