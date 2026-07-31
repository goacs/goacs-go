import { ref } from 'vue'
import { ApiValidationError } from '@/api/http'

// Maps goacs-go's 422 {message, data: {field: message}} shape onto form field
// errors, so a form can render inline messages without each caller
// re-implementing the same try/catch.
export function useApiErrors() {
  const fieldErrors = ref<Record<string, string>>({})
  const generalError = ref<string | null>(null)

  function clear() {
    fieldErrors.value = {}
    generalError.value = null
  }

  async function run<T>(action: () => Promise<T>): Promise<T | undefined> {
    clear()
    try {
      return await action()
    } catch (error) {
      if (error instanceof ApiValidationError) {
        fieldErrors.value = error.fields
      } else {
        generalError.value = 'Something went wrong. Please try again.'
      }
      return undefined
    }
  }

  return { fieldErrors, generalError, clear, run }
}
