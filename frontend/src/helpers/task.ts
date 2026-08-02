// Builds the JSON-encoded payload string that AddTaskRequest.payload expects
// (goacs-go unmarshals it straight into tasks.Task.Payload), one shape per
// task type - mirrors goacs-php's helpers/Task.js.
export function payloadForTaskType(taskType: string, form: Record<string, string>): string {
  switch (taskType) {
    case 'RunScript':
      return JSON.stringify({ script: form.script ?? '' })
    case 'UploadFirmware':
      return JSON.stringify({ filename: form.filename ?? '', filetype: 'firmware' })
    case 'AddObject':
    case 'DeleteObject':
      return JSON.stringify({ path: form.path ?? '' })
    default:
      return JSON.stringify({})
  }
}
