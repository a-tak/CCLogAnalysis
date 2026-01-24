import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { api, ApiError } from '@/lib/api/client'
import type { SessionDetail } from '@/lib/api/types'
import { TokenBreakdownChart } from '@/components/charts/TokenBreakdownChart'
import { ModelUsageChart } from '@/components/charts/ModelUsageChart'

function formatDate(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleString()
}

export function SessionDetailPage() {
  const { projectName, sessionId } = useParams<{ projectName: string; sessionId: string }>()
  const [session, setSession] = useState<SessionDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

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

  if (loading) {
    return (
      <div className="space-y-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Session Detail</h1>
          <p className="text-muted-foreground">Loading...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Session Detail</h1>
          <p className="text-destructive">{error}</p>
        </div>
      </div>
    )
  }

  if (!session) {
    return (
      <div className="space-y-4">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Session Detail</h1>
          <p className="text-muted-foreground">Session not found</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Session Detail</h1>
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
          <CardDescription>{session.toolCalls.length} tool calls</CardDescription>
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
              {session.toolCalls.slice(0, 20).map((call, index) => (
                <TableRow key={index}>
                  <TableCell className="text-muted-foreground">
                    {formatDate(call.timestamp)}
                  </TableCell>
                  <TableCell className="font-medium">{call.name}</TableCell>
                  <TableCell className="max-w-md truncate text-muted-foreground">
                    {JSON.stringify(call.input)}
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
          {session.toolCalls.length > 20 && (
            <p className="mt-4 text-sm text-muted-foreground">
              Showing 20 of {session.toolCalls.length} tool calls
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
