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

export interface CPIData {
  month: string
  budget: number
  total_scheduled_qty: string
  cumulative_qty: string
  percent_complete: string
  earned_value: number
  actual_cost: number
  cpi: string
}

export async function fetchCostPerformanceIndex(jobNumber: string): Promise<CPIData[]> {
  const response = await fetch(`/api/jobs/cost-performance-index?job=${encodeURIComponent(jobNumber)}`)

  if (!response.ok) {
    throw new Error(`Failed to fetch CPI data: ${response.status}`)
  }

  return response.json()
}

export interface OverBudgetPhase {
  phase: string
  description: string
  budget: number
  actual_cost: number
  variance: number
}

export async function fetchOverBudgetPhases(jobNumber: string): Promise<OverBudgetPhase[]> {
  const response = await fetch(`/api/jobs/over-budget-phases?job=${encodeURIComponent(jobNumber)}`)

  if (!response.ok) {
    throw new Error(`Failed to fetch over-budget phases: ${response.status}`)
  }

  return response.json()
}
