export interface Fault {
  uuid: string
  cpe_uuid: string
  code: string
  message: string
  created_at: string
}

export interface DashboardData {
  devices_count: number
  informs_count: number
  faults_count: number
  faults: Fault[]
}
