import { cn } from '@/lib/utils'
import type { Message } from '@/lib/api/types'
import { ContentBlock } from './ContentBlock'

interface MessageItemProps {
  message: Message
}

function formatDate(isoString: string): string {
  const date = new Date(isoString)
  return date.toLocaleString()
}

export function MessageItem({ message }: MessageItemProps) {
  const isUser = message.type === 'user'

  return (
    <div className={cn(
      "flex",
      isUser ? "justify-end" : "justify-start"
    )}>
      <div className={cn(
        "max-w-3xl rounded-lg p-4",
        isUser ? "bg-muted" : "bg-primary/10"
      )}>
        {/* Timestamp and model */}
        <div className="text-sm text-muted-foreground mb-2">
          {formatDate(message.timestamp)}
          {message.model && (
            <span className="ml-2 px-2 py-0.5 rounded bg-primary/20 text-xs">
              {message.model}
            </span>
          )}
        </div>

        {/* Content array */}
        <div className="space-y-2">
          {message.content.map((content, index) => (
            <ContentBlock key={index} content={content} />
          ))}
        </div>
      </div>
    </div>
  )
}
