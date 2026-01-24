import type { Message } from '@/lib/api/types'
import { MessageItem } from './MessageItem'

interface ConversationHistoryProps {
  messages: Message[]
}

export function ConversationHistory({ messages }: ConversationHistoryProps) {
  if (messages.length === 0) {
    return <p className="text-muted-foreground">No messages found</p>
  }

  return (
    <div className="space-y-4">
      {messages.map((message, index) => (
        <MessageItem key={index} message={message} />
      ))}
    </div>
  )
}
