import { http, unwrap } from '../http'
import type { DashboardData } from '../types/dashboard'

export const dashboardApi = {
  get: () => unwrap<DashboardData>(http.get('/dashboard')),
}
