import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

// Types for chat functionality
export interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  sources?: BookmarkSource[]
  timestamp: number
}

export interface BookmarkSource {
  id: number
  title: string
  url: string
  excerpt: string
}

export interface EmbeddingStatusSummary {
  total: number
  completed: number
  pending: number
  processing: number
  failed: number
  skipped: number
}

export interface ChatStatus {
  enabled: boolean
  embeddingStatus?: EmbeddingStatusSummary
}

// Session storage key
const CHAT_HISTORY_KEY = 'shiori_chat_history'

export const useChatStore = defineStore('chat', () => {
  // State
  const messages = ref<ChatMessage[]>([])
  const isOpen = ref(false)
  const isStreaming = ref(false)
  const error = ref<string | null>(null)
  const status = ref<ChatStatus | null>(null)
  const statusLoading = ref(false)

  // Computed
  const hasMessages = computed(() => messages.value.length > 0)
  const isEnabled = computed(() => status.value?.enabled ?? false)
  const embeddingStatus = computed(() => status.value?.embeddingStatus ?? null)

  // Load messages from session storage
  const loadFromStorage = () => {
    try {
      const stored = sessionStorage.getItem(CHAT_HISTORY_KEY)
      if (stored) {
        const parsed = JSON.parse(stored)
        if (Array.isArray(parsed)) {
          messages.value = parsed
        }
      }
    } catch (e) {
      console.error('Failed to load chat history from storage:', e)
    }
  }

  // Save messages to session storage
  const saveToStorage = () => {
    try {
      sessionStorage.setItem(CHAT_HISTORY_KEY, JSON.stringify(messages.value))
    } catch (e) {
      console.error('Failed to save chat history to storage:', e)
    }
  }

  // Actions
  const openPanel = () => {
    isOpen.value = true
    loadFromStorage()
  }

  const closePanel = () => {
    isOpen.value = false
  }

  const togglePanel = () => {
    if (isOpen.value) {
      closePanel()
    } else {
      openPanel()
    }
  }

  const clearHistory = () => {
    messages.value = []
    sessionStorage.removeItem(CHAT_HISTORY_KEY)
  }

  const addUserMessage = (content: string) => {
    const message: ChatMessage = {
      role: 'user',
      content,
      timestamp: Date.now(),
    }
    messages.value.push(message)
    saveToStorage()
  }

  const addAssistantMessage = (content: string, sources?: BookmarkSource[]) => {
    const message: ChatMessage = {
      role: 'assistant',
      content,
      sources,
      timestamp: Date.now(),
    }
    messages.value.push(message)
    saveToStorage()
  }

  // Update the last assistant message (for streaming)
  const updateLastAssistantMessage = (content: string, sources?: BookmarkSource[]) => {
    const lastIndex = messages.value.length - 1
    if (lastIndex >= 0 && messages.value[lastIndex].role === 'assistant') {
      messages.value[lastIndex].content = content
      if (sources) {
        messages.value[lastIndex].sources = sources
      }
    }
  }

  // Start streaming a new assistant message
  const startAssistantStream = () => {
    isStreaming.value = true
    error.value = null
    const message: ChatMessage = {
      role: 'assistant',
      content: '',
      timestamp: Date.now(),
    }
    messages.value.push(message)
  }

  // Complete streaming
  const completeStream = () => {
    isStreaming.value = false
    saveToStorage()
  }

  // Set error
  const setError = (errorMessage: string) => {
    error.value = errorMessage
    isStreaming.value = false
  }

  // Clear error
  const clearError = () => {
    error.value = null
  }

  // Fetch chat status from API
  const fetchStatus = async (token: string, basePath: string = '') => {
    statusLoading.value = true
    try {
      const response = await fetch(`${basePath}/api/v1/chat/status`, {
        headers: {
          Authorization: `Bearer ${token}`,
        },
      })

      if (!response.ok) {
        throw new Error(`Failed to fetch chat status: ${response.status}`)
      }

      const data = await response.json()
      status.value = {
        enabled: data.enabled ?? false,
        embeddingStatus: data.embedding_status,
      }
    } catch (e) {
      console.error('Failed to fetch chat status:', e)
      status.value = { enabled: false }
    } finally {
      statusLoading.value = false
    }
  }

  // Get message history for API (last N messages)
  const getHistoryForApi = (
    limit: number = 10,
  ): { role: 'user' | 'assistant'; content: string }[] => {
    return messages.value.slice(-limit).map((m) => ({
      role: m.role,
      content: m.content,
    }))
  }

  // Initialize store
  const init = () => {
    loadFromStorage()
  }

  return {
    // State
    messages,
    isOpen,
    isStreaming,
    error,
    status,
    statusLoading,

    // Computed
    hasMessages,
    isEnabled,
    embeddingStatus,

    // Actions
    openPanel,
    closePanel,
    togglePanel,
    clearHistory,
    addUserMessage,
    addAssistantMessage,
    updateLastAssistantMessage,
    startAssistantStream,
    completeStream,
    setError,
    clearError,
    fetchStatus,
    getHistoryForApi,
    init,
    loadFromStorage,
    saveToStorage,
  }
})
