export interface MonthlyPerformanceData {
  month: string
  cost_total: number
  pay_app_total: number
  cumulative_cost: number
  cumulative_pay_app: number
}

export async function fetchMonthlyPerformance(jobNumber: string): Promise<MonthlyPerformanceData[]> {
  const response = await fetch(`/api/jobs/cost-over-time?job=${encodeURIComponent(jobNumber)}`)

  if (!response.ok) {
    throw new Error(`Failed to fetch performance data: ${response.status}`)
  }

  return response.json()
}
