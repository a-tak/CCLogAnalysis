import type {
  ProjectsResponse,
  SessionsResponse,
  SessionDetail,
  AnalyzeRequest,
  AnalyzeResponse,
  ErrorResponse,
  ProjectStats,
  TimeSeriesResponse,
  ProjectGroupListResponse,
  ProjectGroupDetail,
  ProjectGroupStats,
  ScanStatus,
} from './types'

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api'

class ApiError extends Error {
  status: number
  error?: ErrorResponse

  constructor(
    message: string,
    status: number,
    error?: ErrorResponse
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.error = error
  }
}

async function fetchApi<T>(
  endpoint: string,
  options?: RequestInit
): Promise<T> {
  const url = `${API_BASE_URL}${endpoint}`

  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    })

    if (!response.ok) {
      const error: ErrorResponse = await response.json().catch(() => ({
        error: 'unknown_error',
        message: 'An unknown error occurred',
      }))

      throw new ApiError(
        error.message || `HTTP ${response.status}`,
        response.status,
        error
      )
    }

    return await response.json()
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }

    throw new ApiError(
      error instanceof Error ? error.message : 'Network error',
      0
    )
  }
}

export const api = {
  // Get all projects
  async getProjects(): Promise<ProjectsResponse> {
    return fetchApi<ProjectsResponse>('/projects')
  },

  // Get sessions (optionally filtered by project)
  async getSessions(projectName?: string): Promise<SessionsResponse> {
    const params = projectName ? `?project=${encodeURIComponent(projectName)}` : ''
    return fetchApi<SessionsResponse>(`/sessions${params}`)
  },

  // Get session detail
  async getSessionDetail(projectName: string, sessionId: string): Promise<SessionDetail> {
    return fetchApi<SessionDetail>(
      `/sessions/${encodeURIComponent(projectName)}/${encodeURIComponent(sessionId)}`
    )
  },

  // Analyze logs
  async analyzeLogs(request?: AnalyzeRequest): Promise<AnalyzeResponse> {
    return fetchApi<AnalyzeResponse>('/analyze', {
      method: 'POST',
      body: JSON.stringify(request || {}),
    })
  },

  // Health check
  async healthCheck(): Promise<{ status: string }> {
    return fetchApi<{ status: string }>('/health')
  },

  // Get project statistics
  async getProjectStats(projectName: string): Promise<ProjectStats> {
    return fetchApi<ProjectStats>(`/projects/${encodeURIComponent(projectName)}/stats`)
  },

  // Get project timeline
  async getProjectTimeline(
    projectName: string,
    period: 'day' | 'week' | 'month' = 'day',
    limit = 30
  ): Promise<TimeSeriesResponse> {
    return fetchApi<TimeSeriesResponse>(
      `/projects/${encodeURIComponent(projectName)}/timeline?period=${period}&limit=${limit}`
    )
  },

  // Get all project groups
  async getProjectGroups(): Promise<ProjectGroupListResponse> {
    return fetchApi<ProjectGroupListResponse>('/groups')
  },

  // Get project group detail
  async getProjectGroup(groupId: number): Promise<ProjectGroupDetail> {
    return fetchApi<ProjectGroupDetail>(`/groups/${groupId}`)
  },

  // Get project group statistics
  async getProjectGroupStats(groupId: number): Promise<ProjectGroupStats> {
    return fetchApi<ProjectGroupStats>(`/groups/${groupId}/stats`)
  },

  // Get scan status
  async getScanStatus(): Promise<ScanStatus> {
    return fetchApi<ScanStatus>('/scan/status')
  },
}

export { ApiError }
