export interface GenerateScriptRequest {
  prompt: string
  events?: string
  requests?: string
}

export interface GenerateScriptResponse {
  script: string
  explanation: string
}
