import { ref, shallowRef } from 'vue'
import type { PaginatorParams, PaginatorResponse } from '@/api/types/paginator'
import type { DataTableFilterMeta, DataTableFilterMetaData } from 'primevue/datatable'

export interface UseServerTableOptions<T> {
  fetcher: (params: PaginatorParams) => Promise<PaginatorResponse<T>>
  perPage?: number
  // Seeds PrimeVue's own filter model (bind via v-model:filters, filterDisplay="row")
  // so columns use the built-in per-column filter row instead of ad-hoc inputs.
  filters?: DataTableFilterMeta
}

// Drives a server-paginated/filtered PrimeVue DataTable against goacs-go's
// PaginatorResponse shape (repository/paginator.go). Sorting is intentionally
// not wired up: the Go paginator only understands page/per_page/filter today.
export function useServerTable<T>(options: UseServerTableOptions<T>) {
  const items = shallowRef<T[]>([])
  const total = ref(0)
  const loading = ref(false)
  const page = ref(1)
  const perPage = ref(options.perPage ?? 25)
  const filters = ref<DataTableFilterMeta>(options.filters ?? {})

  let filterTimeout: ReturnType<typeof setTimeout> | undefined

  // Flattens PrimeVue's { field: { value, matchMode } } filter model into the
  // plain key/value map goacs-go's paginator expects (filter[key]=value); the
  // backend always does a substring match, so matchMode is UI-only metadata.
  function filterParams(): Record<string, string> {
    const flat: Record<string, string> = {}
    for (const [key, meta] of Object.entries(filters.value)) {
      const value = (meta as DataTableFilterMetaData).value
      if (value !== null && value !== undefined && value !== '') flat[key] = String(value)
    }
    return flat
  }

  async function load() {
    loading.value = true
    try {
      const response = await options.fetcher({
        page: page.value,
        per_page: perPage.value,
        filter: filterParams(),
      })
      items.value = response.data ?? []
      total.value = response.total
    } finally {
      loading.value = false
    }
  }

  function onPage(event: { page: number; rows: number }) {
    page.value = event.page + 1
    perPage.value = event.rows
    load()
  }

  function onFilter() {
    page.value = 1
    clearTimeout(filterTimeout)
    filterTimeout = setTimeout(load, 300)
  }

  function reload() {
    return load()
  }

  return { items, total, loading, page, perPage, filters, load, onPage, onFilter, reload }
}
