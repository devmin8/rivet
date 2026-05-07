import { http, setCSRFToken } from '~/lib/http'
import type { AuthUser, AuthUserResponse, SignInRequest } from '~/features/auth/types'

export async function signIn(request: SignInRequest): Promise<AuthUser> {
  const response = await http<AuthUserResponse>('/auth/signin', {
    method: 'POST',
    body: JSON.stringify(request),
  })

  setCSRFToken(response.csrf_token)
  return { id: response.id }
}

export async function getCurrentUser(): Promise<AuthUser> {
  const response = await http<AuthUserResponse>('/auth/me')

  setCSRFToken(response.csrf_token)
  return { id: response.id }
}
