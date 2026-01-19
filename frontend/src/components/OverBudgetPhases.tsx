import { useState, useEffect } from 'react'
import { OverBudgetPhase, fetchOverBudgetPhases } from '../api/costData'
import './OverBudgetPhases.css'

interface OverBudgetPhasesProps {
  jobNumber: string
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

export function OverBudgetPhases({ jobNumber }: OverBudgetPhasesProps) {
  const [data, setData] = useState<OverBudgetPhase[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!jobNumber) {
      setData([])
      return
    }

    setIsLoading(true)
    setError(null)

    fetchOverBudgetPhases(jobNumber)
      .then(setData)
      .catch((err) => setError(err.message))
      .finally(() => setIsLoading(false))
  }, [jobNumber])

  if (!jobNumber) {
    return <div className="over-budget-placeholder">Select a job to view over-budget phases</div>
  }

  if (isLoading) {
    return <div className="over-budget-placeholder">Loading over-budget phases...</div>
  }

  if (error) {
    return <div className="over-budget-placeholder error">Error: {error}</div>
  }

  if (data.length === 0) {
    return (
      <div className="over-budget-container">
        <h3>Over Budget Phases</h3>
        <div className="over-budget-success">
          All phase codes are within budget
        </div>
      </div>
    )
  }

  return (
    <div className="over-budget-container">
      <h3>Over Budget Phases</h3>
      <p className="over-budget-subtitle">
        {data.length} phase{data.length !== 1 ? 's' : ''} exceeding budget
      </p>
      <div className="over-budget-table-wrapper">
        <table className="over-budget-table">
          <thead>
            <tr>
              <th>Phase</th>
              <th>Description</th>
              <th className="text-right">Budget</th>
              <th className="text-right">Actual</th>
              <th className="text-right">Over By</th>
            </tr>
          </thead>
          <tbody>
            {data.map((phase) => (
              <tr key={phase.phase}>
                <td className="phase-code">{phase.phase}</td>
                <td className="phase-description">{phase.description || '-'}</td>
                <td className="text-right">{formatCurrency(phase.budget)}</td>
                <td className="text-right">{formatCurrency(phase.actual_cost)}</td>
                <td className="text-right variance-negative">
                  {formatCurrency(phase.variance)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
