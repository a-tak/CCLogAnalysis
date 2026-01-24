import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { api, ApiError } from '@/lib/api/client'
import type { Project, ProjectGroup } from '@/lib/api/types'
import { Folder, GitBranch } from 'lucide-react'

export function ProjectsPage() {
  const [projects, setProjects] = useState<Project[]>([])
  const [groups, setGroups] = useState<ProjectGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function loadData() {
      try {
        setLoading(true)
        setError(null)
        const [projectsRes, groupsRes] = await Promise.all([
          api.getProjects(),
          api.getProjectGroups(),
        ])
        setProjects(projectsRes.projects)
        setGroups(groupsRes.groups)
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
  }, [])

  const formatDate = (dateStr: string) => {
    const date = new Date(dateStr)
    return date.toLocaleDateString('ja-JP', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">プロジェクト</h1>
        <p className="text-muted-foreground">
          Claude Codeのプロジェクト一覧
        </p>
      </div>

      <Tabs defaultValue="projects" className="w-full">
        <TabsList>
          <TabsTrigger value="projects">全プロジェクト</TabsTrigger>
          <TabsTrigger value="groups">グループ別</TabsTrigger>
        </TabsList>

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
      </Tabs>
    </div>
  )
}
