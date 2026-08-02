import { http, unwrap, unwrapPaginated } from '../http'
import { paginatorParamsToQuery, type PaginatorParams } from '../types/paginator'
import type { User, UserUpdateRequest } from '../types/user'

export interface CreateUserRequest {
  username: string
  password: string
  email: string
}

export const userApi = {
  list: (params: PaginatorParams) =>
    unwrapPaginated<User>(http.get('/settings/user', { params: paginatorParamsToQuery(params) })),
  get: (uuid: string) => unwrap<User>(http.get(`/settings/user/${uuid}`)),
  // Go's UserCreate writes the bare User object (no {message,data} envelope,
  // unlike every other endpoint), so this reads response.data directly.
  create: (payload: CreateUserRequest) => http.post<User>('/settings/user', payload).then((r) => r.data),
  update: (uuid: string, payload: UserUpdateRequest) => unwrap<User>(http.put(`/settings/user/${uuid}`, payload)),
  delete: (uuid: string) => unwrap<string>(http.delete(`/settings/user/${uuid}`)),
}
