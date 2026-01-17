export interface UploadResponse {
  success: boolean
  message: string
  filename?: string
  rowsProcessed?: number
}

export type UploadType = 'pay-application'

export async function uploadExcelFile(
  file: File,
  uploadType: UploadType,
  options?: {
    jobNumber?: string
    date?: string // Format: YYYY-MM for pay applications
  }
): Promise<UploadResponse> {
  const formData = new FormData()
  formData.append('file', file)
  formData.append('type', uploadType)

  if (options?.jobNumber) {
    formData.append('jobNumber', options.jobNumber)
  }
  if (options?.date) {
    formData.append('date', options.date)
  }

  const response = await fetch('/api/upload', {
    method: 'POST',
    body: formData,
  })

  if (!response.ok) {
    const errorText = await response.text()
    throw new Error(errorText || `Upload failed with status ${response.status}`)
  }

  return response.json()
}
