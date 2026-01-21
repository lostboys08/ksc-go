import { useState, useEffect } from 'react'
import { MonthlyPerformanceData, fetchMonthlyPerformance } from '../api/costData'
import './PayAppMonthlyList.css'

interface PayAppMonthlyListProps {
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

function formatMonth(monthStr: string): string {
  const [year, month] = monthStr.split('-')
  const date = new Date(parseInt(year), parseInt(month) - 1)
  return date.toLocaleDateString('en-US', { year: 'numeric', month: 'long' })
}

export function PayAppMonthlyList({ jobNumber }: PayAppMonthlyListProps) {
  const [data, setData] = useState<MonthlyPerformanceData[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!jobNumber) {
      setData([])
      return
    }

    setIsLoading(true)
    setError(null)

    fetchMonthlyPerformance(jobNumber)
      .then(setData)
      .catch((err) => setError(err.message))
      .finally(() => setIsLoading(false))
  }, [jobNumber])

  if (!jobNumber) {
    return <div className="pay-app-placeholder">Select a project to view pay applications</div>
  }

  if (isLoading) {
    return <div className="pay-app-placeholder">Loading pay applications...</div>
  }

  if (error) {
    return <div className="pay-app-placeholder error">Error: {error}</div>
  }

  const payAppData = data.filter((d) => d.pay_app_total > 0)

  if (payAppData.length === 0) {
    return (
      <div className="pay-app-container">
        <h3>Monthly Pay Applications</h3>
        <div className="pay-app-empty">No pay application data found for this project</div>
      </div>
    )
  }

  const grandTotal = payAppData.reduce((sum, d) => sum + d.pay_app_total, 0)

  return (
    <div className="pay-app-container">
      <h3>Monthly Pay Applications</h3>
      <p className="pay-app-subtitle">
        {payAppData.length} month{payAppData.length !== 1 ? 's' : ''} with pay app data
      </p>
      <div className="pay-app-table-wrapper">
        <table className="pay-app-table">
          <thead>
            <tr>
              <th>Month</th>
              <th className="text-right">Amount</th>
              <th className="text-right">Cumulative</th>
            </tr>
          </thead>
          <tbody>
            {payAppData.map((item) => (
              <tr key={item.month}>
                <td className="month-cell">{formatMonth(item.month)}</td>
                <td className="text-right">{formatCurrency(item.pay_app_total)}</td>
                <td className="text-right cumulative">{formatCurrency(item.cumulative_pay_app)}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr>
              <td className="total-label">Total</td>
              <td className="text-right total-value">{formatCurrency(grandTotal)}</td>
              <td></td>
            </tr>
          </tfoot>
        </table>
      </div>
    </div>
  )
}
