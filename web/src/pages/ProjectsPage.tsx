import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { api, ApiError } from '@/lib/api/client'
import type { Project, ProjectGroup, TotalStats, TimeSeriesResponse } from '@/lib/api/types'
import { Folder, GitBranch, Activity, Zap, AlertCircle, Layers } from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'

export function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([])
  const [groups, setGroups] = useState<ProjectGroup[]>([])
  const [totalStats, setTotalStats] = useState<TotalStats | null>(null)
  const [timeline, setTimeline] = useState<TimeSeriesResponse | null>(null)
  const [period, setPeriod] = useState<'day' | 'week' | 'month'>('day')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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
        console.log('Fetched projects:', projectsRes.projects.length, 'projects')
        console.log('Fetched groups:', groupsRes.groups.length, 'groups')
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
              <CardDescription>全プロジェクトの時系列トークン使用状況</CardDescription>
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
          {loading && (
            <div className="flex items-center justify-center py-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
            </div>
          )}
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
      </Card>

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
              {loading && (
                <div className="flex items-center justify-center py-8">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              )}

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
                            <div className="truncate" title={group.gitRoot}>
                              {group.gitRoot}
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
              {loading && (
                <div className="flex items-center justify-center py-8">
                  <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
                </div>
              )}

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
