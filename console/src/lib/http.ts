import { env } from '~/lib/env'
import { ApiError, type ApiErrorBody } from '~/lib/errors'

let csrfToken = ''

export function setCSRFToken(token: string) {
  csrfToken = token
}

export function clearCSRFToken() {
  csrfToken = ''
}

export async function http<TResponse>(path: string, init: RequestInit = {}): Promise<TResponse> {
  const method = init.method ?? 'GET'
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')

  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  if (requiresCSRF(method) && csrfToken !== '') {
    headers.set('X-CSRF-Token', csrfToken)
  }

  const response = await fetch(apiURL(path), {
    ...init,
    cache: 'no-store',
    credentials: 'include',
    headers,
    redirect: 'error',
  })

  if (!response.ok) {
    if (response.status === 401) {
      clearCSRFToken()
    }

    throw new ApiError(response.status, await errorBody(response))
  }

  if (response.status === 204) {
    return undefined as TResponse
  }

  return (await response.json()) as TResponse
}

function apiURL(path: string): string {
  const normalizedPath = path.startsWith('/') ? path.slice(1) : path
  return `${env.rivetApiURL}/${normalizedPath}`
}

function requiresCSRF(method: string): boolean {
  const normalizedMethod = method.toUpperCase()
  return (
    normalizedMethod !== 'GET' &&
    normalizedMethod !== 'HEAD' &&
    normalizedMethod !== 'OPTIONS' &&
    normalizedMethod !== 'TRACE'
  )
}

async function errorBody(response: Response): Promise<ApiErrorBody> {
  if (response.headers.get('content-type')?.includes('application/json')) {
    return (await response.json()) as ApiErrorBody
  }

  return {
    error: 'request_failed',
    message: 'The request could not be completed.',
  }
}
