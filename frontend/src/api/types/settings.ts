import type { CPE } from './device'

export type ConfigValues = Record<string, string>

export interface DebugSettings {
  debug: boolean
  debug_new_devices: boolean
  devices: CPE[]
}

export interface SaveDebugSettingsRequest {
  debug: boolean
  debug_new_devices: boolean
  devices: string[]
}
