import { useState, useEffect, useRef } from 'react'
import { Job, fetchJobs } from '../api/jobs'
import './JobComboBox.css'

interface JobComboBoxProps {
  value: string
  onChange: (jobNumber: string) => void
}

function fuzzyMatch(text: string, query: string): boolean {
  const lowerText = text.toLowerCase()
  const lowerQuery = query.toLowerCase()

  let queryIndex = 0
  for (let i = 0; i < lowerText.length && queryIndex < lowerQuery.length; i++) {
    if (lowerText[i] === lowerQuery[queryIndex]) {
      queryIndex++
    }
  }
  return queryIndex === lowerQuery.length
}

export function JobComboBox({ value, onChange }: JobComboBoxProps) {
  const [jobs, setJobs] = useState<Job[]>([])
  const [isOpen, setIsOpen] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    setIsLoading(true)
    fetchJobs()
      .then(setJobs)
      .catch((err) => setError(err.message))
      .finally(() => setIsLoading(false))
  }, [])

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  const filteredJobs = value
    ? jobs.filter((job) =>
        fuzzyMatch(job.job_number, value) || fuzzyMatch(job.job_name, value)
      )
    : jobs

  const handleSelect = (job: Job) => {
    onChange(job.job_number)
    setIsOpen(false)
  }

  return (
    <div className="job-combobox" ref={containerRef}>
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onFocus={() => setIsOpen(true)}
        placeholder="Type to search jobs..."
        className="job-combobox-input"
      />
      {isOpen && (
        <div className="job-combobox-dropdown">
          {isLoading && <div className="job-combobox-message">Loading...</div>}
          {error && <div className="job-combobox-message error">{error}</div>}
          {!isLoading && !error && filteredJobs.length === 0 && (
            <div className="job-combobox-message">No jobs found</div>
          )}
          {!isLoading && !error && filteredJobs.map((job) => (
            <div
              key={job.id}
              className="job-combobox-option"
              onClick={() => handleSelect(job)}
            >
              <span className="job-number">{job.job_number}</span>
              <span className="job-name">{job.job_name}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
