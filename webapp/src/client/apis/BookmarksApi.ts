/* tslint:disable */
/* eslint-disable */
import * as runtime from '../runtime'
import type { ApiV1ListBookmarksResponse } from '../models/index'
import { ApiV1ListBookmarksResponseFromJSON } from '../models/index'

export interface ApiV1BookmarksListRequest {
  page?: number
  limit?: number
  keyword?: string
  tags?: string
  exclude?: string
}

export class BookmarksApi extends runtime.BaseAPI {
  /**
   * List bookmarks
   */
  async apiV1BookmarksGetRaw(
    requestParameters: ApiV1BookmarksListRequest,
    initOverrides?: RequestInit | runtime.InitOverrideFunction,
  ): Promise<runtime.ApiResponse<ApiV1ListBookmarksResponse>> {
    const queryParameters: any = {}

    if (requestParameters.page !== undefined) {
      queryParameters['page'] = requestParameters.page
    }

    if (requestParameters.limit !== undefined) {
      queryParameters['limit'] = requestParameters.limit
    }

    if (requestParameters.keyword !== undefined) {
      queryParameters['keyword'] = requestParameters.keyword
    }

    if (requestParameters.tags !== undefined) {
      queryParameters['tags'] = requestParameters.tags
    }

    if (requestParameters.exclude !== undefined) {
      queryParameters['exclude'] = requestParameters.exclude
    }

    const headerParameters: runtime.HTTPHeaders = {}

    const response = await this.request(
      {
        path: `/api/v1/bookmarks`,
        method: 'GET',
        headers: headerParameters,
        query: queryParameters,
      },
      initOverrides,
    )

    return new runtime.JSONApiResponse(response, (jsonValue) =>
      ApiV1ListBookmarksResponseFromJSON(jsonValue),
    )
  }

  /**
   * List bookmarks
   */
  async apiV1BookmarksGet(
    requestParameters: ApiV1BookmarksListRequest = {},
    initOverrides?: RequestInit | runtime.InitOverrideFunction,
  ): Promise<ApiV1ListBookmarksResponse> {
    const response = await this.apiV1BookmarksGetRaw(requestParameters, initOverrides)
    return await response.value()
  }
}
