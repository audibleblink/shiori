import { defineStore } from 'pinia'
import { ref } from 'vue'
import { Configuration, BookmarksApi } from '@/client'
import type { ModelBookmarkDTO } from '@/client/models'
import { useAuthStore } from './auth'

export const useBookmarksStore = defineStore('bookmarks', () => {
  const bookmarks = ref<ModelBookmarkDTO[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)
  const total = ref(0)
  const currentPage = ref(1)
  const maxPage = ref(0)

  // API client
  const getBookmarksApi = () => {
    const authStore = useAuthStore()
    const token = authStore.token

    const config = new Configuration({
      basePath: 'http://localhost:8080',
      accessToken: token || undefined,
      headers: token
        ? {
            Authorization: `Bearer ${token}`,
          }
        : undefined,
    })
    return new BookmarksApi(config)
  }

  // Get bookmarks
  const fetchBookmarks = async (page = 1, limit = 30, keyword = '', tags = '', exclude = '') => {
    isLoading.value = true
    error.value = null

    try {
      const api = getBookmarksApi()
      const response = await api.apiV1BookmarksGet({ page, limit, keyword, tags, exclude })

      bookmarks.value = response.bookmarks
      total.value = response.total
      currentPage.value = response.page
      maxPage.value = response.maxPage

      return response
    } catch (err) {
      console.error('Failed to fetch bookmarks:', err)
      if (err instanceof Error && err.message.includes('401')) {
        error.value = 'Authentication error. Please log in again.'
      } else {
        error.value = 'Failed to load bookmarks. Please try again.'
      }
      throw err
    } finally {
      isLoading.value = false
    }
  }

  return {
    bookmarks,
    total,
    currentPage,
    maxPage,
    isLoading,
    error,
    fetchBookmarks,
  }
})
