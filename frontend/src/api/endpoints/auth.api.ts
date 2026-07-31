import { http, unwrap } from '../http'
import type { User } from '../types/user'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  user: User
  token: string
}

export const authApi = {
  login: (payload: LoginRequest) => unwrap<LoginResponse>(http.post('/auth/login', payload)),
  logout: () => unwrap<string>(http.post('/auth/logout')),
  refresh: () => unwrap<LoginResponse>(http.post('/auth/refresh')),
}
