export interface User {
  uuid: string
  username: string
  email: string
  status: number
}

export interface UserUpdateRequest {
  username: string
  email: string
  password?: string
}
