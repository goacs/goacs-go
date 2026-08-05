import { http, unwrap, unwrapPaginated } from '../http'
import { paginatorParamsToQuery, type PaginatorParams } from '../types/paginator'
import type { CPE, CPETemplate, DiagnosticsReport, Flag, Parameter } from '../types/device'
import type { Task, AddTaskRequest } from '../types/task'
import type { LogEntry } from '../types/log'

export interface ParameterRequest {
  name: string
  value: string
  type: string
  flag: Flag
}

export interface AssignTemplateRequest {
  template_id: number
  priority: number
}

export interface DownloadDiagnosticsRequest {
  url?: string
  bytes?: number
  number_of_connections?: number
}

export interface UploadDiagnosticsRequest {
  url?: string
  test_file_length: number
  number_of_connections?: number
}

export const deviceApi = {
  list: (params: PaginatorParams) =>
    unwrapPaginated<CPE>(http.get('/device', { params: paginatorParamsToQuery(params) })),

  get: (uuid: string) => unwrap<CPE>(http.get(`/device/${uuid}`)),
  delete: (uuid: string) => http.delete(`/device/${uuid}`),

  kick: (uuid: string) => unwrap<string>(http.get(`/device/${uuid}/kick`)),
  provision: (uuid: string) => unwrap<string>(http.get(`/device/${uuid}/provision`)),
  lookup: (uuid: string) => unwrap<string>(http.get(`/device/${uuid}/lookup`)),
  clearCache: (uuid: string) => unwrap<string>(http.delete(`/device/${uuid}/cache`)),

  getParameters: (uuid: string, params: PaginatorParams) =>
    unwrapPaginated<Parameter>(http.get(`/device/${uuid}/parameters`, { params: paginatorParamsToQuery(params) })),
  createParameter: (uuid: string, payload: ParameterRequest) =>
    unwrap<string>(http.post(`/device/${uuid}/parameters`, payload)),
  // Go returns a bare 204 body here (no {message,data} envelope, unlike every
  // other endpoint) - success is "the promise didn't throw", not the resolved
  // value, so just normalize the return type to match createParameter's.
  updateParameter: (uuid: string, payload: ParameterRequest) =>
    http.put(`/device/${uuid}/parameters`, payload).then(() => ''),
  deleteParameter: (uuid: string, name: string) =>
    http.delete(`/device/${uuid}/parameters`, { data: { name } }),
  patchParameters: (uuid: string, parameters: ParameterRequest[]) =>
    unwrap<string>(http.patch(`/device/${uuid}/parameters/patch`, { parameters })),

  getCachedParameters: (uuid: string, params: PaginatorParams) =>
    unwrapPaginated<Parameter>(
      http.get(`/device/${uuid}/parameters/cached`, { params: paginatorParamsToQuery(params) }),
    ),
  downloadCachedParametersCsv: (uuid: string) =>
    http.get(`/device/${uuid}/parameters/cached/download`, { responseType: 'blob' }),

  addObject: (uuid: string, name: string, key?: string) =>
    http.post(`/device/${uuid}/addobject`, { name, key }),
  triggerGetParameterValues: (uuid: string) => http.post(`/device/${uuid}/getparametervalues`),

  getTemplates: (uuid: string) => unwrap<CPETemplate[]>(http.get(`/device/${uuid}/templates`)),
  assignTemplate: (uuid: string, payload: AssignTemplateRequest) =>
    unwrap<string>(http.post(`/device/${uuid}/templates`, payload)),
  updateTemplatePriority: (uuid: string, payload: AssignTemplateRequest) =>
    unwrap<string>(http.patch(`/device/${uuid}/templates`, payload)),
  unassignTemplate: (uuid: string, templateId: number) =>
    unwrap<string>(http.delete(`/device/${uuid}/templates/${templateId}`)),

  getTasks: (uuid: string) => unwrap<Task[]>(http.get(`/device/${uuid}/tasks`)),
  addTask: (uuid: string, payload: AddTaskRequest) => unwrap<string>(http.post(`/device/${uuid}/tasks`, payload)),
  getTask: (uuid: string, taskId: number) => unwrap<Task>(http.get(`/device/${uuid}/tasks/${taskId}`)),
  updateTask: (uuid: string, taskId: number, payload: AddTaskRequest) =>
    unwrap<string>(http.put(`/device/${uuid}/tasks/${taskId}`, payload)),
  deleteTask: (uuid: string, taskId: number) => unwrap<string>(http.delete(`/device/${uuid}/tasks/${taskId}`)),

  runDownloadDiagnostics: (uuid: string, payload: DownloadDiagnosticsRequest) =>
    unwrap<string>(http.post(`/device/${uuid}/diagnostics/download`, payload)),
  runUploadDiagnostics: (uuid: string, payload: UploadDiagnosticsRequest) =>
    unwrap<string>(http.post(`/device/${uuid}/diagnostics/upload`, payload)),
  getDiagnosticsReport: (uuid: string) =>
    unwrap<DiagnosticsReport>(http.get(`/device/${uuid}/diagnostics/report`)),

  getLogs: (uuid: string, params: PaginatorParams) =>
    unwrapPaginated<LogEntry>(http.get(`/device/${uuid}/logs`, { params: paginatorParamsToQuery(params) })),
  downloadLogs: (uuid: string, sessionId: string) =>
    http.get(`/device/${uuid}/logs/download`, { params: { session_id: sessionId }, responseType: 'blob' }),
}
