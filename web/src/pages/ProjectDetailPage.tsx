import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { api } from '@/lib/api/client'
import type { ProjectStats, TimeSeriesResponse, SessionSummary } from '@/lib/api/types'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { ArrowLeft, Activity, Zap, TrendingUp, AlertCircle } from 'lucide-react'
import { SessionListTab } from '@/components/sessions/SessionListTab'

export default function ProjectDetailPage() {
  const { name } = useParams<{ name: string }>()
  const [stats, setStats] = useState<ProjectStats | null>(null)
  const [timeline, setTimeline] = useState<TimeSeriesResponse | null>(null)
  const [period, setPeriod] = useState<'day' | 'week' | 'month'>('day')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [sessions, setSessions] = useState<SessionSummary[]>([])
  const [sessionsLoading, setSessionsLoading] = useState(false)
  const [activeTab, setActiveTab] = useState('stats')

  useEffect(() => {
    if (!name) return

    const fetchData = async () => {
      setLoading(true)
      setError(null)

      try {
        const [statsData, timelineData] = await Promise.all([
          api.getProjectStats(name),
          api.getProjectTimeline(name, period, 30),
        ])

        setStats(statsData)
        setTimeline(timelineData)
      } catch (err) {
        setError(err instanceof Error ? err.message : '統計の取得に失敗しました')
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [name, period])

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

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
          <p className="text-muted-foreground">読み込み中...</p>
        </div>
      </div>
    )
  }

  if (error || !stats) {
    return (
      <div className="container mx-auto py-8">
        <div className="flex items-center gap-4 mb-6">
          <Button asChild variant="ghost" size="sm">
            <Link to="/projects">
              <ArrowLeft className="h-4 w-4 mr-2" />
              プロジェクト一覧に戻る
            </Link>
          </Button>
        </div>
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
      {/* Header */}
      <div className="flex items-center gap-4 mb-6">
        <Button asChild variant="ghost" size="sm">
          <Link to="/projects">
            <ArrowLeft className="h-4 w-4 mr-2" />
            プロジェクト一覧に戻る
          </Link>
        </Button>
        <h1 className="text-3xl font-bold">{name}</h1>
      </div>

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
          </Card>

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
