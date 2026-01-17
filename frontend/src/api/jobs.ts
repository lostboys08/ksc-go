export interface Job {
  id: string
  job_number: string
  job_name: string
}

export async function fetchJobs(): Promise<Job[]> {
  const response = await fetch('/api/jobs')

  if (!response.ok) {
    throw new Error(`Failed to fetch jobs: ${response.status}`)
  }

  return response.json()
}
