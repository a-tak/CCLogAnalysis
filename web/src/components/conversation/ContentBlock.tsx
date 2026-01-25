import type { ContentBlock as ContentBlockType } from '@/lib/api/types'
import { ToolUseBlock } from './ToolUseBlock'
import { ToolResultBlock } from './ToolResultBlock'

interface ContentBlockProps {
  content: ContentBlockType
}

export function ContentBlock({ content }: ContentBlockProps) {
  if (!content || typeof content !== 'object') {
    return null
  }

  switch (content.type) {
    case 'text':
      return (
        <div className="whitespace-pre-wrap break-words">
          {content.text || ''}
        </div>
      )

    case 'thinking':
      // thinkingタイプはテキストがないので何も表示しない
      return null

    case 'tool_use':
      return <ToolUseBlock toolUse={content} />

    case 'tool_result':
      return <ToolResultBlock toolResult={content} />

    default:
      return null
  }
}
