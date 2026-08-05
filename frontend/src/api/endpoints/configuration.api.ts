import { http, unwrap, unwrapPaginated } from '../http'
import { paginatorParamsToQuery, type PaginatorParams } from '../types/paginator'
import type {
  Provision,
  ProvisionSimulateRequest,
  ProvisionSimulateResult,
  ProvisionStoreRequest,
} from '../types/configuration'

export const configurationApi = {
  list: (params: PaginatorParams) =>
    unwrapPaginated<Provision>(http.get('/provision', { params: paginatorParamsToQuery(params) })),
  get: (id: number) => unwrap<Provision>(http.get(`/provision/${id}`)),
  create: (payload: ProvisionStoreRequest) => unwrap<Provision>(http.post('/provision', payload)),
  update: (id: number, payload: ProvisionStoreRequest) => unwrap<Provision>(http.post(`/provision/${id}`, payload)),
  delete: (id: number) => unwrap<string>(http.delete(`/provision/${id}`)),
  clone: (id: number) => unwrap<Provision>(http.get(`/provision/${id}/clone`)),
  updateEnabled: (id: number, enabled: boolean) =>
    unwrap<Provision>(http.patch(`/provision/${id}/enabled`, { enabled })),
  reorder: (ids: number[]) => unwrap<string>(http.patch('/provision', { ids })),
  simulate: (payload: ProvisionSimulateRequest) =>
    unwrap<ProvisionSimulateResult[]>(http.put('/provision/simulate', payload)),
}
