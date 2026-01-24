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
  firstUserMessage: string
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

// Content block types for message content
export interface ContentBlock {
  type: 'text' | 'tool_use' | 'tool_result'

  // text type
  text?: string

  // tool_use type
  id?: string
  name?: string
  input?: Record<string, unknown>

  // tool_result type
  tool_use_id?: string
  content?: string
  is_error?: boolean
}

export interface Message {
  type: 'user' | 'assistant'
  timestamp: string
  model?: string
  content: ContentBlock[]
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

// Project Statistics
export interface ProjectStats {
  totalSessions: number
  totalInputTokens: number
  totalOutputTokens: number
  totalCacheCreationTokens: number
  totalCacheReadTokens: number
  totalTokens: number
  avgTokens: number
  firstSession: string
  lastSession: string
  errorRate: number
}

export interface BranchStats {
  branch: string
  sessionCount: number
  totalInputTokens: number
  totalOutputTokens: number
  totalCacheCreationTokens: number
  totalCacheReadTokens: number
  totalTokens: number
  lastActivity: string
}

export interface TimeSeriesDataPoint {
  periodStart: string
  periodEnd: string
  sessionCount: number
  totalInputTokens: number
  totalOutputTokens: number
  totalCacheCreationTokens: number
  totalCacheReadTokens: number
  totalTokens: number
}

export interface TimeSeriesResponse {
  period: 'day' | 'week' | 'month'
  data: TimeSeriesDataPoint[]
}

// Project Groups
export interface ProjectGroup {
  id: number
  name: string
  gitRoot: string
  createdAt: string
  updatedAt: string
}

export interface ProjectGroupListResponse {
  groups: ProjectGroup[]
}

export interface ProjectGroupDetail {
  id: number
  name: string
  gitRoot: string
  createdAt: string
  updatedAt: string
  projects: Project[]
}

export interface ProjectGroupStats {
  totalProjects: number
  totalSessions: number
  totalInputTokens: number
  totalOutputTokens: number
  totalCacheCreationTokens: number
  totalCacheReadTokens: number
  avgTokens: number
  firstSession: string
  lastSession: string
  errorRate: number
}
