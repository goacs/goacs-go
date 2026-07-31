// Matches goacs-go's repository.PaginatorResponse (repository/paginator.go).
export interface PaginatorResponse<T> {
  page: number
  per_page: number
  filter: Record<string, string>
  next_page: number
  prev_page: number
  total: number
  data: T[]
}

export interface PaginatorParams {
  page?: number
  per_page?: number
  filter?: Record<string, string>
  sort?: Record<string, 'asc' | 'desc'>
}

// Serializes to Go's expected query format: filter[key]=value, sort[key]=asc|desc.
export function paginatorParamsToQuery(params: PaginatorParams): Record<string, string> {
  const query: Record<string, string> = {}

  if (params.page) query['page'] = String(params.page)
  if (params.per_page) query['per_page'] = String(params.per_page)

  for (const [key, value] of Object.entries(params.filter ?? {})) {
    if (value === '' || value === undefined || value === null) continue
    query[`filter[${key}]`] = value
  }

  for (const [key, value] of Object.entries(params.sort ?? {})) {
    query[`sort[${key}]`] = value
  }

  return query
}
