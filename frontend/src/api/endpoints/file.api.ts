import { http, unwrap } from '../http'
import type { FileInfo } from '../types/file'

export const fileApi = {
  list: () => unwrap<FileInfo[] | null>(http.get('/file')),
  upload: (file: File, onProgress?: (percent: number) => void) => {
    const form = new FormData()
    form.append('file', file)
    return unwrap<FileInfo>(
      http.post('/file', form, {
        headers: { 'Content-Type': 'multipart/form-data' },
        onUploadProgress: (event) => {
          if (onProgress && event.total) onProgress(Math.round((event.loaded / event.total) * 100))
        },
      }),
    )
  },
  delete: (filename: string) => unwrap<string>(http.delete(`/file/${encodeURIComponent(filename)}`)),
  downloadUrl: (filename: string) =>
    `${import.meta.env.VITE_API_URL.replace(/\/api$/, '')}/file/${encodeURIComponent(filename)}`,
}
