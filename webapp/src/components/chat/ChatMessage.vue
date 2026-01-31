<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ChatMessage, BookmarkSource } from '@/stores/chat'

const props = defineProps<{
  message: ChatMessage
  isStreaming?: boolean
}>()

const { t } = useI18n()

const isUser = computed(() => props.message.role === 'user')
const hasSources = computed(() => props.message.sources && props.message.sources.length > 0)

function navigateToBookmark(source: BookmarkSource) {
  // Open bookmark in new tab
  window.open(`/bookmark/${source.id}`, '_blank')
}
</script>

<template>
  <div :class="['flex', isUser ? 'justify-end' : 'justify-start']">
    <div
      :class="[
        'max-w-[85%] rounded-lg px-4 py-2',
        isUser
          ? 'bg-red-500 text-white'
          : 'bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-gray-100',
      ]"
    >
      <!-- Message content -->
      <div class="whitespace-pre-wrap break-words">
        {{ message.content }}
        <span
          v-if="isStreaming && !message.content"
          class="inline-block w-2 h-4 bg-current animate-pulse"
        ></span>
      </div>

      <!-- Streaming indicator -->
      <div
        v-if="isStreaming && message.content"
        class="mt-1 flex items-center space-x-1 text-xs opacity-70"
      >
        <div class="flex space-x-1">
          <span
            class="w-1.5 h-1.5 bg-current rounded-full animate-bounce"
            style="animation-delay: 0ms"
          ></span>
          <span
            class="w-1.5 h-1.5 bg-current rounded-full animate-bounce"
            style="animation-delay: 150ms"
          ></span>
          <span
            class="w-1.5 h-1.5 bg-current rounded-full animate-bounce"
            style="animation-delay: 300ms"
          ></span>
        </div>
      </div>

      <!-- Sources -->
      <div
        v-if="hasSources && !isUser"
        class="mt-3 pt-3 border-t border-gray-200 dark:border-gray-600"
      >
        <p class="text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">
          {{ t('chat.sources') }}
        </p>
        <div class="space-y-2">
          <button
            v-for="source in message.sources"
            :key="source.id"
            @click="navigateToBookmark(source)"
            class="w-full text-left p-2 bg-white dark:bg-gray-800 rounded border border-gray-200 dark:border-gray-600 hover:border-red-300 dark:hover:border-red-500 transition-colors group"
            data-testid="source-citation"
          >
            <p
              class="text-sm font-medium text-gray-900 dark:text-gray-100 group-hover:text-red-500 truncate"
            >
              {{ source.title || source.url }}
            </p>
            <p
              v-if="source.excerpt"
              class="text-xs text-gray-500 dark:text-gray-400 mt-1 line-clamp-2"
            >
              {{ source.excerpt }}
            </p>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

@keyframes bounce {
  0%,
  100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-25%);
  }
}

.animate-bounce {
  animation: bounce 0.6s infinite;
}
</style>
