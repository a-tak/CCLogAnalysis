import { useState } from 'react'
import { cn } from '@/lib/utils'
import type { ContentBlock } from '@/lib/api/types'

interface ToolResultBlockProps {
  toolResult: ContentBlock
}

export function ToolResultBlock({ toolResult }: ToolResultBlockProps) {
  const [expanded, setExpanded] = useState(toolResult.is_error || false)
  const isError = toolResult.is_error || false

  return (
    <div className={cn(
      "border rounded-md p-3",
      isError ? "border-destructive bg-destructive/10" : "border-muted bg-muted/50"
    )}>
      <button
        onClick={() => setExpanded(!expanded)}
        className={cn(
          "flex items-center gap-2 text-sm font-medium",
          isError ? "text-destructive" : "text-muted-foreground"
        )}
      >
        <span>{expanded ? '▼' : '▶'}</span>
        Tool Result {isError && '(Error)'}
      </button>

      {expanded && (
        <div className="mt-2 text-sm">
          <pre className={cn(
            "mt-1 p-2 rounded overflow-x-auto max-h-96",
            "whitespace-pre-wrap break-words font-mono",
            isError ? "bg-destructive/20" : "bg-muted"
          )}>
            <code>{toolResult.content}</code>
          </pre>
        </div>
      )}
    </div>
  )
}
