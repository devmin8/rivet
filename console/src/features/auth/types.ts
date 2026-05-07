export interface AuthUser {
  id: string
}

export interface AuthUserResponse {
  id: string
  csrf_token: string
}

export interface SignInRequest {
  username: string
  password: string
}
