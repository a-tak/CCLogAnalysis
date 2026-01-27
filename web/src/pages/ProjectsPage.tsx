import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { LoadingSpinner } from '@/components/ui/loading-spinner'
import { api, ApiError } from '@/lib/api/client'
import type { Project, ProjectGroup, TotalStats, TimeSeriesResponse, DailyStatsResponse } from '@/lib/api/types'
import { Folder, GitBranch, Activity, Zap, AlertCircle, Layers, X } from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

export function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([])
  const [groups, setGroups] = useState<ProjectGroup[]>([])
  const [totalStats, setTotalStats] = useState<TotalStats | null>(null)
  const [timeline, setTimeline] = useState<TimeSeriesResponse | null>(null)
  const [period, setPeriod] = useState<'day' | 'week' | 'month'>('day')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // Drilldown state
  const [selectedDate, setSelectedDate] = useState<string | null>(null)
  const [dailyStats, setDailyStats] = useState<DailyStatsResponse | null>(null)
  const [dailyLoading, setDailyLoading] = useState(false)
  const [dailyError, setDailyError] = useState<string | null>(null)

  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true)
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
        } else {
          setError('Failed to load data')
        }
      } finally {
        setLoading(false)
      }
    }

    loadData()
  }, [period])

  // Fetch daily stats when date is selected
  useEffect(() => {
    if (!selectedDate) {
      setDailyStats(null)
      return
    }

    const dateToFetch = selectedDate
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
        const stats = await api.getDailyStats(dateToFetch)
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
  }, [selectedDate])

  const isValidDateFormat = (dateStr: string): boolean => {
    // YYYY-MM-DD 形式の検証
    const datePattern = /^\d{4}-\d{2}-\d{2}$/
    if (!datePattern.test(dateStr)) {
      return false
    }
    // 実際に有効な日付かチェック
    const date = new Date(dateStr)
    return !isNaN(date.getTime())
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

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">ダッシュボード</h1>
        <p className="text-muted-foreground">
          Claude Codeのトークン使用状況
        </p>
      </div>

      {/* Total Summary Cards */}
      {!loading && !error && totalStats && (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">グループ数</CardTitle>
              <Layers className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatNumber(totalStats.totalGroups)}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">総セッション数</CardTitle>
              <Activity className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatNumber(totalStats.totalSessions)}</div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">総トークン数</CardTitle>
              <Zap className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatNumber(totalStats.totalTokens)}</div>
              <p className="text-xs text-muted-foreground">
                入力: {formatNumber(totalStats.totalInputTokens)} / 出力: {formatNumber(totalStats.totalOutputTokens)}
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">エラー率</CardTitle>
              <AlertCircle className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{formatPercent(totalStats.errorRate)}</div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Timeline Chart */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>トークン使用量推移</CardTitle>
              <CardDescription>
                全プロジェクトの時系列トークン使用状況
              </CardDescription>
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
          {loading && <LoadingSpinner />}
          {!loading && timeline && timeline.data.length > 0 ? (
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={[...timeline.data].reverse()}>
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
                <YAxis tickFormatter={(value) => formatNumber(value)} />
                <Tooltip
                  labelFormatter={(value) => {
                    const date = new Date(value as string)
                    return date.toLocaleDateString('ja-JP')
                  }}
                  formatter={(value) => [formatNumber(value as number), '']}
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
          ) : !loading && (
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
                  {formatDate(selectedDate)} のグループ別トークン使用量
                </CardTitle>
                <CardDescription>
                  グループをクリックすると詳細ページに移動します
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
            {!dailyLoading && !dailyError && dailyStats && dailyStats.groups.length > 0 && (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>グループ名</TableHead>
                    <TableHead className="text-right">セッション数</TableHead>
                    <TableHead className="text-right">入力トークン</TableHead>
                    <TableHead className="text-right">出力トークン</TableHead>
                    <TableHead className="text-right">合計トークン</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {dailyStats.groups.map((group) => (
                    <TableRow key={group.groupId} className="cursor-pointer hover:bg-accent">
                      <TableCell className="font-medium">
                        <Link
                          to={`/groups/${group.groupId}`}
                          className="text-primary hover:underline flex items-center gap-2"
                        >
                          <GitBranch className="h-4 w-4" />
                          {group.groupName}
                        </Link>
                      </TableCell>
                      <TableCell className="text-right">
                        <Badge variant="secondary">{group.sessionCount}</Badge>
                      </TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {formatNumber(group.totalInputTokens)}
                      </TableCell>
                      <TableCell className="text-right text-muted-foreground">
                        {formatNumber(group.totalOutputTokens)}
                      </TableCell>
                      <TableCell className="text-right font-semibold">
                        {formatNumber(group.totalTokens)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )}
            {!dailyLoading && !dailyError && dailyStats && dailyStats.groups.length === 0 && (
              <div className="text-center py-8 text-muted-foreground">
                この日のデータはありません
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Tabs: Groups (default) / All Projects */}
      <Tabs defaultValue="groups" className="w-full">
        <TabsList>
          <TabsTrigger value="groups">グループ別</TabsTrigger>
          <TabsTrigger value="projects">全プロジェクト</TabsTrigger>
        </TabsList>

        <TabsContent value="groups" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>プロジェクトグループ</CardTitle>
              <CardDescription>
                {loading && '読み込み中...'}
                {error && `エラー: ${error}`}
                {!loading && !error && `${groups.length} 件のグループ`}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {loading && <LoadingSpinner />}

              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}

              {!loading && !error && groups.length === 0 && (
                <p className="text-sm text-muted-foreground">グループが見つかりません</p>
              )}

              {!loading && !error && groups.length > 0 && (
                <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
                  {groups.map((group) => (
                    <Link key={group.id} to={`/groups/${group.id}`}>
                      <Card className="hover:bg-accent transition-colors cursor-pointer">
                        <CardHeader>
                          <CardTitle className="text-lg flex items-center gap-2">
                            <GitBranch className="h-5 w-5" />
                            {group.name}
                          </CardTitle>
                          <CardDescription className="text-xs">
                            <div className="truncate" title={group.gitRoot || undefined}>
                              {group.gitRoot || '-'}
                            </div>
                            <div className="text-xs text-muted-foreground mt-1">
                              更新: {formatDate(group.updatedAt)}
                            </div>
                          </CardDescription>
                        </CardHeader>
                      </Card>
                    </Link>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="projects" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>プロジェクト一覧</CardTitle>
              <CardDescription>
                {loading && '読み込み中...'}
                {error && `エラー: ${error}`}
                {!loading && !error && `${projects.length} 件のプロジェクト`}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {loading && <LoadingSpinner />}

              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}

              {!loading && !error && projects.length === 0 && (
                <p className="text-sm text-muted-foreground">プロジェクトが見つかりません</p>
              )}

              {!loading && !error && projects.length > 0 && (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>プロジェクト名</TableHead>
                      <TableHead>パス</TableHead>
                      <TableHead className="text-right">セッション数</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {projects.map((project) => (
                      <TableRow key={project.name}>
                        <TableCell className="font-medium">
                          <Link
                            to={`/projects/${encodeURIComponent(project.name)}`}
                            className="text-primary hover:underline flex items-center gap-2"
                          >
                            <Folder className="h-4 w-4" />
                            {project.name}
                          </Link>
                        </TableCell>
                        <TableCell className="text-muted-foreground text-sm">
                          {project.decodedPath}
                        </TableCell>
                        <TableCell className="text-right">
                          <Badge variant="secondary">{project.sessionCount}</Badge>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
