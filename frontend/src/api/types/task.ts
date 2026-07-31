export const TASK_TYPES = ['RunScript', 'UploadFirmware', 'Reboot', 'FactoryReset', 'AddObject', 'DeleteObject'] as const

export interface Task {
  id: number
  for_name: string
  for_id: string
  event: string
  not_before: string
  task: string
  payload: Record<string, unknown>
  infinite: boolean
  created_at: string
  done_at: string | null
}

export interface AddTaskRequest {
  event: string
  task: string
  payload: string
}
