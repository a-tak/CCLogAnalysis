import type { ContentBlock as ContentBlockType } from '@/lib/api/types'
import { ToolUseBlock } from './ToolUseBlock'
import { ToolResultBlock } from './ToolResultBlock'

interface ContentBlockProps {
  content: ContentBlockType
}

export function ContentBlock({ content }: ContentBlockProps) {
  switch (content.type) {
    case 'text':
      return (
        <div className="whitespace-pre-wrap break-words">
          {content.text}
        </div>
      )

    case 'tool_use':
      return <ToolUseBlock toolUse={content} />

    case 'tool_result':
      return <ToolResultBlock toolResult={content} />

    default:
      console.warn('Unknown content type:', content)
      return <div className="text-muted-foreground">Unknown content type</div>
  }
}
