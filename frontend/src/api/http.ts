import axios, { AxiosError } from 'axios'
import type { AxiosInstance } from 'axios'
import type { ApiEnvelope, ValidationErrors } from './types/apiEnvelope'
import type { PaginatorResponse } from './types/paginator'

// Holds the JWT outside of Pinia so this module never has to import the auth
// store (store -> http is a one-way dependency; the reverse would cycle).
// The auth store writes to this on login/logout/restore.
export const authState: { token: string | null } = { token: null }

let unauthorizedHandler: (() => void) | null = null
export function setUnauthorizedHandler(handler: () => void): void {
  unauthorizedHandler = handler
}

export class ApiValidationError extends Error {
  fields: ValidationErrors

  constructor(message: string, fields: ValidationErrors) {
    super(message)
    this.name = 'ApiValidationError'
    this.fields = fields
  }
}

export const http: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
})

http.interceptors.request.use((config) => {
  if (authState.token) {
    config.headers.Authorization = `Bearer ${authState.token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiEnvelope<unknown>>) => {
    if (error.response?.status === 401) {
      unauthorizedHandler?.()
    }

    if (error.response?.status === 422) {
      const body = error.response.data
      return Promise.reject(
        new ApiValidationError(body?.message ?? 'Validation error', (body?.data as ValidationErrors) ?? {}),
      )
    }

    return Promise.reject(error)
  },
)

// unwrap<T> strips the {message, data} envelope down to just the payload.
export async function unwrap<T>(promise: Promise<{ data: ApiEnvelope<T> }>): Promise<T> {
  const response = await promise
  return response.data.data
}

// unwrapPaginated<T> keeps the whole PaginatorResponse - page/total/etc. are
// as meaningful to callers (server tables) as the data array itself.
export async function unwrapPaginated<T>(
  promise: Promise<{ data: PaginatorResponse<T> }>,
): Promise<PaginatorResponse<T>> {
  const response = await promise
  return response.data
}
