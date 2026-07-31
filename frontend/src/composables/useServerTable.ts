import { ref, shallowRef } from 'vue'
import type { PaginatorParams, PaginatorResponse } from '@/api/types/paginator'

export interface UseServerTableOptions<T> {
  fetcher: (params: PaginatorParams) => Promise<PaginatorResponse<T>>
  perPage?: number
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
  const filter = ref<Record<string, string>>({})

  let filterTimeout: ReturnType<typeof setTimeout> | undefined

  async function load() {
    loading.value = true
    try {
      const response = await options.fetcher({
        page: page.value,
        per_page: perPage.value,
        filter: filter.value,
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

  function onFilterChange(newFilter: Record<string, string>) {
    filter.value = newFilter
    page.value = 1
    clearTimeout(filterTimeout)
    filterTimeout = setTimeout(load, 300)
  }

  function reload() {
    return load()
  }

  return { items, total, loading, page, perPage, filter, load, onPage, onFilterChange, reload }
}
