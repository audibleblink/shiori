/**
 * Chat API service for SSE streaming
 *
 * Handles Server-Sent Events (SSE) for the chat endpoint
 */

export interface ChatStreamEvents {
  onChunk: (content: string) => void
  onSources: (sources: BookmarkSource[]) => void
  onDone: () => void
  onError: (error: string) => void
}

export interface BookmarkSource {
  id: number
  title: string
  url: string
  excerpt: string
}

export interface ChatHistory {
  role: 'user' | 'assistant'
  content: string
}

/**
 * Start a streaming chat request
 *
 * @param message - The user's message
 * @param history - Previous conversation history
 * @param token - Authentication token
 * @param events - Event handlers for streaming events
 * @param basePath - API base path (optional)
 * @returns AbortController to cancel the stream
 */
export function streamChat(
  message: string,
  history: ChatHistory[],
  token: string,
  events: ChatStreamEvents,
  basePath: string = '',
): AbortController {
  const controller = new AbortController()

  const fetchStream = async () => {
    try {
      const response = await fetch(`${basePath}/api/v1/chat`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
          Accept: 'text/event-stream',
        },
        body: JSON.stringify({ message, history }),
        signal: controller.signal,
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        const errorMessage =
          errorData.error || errorData.message || `HTTP ${response.status}: ${response.statusText}`
        events.onError(errorMessage)
        return
      }

      if (!response.body) {
        events.onError('Response body is not available')
        return
      }

      const reader = response.body.getReader()
      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()

        if (done) {
          // Process any remaining data in buffer
          if (buffer.trim()) {
            processSSEBuffer(buffer, events)
          }
          break
        }

        buffer += decoder.decode(value, { stream: true })

        // Process complete SSE events
        const lines = buffer.split('\n')
        buffer = lines.pop() || '' // Keep incomplete line in buffer

        let currentEvent = ''
        let currentData = ''

        for (const line of lines) {
          if (line.startsWith('event: ')) {
            // If we have a previous event pending, process it
            if (currentEvent && currentData) {
              processSSEEvent(currentEvent, currentData, events)
            }
            currentEvent = line.slice(7).trim()
            currentData = ''
          } else if (line.startsWith('data: ')) {
            currentData = line.slice(6)
          } else if (line === '' && currentEvent && currentData) {
            // Empty line signals end of event
            processSSEEvent(currentEvent, currentData, events)
            currentEvent = ''
            currentData = ''
          }
        }

        // Keep pending event data for next iteration
        if (currentEvent && currentData) {
          buffer = `event: ${currentEvent}\ndata: ${currentData}\n${buffer}`
        }
      }
    } catch (error) {
      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          // Stream was intentionally cancelled
          return
        }
        events.onError(error.message)
      } else {
        events.onError('An unknown error occurred')
      }
    }
  }

  fetchStream()
  return controller
}

/**
 * Process a complete SSE event
 */
function processSSEEvent(event: string, data: string, events: ChatStreamEvents): void {
  try {
    switch (event) {
      case 'chunk': {
        const parsed = JSON.parse(data)
        if (parsed.content) {
          events.onChunk(parsed.content)
        }
        break
      }
      case 'sources': {
        const parsed = JSON.parse(data)
        if (Array.isArray(parsed.sources)) {
          events.onSources(parsed.sources)
        }
        break
      }
      case 'done': {
        events.onDone()
        break
      }
      case 'error': {
        const parsed = JSON.parse(data)
        events.onError(parsed.error || parsed.message || 'Unknown error')
        break
      }
      default:
        // Unknown event type, ignore
        break
    }
  } catch (e) {
    console.error('Failed to parse SSE event:', event, data, e)
  }
}

/**
 * Process any remaining buffer content
 */
function processSSEBuffer(buffer: string, events: ChatStreamEvents): void {
  const lines = buffer.split('\n')
  let currentEvent = ''
  let currentData = ''

  for (const line of lines) {
    if (line.startsWith('event: ')) {
      if (currentEvent && currentData) {
        processSSEEvent(currentEvent, currentData, events)
      }
      currentEvent = line.slice(7).trim()
      currentData = ''
    } else if (line.startsWith('data: ')) {
      currentData = line.slice(6)
    } else if (line === '' && currentEvent && currentData) {
      processSSEEvent(currentEvent, currentData, events)
      currentEvent = ''
      currentData = ''
    }
  }

  // Process final event if exists
  if (currentEvent && currentData) {
    processSSEEvent(currentEvent, currentData, events)
  }
}

/**
 * Fetch chat status
 *
 * @param token - Authentication token
 * @param basePath - API base path (optional)
 * @returns Chat status or null on error
 */
export async function fetchChatStatus(
  token: string,
  basePath: string = '',
): Promise<ChatStatusResponse | null> {
  try {
    const response = await fetch(`${basePath}/api/v1/chat/status`, {
      headers: {
        Authorization: `Bearer ${token}`,
      },
    })

    if (!response.ok) {
      return null
    }

    return await response.json()
  } catch (error) {
    console.error('Failed to fetch chat status:', error)
    return null
  }
}

export interface ChatStatusResponse {
  enabled: boolean
  embedding_status?: {
    total: number
    completed: number
    pending: number
    processing: number
    failed: number
    skipped: number
  }
}
