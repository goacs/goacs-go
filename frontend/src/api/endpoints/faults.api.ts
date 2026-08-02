import { http, unwrap, unwrapPaginated } from '../http'
import { paginatorParamsToQuery, type PaginatorParams } from '../types/paginator'
import type { LogEntry } from '../types/log'
import type { Fault } from '../types/dashboard'

export const faultsApi = {
  today: () => unwrap<Fault[]>(http.get('/faults/today')),
  list: (params: PaginatorParams) =>
    unwrapPaginated<LogEntry>(http.get('/faults', { params: paginatorParamsToQuery(params) })),
}
