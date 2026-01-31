<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>

<template>
  <div class="flex-1 flex flex-col items-center justify-center p-6 text-center">
    <div
      class="w-16 h-16 rounded-full bg-gray-100 dark:bg-gray-700 flex items-center justify-center mb-4"
    >
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="h-8 w-8 text-gray-400 dark:text-gray-500"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M13 10V3L4 14h7v7l9-11h-7z"
        />
      </svg>
    </div>

    <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">
      {{ t('chat.disabled_title') }}
    </h3>

    <p class="text-gray-600 dark:text-gray-400 mb-6 max-w-xs">
      {{ t('chat.disabled_description') }}
    </p>

    <div class="bg-gray-50 dark:bg-gray-700/50 rounded-lg p-4 text-left w-full max-w-xs">
      <p class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-3">
        {{ t('chat.setup_requirements') }}
      </p>
      <ul class="space-y-2 text-sm text-gray-600 dark:text-gray-400">
        <li class="flex items-start">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-5 w-5 mr-2 flex-shrink-0 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <span>{{ t('chat.requirement_postgres') }}</span>
        </li>
        <li class="flex items-start">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-5 w-5 mr-2 flex-shrink-0 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <span>{{ t('chat.requirement_ollama') }}</span>
        </li>
        <li class="flex items-start">
          <svg
            xmlns="http://www.w3.org/2000/svg"
            class="h-5 w-5 mr-2 flex-shrink-0 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <span>{{ t('chat.requirement_anthropic') }}</span>
        </li>
      </ul>
    </div>

    <a
      href="https://github.com/go-shiori/shiori/blob/master/docs/ai-chat.md"
      target="_blank"
      rel="noopener noreferrer"
      class="mt-6 inline-flex items-center text-red-500 hover:text-red-600 text-sm font-medium"
    >
      {{ t('chat.view_documentation') }}
      <svg
        xmlns="http://www.w3.org/2000/svg"
        class="h-4 w-4 ml-1"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
        />
      </svg>
    </a>
  </div>
</template>
