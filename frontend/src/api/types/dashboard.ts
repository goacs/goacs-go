export interface Fault {
  uuid: string
  cpe_uuid: string
  serial_number: string
  code: string
  message: string
  created_at: string
}

export interface DashboardData {
  devices_count: number
  online_count: number
  online_offset: number
  faults_count: number
  faults: Fault[]
}
