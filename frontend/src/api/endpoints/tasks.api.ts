import { http, unwrap } from '../http'
import type { Task, AddTaskRequest } from '../types/task'

export const tasksApi = {
  list: () => unwrap<Task[]>(http.get('/tasks')),
  get: (id: number) => unwrap<Task>(http.get(`/tasks/${id}`)),
  create: (payload: AddTaskRequest) => unwrap<string>(http.post('/tasks', payload)),
  update: (id: number, payload: AddTaskRequest) => unwrap<string>(http.post(`/tasks/${id}`, payload)),
  delete: (id: number) => unwrap<string>(http.delete(`/tasks/${id}`)),
}
