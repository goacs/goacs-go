export const LOG_TYPES = ['FAULT', 'INFO', 'ERROR', 'REQUEST', 'RESPONSE'] as const

export interface LogEntry {
  id: number
  cpe_uuid: string
  full_xml: string
  code: string
  message: string
  type: string
  from: string
  session_id: string
  detail: string | null
  created_at: string
}
