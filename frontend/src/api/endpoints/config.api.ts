import { http, unwrap } from '../http'
import type { ConfigValues, DebugSettings, SaveDebugSettingsRequest } from '../types/settings'

export const configApi = {
  get: () => unwrap<ConfigValues>(http.get('/settings')),
  save: (config: ConfigValues) => unwrap<string>(http.post('/settings', { config })),

  getDebug: () => unwrap<DebugSettings>(http.get('/settings/debug')),
  saveDebug: (payload: SaveDebugSettingsRequest) => unwrap<string>(http.post('/settings/debug', payload)),
}
