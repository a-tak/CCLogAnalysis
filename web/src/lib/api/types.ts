// API response types based on API設計.md

export interface Project {
  name: string
  decodedPath: string
  sessionCount: number
}

export interface ProjectsResponse {
  projects: Project[]
}

export interface SessionSummary {
  id: string
  projectName: string
  gitBranch: string
  startTime: string
  endTime: string
  totalTokens: number
  errorCount: number
}

export interface SessionsResponse {
  sessions: SessionSummary[]
}

export interface TokenSummary {
  inputTokens: number
  outputTokens: number
  cacheCreationInputTokens: number
  cacheReadInputTokens: number
  totalTokens: number
}

export interface ModelUsage {
  model: string
  tokens: TokenSummary
}

export interface ToolCall {
  timestamp: string
  name: string
  input: Record<string, unknown>
  isError: boolean
}

export interface Message {
  type: 'user' | 'assistant'
  timestamp: string
  model?: string
  content: unknown[]
}

export interface SessionDetail {
  id: string
  projectName: string
  projectPath: string
  gitBranch: string
  startTime: string
  endTime: string
  duration: string
  totalTokens: TokenSummary
  modelUsage: ModelUsage[]
  toolCalls: ToolCall[]
  messages: Message[]
  errorCount: number
}

export interface AnalyzeRequest {
  projectNames?: string[]
  force?: boolean
}

export interface AnalyzeResponse {
  status: 'completed' | 'error'
  sessionsFound: number
  sessionsParsed: number
  errorCount: number
  message?: string
}

export interface ErrorResponse {
  error: string
  message: string
}
