/* tslint:disable */
/* eslint-disable */
import type { ModelBookmarkDTO } from './ModelBookmarkDTO'
import { ModelBookmarkDTOFromJSON, ModelBookmarkDTOToJSON } from './ModelBookmarkDTO'

export interface ApiV1ListBookmarksResponse {
  bookmarks: Array<ModelBookmarkDTO>
  page: number
  maxPage: number
  total: number
}

export function ApiV1ListBookmarksResponseFromJSON(json: any): ApiV1ListBookmarksResponse {
  return ApiV1ListBookmarksResponseFromJSONTyped(json, false)
}

export function ApiV1ListBookmarksResponseFromJSONTyped(
  json: any,
  ignoreDiscriminator: boolean,
): ApiV1ListBookmarksResponse {
  if (json === undefined || json === null) {
    return json
  }
  return {
    bookmarks: (json['bookmarks'] as Array<any>).map(ModelBookmarkDTOFromJSON),
    page: json['page'],
    maxPage: json['maxPage'],
    total: json['total'],
  }
}

export function ApiV1ListBookmarksResponseToJSON(value?: ApiV1ListBookmarksResponse | null): any {
  if (value === undefined) {
    return undefined
  }
  if (value === null) {
    return null
  }
  return {
    bookmarks: (value.bookmarks as Array<any>).map(ModelBookmarkDTOToJSON),
    page: value.page,
    maxPage: value.maxPage,
    total: value.total,
  }
}
