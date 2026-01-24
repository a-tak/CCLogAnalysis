import { useState } from 'react'
import type { ContentBlock } from '@/lib/api/types'

interface ToolUseBlockProps {
  toolUse: ContentBlock
}

export function ToolUseBlock({ toolUse }: ToolUseBlockProps) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="border border-primary/20 rounded-md p-3 bg-primary/5">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 text-sm font-medium text-primary"
      >
        <span>{expanded ? '▼' : '▶'}</span>
        Tool: {toolUse.name}
      </button>

      {expanded && (
        <div className="mt-2 text-sm">
          <div className="font-medium text-muted-foreground">Input:</div>
          <pre className="mt-1 p-2 bg-muted rounded overflow-x-auto">
            <code>{JSON.stringify(toolUse.input, null, 2)}</code>
          </pre>
        </div>
      )}
    </div>
  )
}
