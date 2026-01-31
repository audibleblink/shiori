<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useChatStore } from '@/stores/chat'
import { useAuthStore } from '@/stores/auth'
import { streamChat } from '@/services/chatApi'
import ChatMessage from './ChatMessage.vue'
import ChatDisabled from './ChatDisabled.vue'

defineProps<{
  isMobile: boolean
}>()

const { t } = useI18n()
const chatStore = useChatStore()
const authStore = useAuthStore()

const inputMessage = ref('')
const messagesContainer = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const abortController = ref<AbortController | null>(null)

// Computed
const canSend = computed(() => {
  return inputMessage.value.trim().length > 0 && !chatStore.isStreaming
})

const showDisabledState = computed(() => {
  return chatStore.status !== null && !chatStore.isEnabled
})

// Auto-scroll to bottom when messages change
watch(
  () => chatStore.messages.length,
  async () => {
    await nextTick()
    scrollToBottom()
  },
)

// Also scroll when streaming content updates
watch(
  () => chatStore.messages[chatStore.messages.length - 1]?.content,
  async () => {
    await nextTick()
    scrollToBottom()
  },
)

// Focus input when panel opens
watch(
  () => chatStore.isOpen,
  async (isOpen) => {
    if (isOpen) {
      await nextTick()
      inputRef.value?.focus()
    }
  },
)

// Fetch status on mount
onMounted(async () => {
  if (authStore.token) {
    await chatStore.fetchStatus(authStore.token)
  }
  chatStore.loadFromStorage()
})

// Cleanup on unmount
onUnmounted(() => {
  if (abortController.value) {
    abortController.value.abort()
  }
})

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

async function sendMessage() {
  if (!canSend.value || !authStore.token) return

  const message = inputMessage.value.trim()
  inputMessage.value = ''

  // Add user message
  chatStore.addUserMessage(message)

  // Start streaming assistant response
  chatStore.startAssistantStream()

  let accumulatedContent = ''

  // Get history for API (exclude the message we just added, as we send it separately)
  const history = chatStore.getHistoryForApi(10).slice(0, -1)

  abortController.value = streamChat(message, history, authStore.token, {
    onChunk: (content: string) => {
      accumulatedContent += content
      chatStore.updateLastAssistantMessage(accumulatedContent)
    },
    onSources: (sources) => {
      chatStore.updateLastAssistantMessage(accumulatedContent, sources)
    },
    onDone: () => {
      chatStore.completeStream()
      abortController.value = null
    },
    onError: (error: string) => {
      chatStore.setError(error)
      // Remove the empty assistant message if there was an error
      if (accumulatedContent === '') {
        chatStore.messages.pop()
      }
      abortController.value = null
    },
  })
}

function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

function cancelStream() {
  if (abortController.value) {
    abortController.value.abort()
    abortController.value = null
    chatStore.completeStream()
  }
}

function newConversation() {
  if (abortController.value) {
    abortController.value.abort()
    abortController.value = null
  }
  chatStore.clearHistory()
  chatStore.clearError()
}

function closePanel() {
  chatStore.closePanel()
}
</script>

