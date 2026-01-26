import { useParams, Link } from 'react-router-dom'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { useSessionsPolling } from '@/hooks/useSessionsPolling'

function formatDate(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleString()
}

export function SessionsPage() {
  const { projectName } = useParams<{ projectName: string }>()

  // 15秒ごとにポーリングしてデータを自動更新
  const { sessions, loading, error } = useSessionsPolling(projectName || '')

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Sessions</h1>
        <p className="text-muted-foreground">
          Project: {projectName}
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Session List</CardTitle>
          <CardDescription>
            {loading && 'Loading sessions...'}
            {error && `Error: ${error}`}
            {!loading && !error && `${sessions.length} sessions found`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {loading && (
            <p className="text-sm text-muted-foreground">Loading...</p>
          )}

          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}

          {!loading && !error && sessions.length === 0 && (
            <p className="text-sm text-muted-foreground">No sessions found</p>
          )}

          {!loading && !error && sessions.length > 0 && (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Session ID</TableHead>
                  <TableHead className="hidden md:table-cell">First Message</TableHead>
                  <TableHead>Branch</TableHead>
                  <TableHead>Start Time</TableHead>
                  <TableHead className="text-right">Tokens</TableHead>
                  <TableHead className="text-right">Errors</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {sessions.map((session) => (
                  <TableRow key={session.id}>
                    <TableCell className="font-medium">
                      <Link
                        to={`/projects/${encodeURIComponent(projectName!)}/sessions/${encodeURIComponent(session.id)}`}
                        className="text-primary hover:underline"
                      >
                        {session.id.substring(0, 8)}...
                      </Link>
                    </TableCell>
                    <TableCell className="hidden md:table-cell max-w-md truncate text-muted-foreground">
                      {session.firstUserMessage || '(No message)'}
                    </TableCell>
                    <TableCell>{session.gitBranch}</TableCell>
                    <TableCell className="text-muted-foreground">
                      {formatDate(session.startTime)}
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
          )}
        </CardContent>
      </Card>
    </div>
  )
}
