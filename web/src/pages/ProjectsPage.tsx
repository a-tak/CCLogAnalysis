import { useState, useCallback } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { LoadingSpinner } from '@/components/ui/loading-spinner'
import { api } from '@/lib/api/client'
import type { DailyStatsResponse } from '@/lib/api/types'
import { Folder, GitBranch, Activity, Zap, AlertCircle, Layers, X } from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { useProjectsPolling } from '@/hooks/useProjectsPolling'
import { useDrilldown } from '@/hooks/useDrilldown'
import { DateBadgeSelector } from '@/components/charts/DateBadgeSelector'
import { formatDate, formatNumber, formatPercent } from '@/lib/utils/formatters'

export function ProjectsPage() {
  const [period, setPeriod] = useState<'day' | 'week' | 'month'>('day')

  // 15秒ごとにポーリングしてデータを自動更新
  const { projects, groups, totalStats, timeline, loading, error } = useProjectsPolling(period)

  // Drilldown hook - fetchData をメモ化して無限リクエストを防止
  const fetchDailyStats = useCallback(
    (date: string) => api.getDailyStats(date),
    []
  )

  const drilldown = useDrilldown<DailyStatsResponse>({
    fetchData: fetchDailyStats,
  })

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
        <DateBadgeSelector
          timeSeriesData={timeline?.data || []}
          selectedDate={drilldown.selectedDate}
          onDateClick={drilldown.handleDateClick}
          period={period}
          loading={loading}
        />
      </Card>

      {/* Drilldown Panel */}
      {drilldown.selectedDate && (
        <Card className="border-primary">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Zap className="h-5 w-5" />
                  {formatDate(drilldown.selectedDate)} のグループ別トークン使用量
                </CardTitle>
                <CardDescription>
                  グループをクリックすると詳細ページに移動します
                </CardDescription>
              </div>
              <Button variant="ghost" size="icon" onClick={drilldown.close}>
                <X className="h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            {drilldown.loading && <LoadingSpinner />}
            {!drilldown.loading && drilldown.error && (
              <div className="flex items-center justify-center py-8">
                <p className="text-sm text-destructive">{drilldown.error}</p>
              </div>
            )}
            {!drilldown.loading && !drilldown.error && drilldown.data && drilldown.data.groups.length > 0 && (
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
                  {drilldown.data.groups.map((group) => (
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
            {!drilldown.loading && !drilldown.error && drilldown.data && drilldown.data.groups.length === 0 && (
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
                            {group.displayName}
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
                            {project.displayName}
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