<template>
  <!-- Backdrop for mobile -->
  <Transition name="fade">
    <div
      v-if="chatStore.isOpen && isMobile"
      class="fixed inset-0 bg-black/50 z-40"
      @click="closePanel"
    ></div>
  </Transition>

  <!-- Chat Panel -->
  <Transition :name="isMobile ? 'slide-up' : 'slide-right'">
    <div
      v-if="chatStore.isOpen"
      :class="[
        'fixed z-50 bg-white dark:bg-gray-800 shadow-xl flex flex-col',
        isMobile
          ? 'inset-x-0 bottom-0 h-[80vh] rounded-t-2xl'
          : 'top-0 right-0 h-full w-[400px] border-l border-gray-200 dark:border-gray-700',
      ]"
      data-testid="chat-panel"
    >
      <!-- Header -->
      <div
        class="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700"
      >
        <div class="flex items-center space-x-2">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-5 w-5 text-red-500"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
            />
          </svg>
          <h2 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {{ t('chat.title') }}
          </h2>
        </div>
        <div class="flex items-center space-x-2">
          <button
            v-if="chatStore.hasMessages"
            @click="newConversation"
            class="text-gray-500 dark:text-gray-400 hover:text-red-500 p-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            :title="t('chat.new_conversation')"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M12 4v16m8-8H4"
              />
            </svg>
          </button>
          <button
            @click="closePanel"
            class="text-gray-500 dark:text-gray-400 hover:text-red-500 p-1.5 rounded-md hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors"
            :title="t('common.close')"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              class="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
      </div>

      <!-- Content -->
      <div class="flex-1 flex flex-col min-h-0">
        <!-- Disabled State -->
        <ChatDisabled v-if="showDisabledState" />

        <!-- Chat Interface -->
        <template v-else>
          <!-- Messages -->
          <div ref="messagesContainer" class="flex-1 overflow-y-auto p-4 space-y-4">
            <!-- Empty state -->
            <div
              v-if="!chatStore.hasMessages && !chatStore.statusLoading"
              class="h-full flex flex-col items-center justify-center text-gray-500 dark:text-gray-400 text-center px-6"
            >
              <svg
                xmlns="http://www.w3.org/2000/svg"
                class="h-12 w-12 mb-4 text-gray-300 dark:text-gray-600"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  stroke-width="1.5"
                  d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z"
                />
              </svg>
              <p class="font-medium mb-2">{{ t('chat.empty_title') }}</p>
              <p class="text-sm">{{ t('chat.empty_description') }}</p>
            </div>

            <!-- Loading status -->
            <div
              v-else-if="chatStore.statusLoading"
              class="h-full flex items-center justify-center"
            >
              <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-red-500"></div>
            </div>

            <!-- Messages list -->
            <template v-else>
              <ChatMessage
                v-for="(message, index) in chatStore.messages"
                :key="index"
                :message="message"
                :is-streaming="chatStore.isStreaming && index === chatStore.messages.length - 1"
              />
            </template>
          </div>

          <!-- Error message -->
          <div
            v-if="chatStore.error"
            class="mx-4 mb-2 px-3 py-2 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm rounded-lg"
          >
            {{ chatStore.error }}
          </div>

          <!-- Input -->
          <div class="p-4 border-t border-gray-200 dark:border-gray-700">
            <div class="flex items-end space-x-2">
              <input
                ref="inputRef"
                v-model="inputMessage"
                @keydown="handleKeydown"
                :placeholder="t('chat.input_placeholder')"
                :disabled="chatStore.isStreaming || showDisabledState"
                class="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100 rounded-lg focus:outline-none focus:ring-2 focus:ring-red-500 disabled:opacity-50 disabled:cursor-not-allowed"
              />
              <button
                v-if="chatStore.isStreaming"
                @click="cancelStream"
                class="p-2 bg-gray-500 text-white rounded-lg hover:bg-gray-600 transition-colors"
                :title="t('chat.stop')"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"
                  />
                </svg>
              </button>
              <button
                v-else
                @click="sendMessage"
                :disabled="!canSend"
                class="p-2 bg-red-500 text-white rounded-lg hover:bg-red-600 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                :title="t('chat.send')"
              >
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-5 w-5"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"
                  />
                </svg>
              </button>
            </div>

            <!-- Embedding status hint -->
            <div
              v-if="chatStore.embeddingStatus && chatStore.embeddingStatus.pending > 0"
              class="mt-2 text-xs text-gray-500 dark:text-gray-400"
            >
              {{
                t('chat.embedding_progress', {
                  pending: chatStore.embeddingStatus.pending,
                  total: chatStore.embeddingStatus.total,
                })
              }}
            </div>
          </div>
        </template>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
/* Slide from right animation (desktop) */
.slide-right-enter-active,
.slide-right-leave-active {
  transition: transform 0.3s ease;
}

.slide-right-enter-from,
.slide-right-leave-to {
  transform: translateX(100%);
}

/* Slide from bottom animation (mobile) */
.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.3s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(100%);
}

/* Fade animation for backdrop */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
