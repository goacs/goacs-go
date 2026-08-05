import { http, unwrap, unwrapPaginated } from '../http'
import { paginatorParamsToQuery, type PaginatorParams } from '../types/paginator'
import type { Template, TemplateParameter } from '../types/template'
import type { Flag } from '../types/device'

export interface TemplateListItem extends Template {
  parameter_count: number
}

export interface TemplateParameterRequest {
  template_id: number
  name: string
  value: string
  type: string
  flag: Flag
}

export const templateApi = {
  list: (params: PaginatorParams) =>
    unwrapPaginated<TemplateListItem>(http.get('/template', { params: paginatorParamsToQuery(params) })),
  get: (id: number) => unwrap<Template>(http.get(`/template/${id}`)),
  create: (name: string) => unwrap<string>(http.post('/template', { name })),

  getParameters: (id: number, params: PaginatorParams) =>
    unwrapPaginated<TemplateParameter>(
      http.get(`/template/${id}/parameters`, { params: paginatorParamsToQuery(params) }),
    ),
  createParameter: (id: number, payload: Omit<TemplateParameterRequest, 'template_id'>) =>
    unwrap<string>(http.post(`/template/${id}/parameters`, { ...payload, template_id: id })),
  updateParameter: (id: number, parameterUuid: string, payload: Omit<TemplateParameterRequest, 'template_id'>) =>
    unwrap<string>(http.post(`/template/${id}/parameters/${parameterUuid}`, { ...payload, template_id: id })),
  deleteParameter: (id: number, parameterUuid: string) =>
    unwrap<string>(http.delete(`/template/${id}/parameters/${parameterUuid}`, { data: { template_id: id } })),
}
