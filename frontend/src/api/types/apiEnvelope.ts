// Matches goacs-go's http/response.ResponseData / ResponseError envelope:
// {"message": "Ok", "data": <payload>}. Validation errors (422) use the same
// shape with data as a map of field -> error message.
export interface ApiEnvelope<T> {
  message: string
  data: T
}

export type ValidationErrors = Record<string, string>
