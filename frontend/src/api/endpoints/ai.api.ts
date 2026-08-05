import { http, unwrap } from '../http'
import type { GenerateScriptRequest, GenerateScriptResponse } from '../types/ai'

export const aiApi = {
  generateScript: (payload: GenerateScriptRequest) =>
    unwrap<GenerateScriptResponse>(http.post('/ai/script', payload)),
}
