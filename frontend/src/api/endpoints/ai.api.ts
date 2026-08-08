import { http, unwrap } from '../http'
import type { GenerateScriptRequest, GenerateScriptResponse } from '../types/ai'

export const aiApi = {
  generateScript: (payload: GenerateScriptRequest) =>
    unwrap<GenerateScriptResponse>(
      http.post('/ai/script', {
        prompt: payload.prompt,
        events: payload.events,
        requests: payload.requests,
        current_script: payload.currentScript,
      }),
    ),
}
