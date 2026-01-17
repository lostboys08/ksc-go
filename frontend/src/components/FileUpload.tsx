import { useState, useRef, DragEvent, ChangeEvent } from 'react'
import { uploadExcelFile, UploadType, UploadResponse } from '../api/upload'
import './FileUpload.css'

interface FileUploadProps {
  uploadType: UploadType
  jobNumber?: string
  date?: string
  onUploadSuccess?: (response: UploadResponse) => void
  onUploadError?: (error: Error) => void
}

export function FileUpload({
  uploadType,
  jobNumber,
  date,
  onUploadSuccess,
  onUploadError,
}: FileUploadProps) {
  const [isDragging, setIsDragging] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [isUploading, setIsUploading] = useState(false)
  const [uploadStatus, setUploadStatus] = useState<{
    type: 'success' | 'error'
    message: string
  } | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(false)
  }

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    setIsDragging(false)

    const droppedFile = e.dataTransfer.files[0]
    if (droppedFile && isValidExcelFile(droppedFile)) {
      setFile(droppedFile)
      setUploadStatus(null)
    } else {
      setUploadStatus({
        type: 'error',
        message: 'Please upload an Excel file (.xlsx)',
      })
    }
  }

  const handleFileSelect = (e: ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0]
    if (selectedFile && isValidExcelFile(selectedFile)) {
      setFile(selectedFile)
      setUploadStatus(null)
    } else if (selectedFile) {
      setUploadStatus({
        type: 'error',
        message: 'Please upload an Excel file (.xlsx)',
      })
    }
  }

  const isValidExcelFile = (file: File): boolean => {
    return (
      file.type ===
        'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' ||
      file.name.endsWith('.xlsx')
    )
  }

  const handleUpload = async () => {
    if (!file) return

    setIsUploading(true)
    setUploadStatus(null)

    try {
      const response = await uploadExcelFile(file, uploadType, {
        jobNumber,
        date,
      })
      setUploadStatus({
        type: 'success',
        message: response.message || 'File uploaded successfully',
      })
      setFile(null)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
      onUploadSuccess?.(response)
    } catch (error) {
      const errorMessage =
        error instanceof Error ? error.message : 'Upload failed'
      setUploadStatus({
        type: 'error',
        message: errorMessage,
      })
      onUploadError?.(error instanceof Error ? error : new Error(errorMessage))
    } finally {
      setIsUploading(false)
    }
  }

  const handleClear = () => {
    setFile(null)
    setUploadStatus(null)
    if (fileInputRef.current) {
      fileInputRef.current.value = ''
    }
  }

  return (
    <div className="file-upload">
      <div
        className={`drop-zone ${isDragging ? 'dragging' : ''} ${file ? 'has-file' : ''}`}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDrop={handleDrop}
        onClick={() => fileInputRef.current?.click()}
      >
        <input
          ref={fileInputRef}
          type="file"
          accept=".xlsx,application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
          onChange={handleFileSelect}
          className="file-input"
        />
        {file ? (
          <div className="file-info">
            <span className="file-icon">📄</span>
            <span className="file-name">{file.name}</span>
            <span className="file-size">
              ({(file.size / 1024).toFixed(1)} KB)
            </span>
          </div>
        ) : (
          <div className="drop-message">
            <span className="upload-icon">📁</span>
            <p>Drag and drop an Excel file here</p>
            <p className="sub-text">or click to select</p>
          </div>
        )}
      </div>

      {uploadStatus && (
        <div className={`status-message ${uploadStatus.type}`}>
          {uploadStatus.message}
        </div>
      )}

      <div className="actions">
        {file && (
          <>
            <button
              className="btn btn-primary"
              onClick={handleUpload}
              disabled={isUploading}
            >
              {isUploading ? 'Uploading...' : 'Upload'}
            </button>
            <button
              className="btn btn-secondary"
              onClick={handleClear}
              disabled={isUploading}
            >
              Clear
            </button>
          </>
        )}
      </div>
    </div>
  )
}
