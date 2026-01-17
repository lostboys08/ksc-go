import { useState, useEffect } from 'react'
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { MonthlyPerformanceData, fetchMonthlyPerformance } from '../api/costData'
import './CostOverTimeChart.css'

interface CostOverTimeChartProps {
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
  return date.toLocaleDateString('en-US', { month: 'short', year: '2-digit' })
}

function formatPercent(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'percent',
    minimumFractionDigits: 1,
    maximumFractionDigits: 1,
  }).format(value)
}

interface TooltipProps {
  active?: boolean
  payload?: Array<{ payload: MonthlyPerformanceData }>
  label?: string
}

function CustomTooltip({ active, payload, label }: TooltipProps) {
  if (!active || !payload || !payload.length) return null

  const data = payload[0].payload
  const cost = data.cumulative_cost
  const revenue = data.cumulative_pay_app
  const grossProfit = revenue - cost
  const markup = cost !== 0 ? grossProfit / cost : 0
  const margin = revenue !== 0 ? grossProfit / revenue : 0

  return (
    <div className="custom-tooltip">
      <p className="tooltip-label">{formatMonth(label || '')}</p>
      <p className="tooltip-cost">Total Cost: {formatCurrency(cost)}</p>
      <p className="tooltip-revenue">Total Pay Apps: {formatCurrency(revenue)}</p>
      <p className="tooltip-profit">Gross Profit: {formatCurrency(grossProfit)}</p>
      <hr className="tooltip-divider" />
      <p className="tooltip-markup">Markup: {formatPercent(markup)}</p>
      <p className="tooltip-margin">Margin: {formatPercent(margin)}</p>
    </div>
  )
}

export function CostOverTimeChart({ jobNumber }: CostOverTimeChartProps) {
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
    return <div className="chart-placeholder">Select a job to view performance data</div>
  }

  if (isLoading) {
    return <div className="chart-placeholder">Loading performance data...</div>
  }

  if (error) {
    return <div className="chart-placeholder error">Error: {error}</div>
  }

  if (data.length === 0) {
    return <div className="chart-placeholder">No performance data available for this job</div>
  }

  return (
    <div className="cost-chart-container">
      <h3>Monthly Performance: Cost vs Pay Applications</h3>
      <ResponsiveContainer width="100%" height={400}>
        <LineChart data={data} margin={{ top: 20, right: 30, left: 20, bottom: 20 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
          <XAxis
            dataKey="month"
            tickFormatter={formatMonth}
            tick={{ fontSize: 12 }}
            stroke="#666"
          />
          <YAxis
            tickFormatter={(value) => formatCurrency(value)}
            tick={{ fontSize: 12 }}
            stroke="#666"
            width={100}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend />
          <Line
            type="monotone"
            dataKey="cumulative_cost"
            stroke="#dc2626"
            strokeWidth={2}
            dot={{ fill: '#dc2626', strokeWidth: 2, r: 4 }}
            activeDot={{ r: 6 }}
            name="Total Cost"
          />
          <Line
            type="monotone"
            dataKey="cumulative_pay_app"
            stroke="#2563eb"
            strokeWidth={2}
            dot={{ fill: '#2563eb', strokeWidth: 2, r: 4 }}
            activeDot={{ r: 6 }}
            name="Total Pay Applications"
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
