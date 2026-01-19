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
  ReferenceLine,
} from 'recharts'
import { CPIData, fetchCostPerformanceIndex } from '../api/costData'
import './CostOverTimeChart.css'

interface CPIChartProps {
  jobNumber: string
}

function formatMonth(monthStr: string): string {
  const [year, month] = monthStr.split('-')
  const date = new Date(parseInt(year), parseInt(month) - 1)
  return date.toLocaleDateString('en-US', { month: 'short', year: '2-digit' })
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value)
}

interface ChartData {
  month: string
  cpi: number
  earned_value: number
  actual_cost: number
  percent_complete: number
}

interface TooltipProps {
  active?: boolean
  payload?: Array<{ payload: ChartData }>
  label?: string
}

function CustomTooltip({ active, payload, label }: TooltipProps) {
  if (!active || !payload || !payload.length) return null

  const data = payload[0].payload

  return (
    <div className="custom-tooltip">
      <p className="tooltip-label">{formatMonth(label || '')}</p>
      <p className="tooltip-cpi">
        CPI: <strong>{data.cpi.toFixed(2)}</strong>
        {data.cpi >= 1 ? ' (Under Budget)' : ' (Over Budget)'}
      </p>
      <hr className="tooltip-divider" />
      <p className="tooltip-ev">Earned Value: {formatCurrency(data.earned_value)}</p>
      <p className="tooltip-ac">Actual Cost: {formatCurrency(data.actual_cost)}</p>
      <p className="tooltip-pc">Percent Complete: {data.percent_complete.toFixed(1)}%</p>
    </div>
  )
}

export function CPIChart({ jobNumber }: CPIChartProps) {
  const [data, setData] = useState<ChartData[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!jobNumber) {
      setData([])
      return
    }

    setIsLoading(true)
    setError(null)

    fetchCostPerformanceIndex(jobNumber)
      .then((rawData: CPIData[]) => {
        const chartData = rawData.map((d) => ({
          month: d.month,
          cpi: parseFloat(d.cpi) || 0,
          earned_value: d.earned_value,
          actual_cost: d.actual_cost,
          percent_complete: parseFloat(d.percent_complete) || 0,
        }))
        setData(chartData)
      })
      .catch((err) => setError(err.message))
      .finally(() => setIsLoading(false))
  }, [jobNumber])

  if (!jobNumber) {
    return <div className="chart-placeholder">Select a job to view CPI data</div>
  }

  if (isLoading) {
    return <div className="chart-placeholder">Loading CPI data...</div>
  }

  if (error) {
    return <div className="chart-placeholder error">Error: {error}</div>
  }

  if (data.length === 0) {
    return <div className="chart-placeholder">No CPI data available for this job</div>
  }

  // Calculate Y-axis domain to include 1.0 reference line
  const cpiValues = data.map((d) => d.cpi).filter((v) => v > 0)
  const minCpi = Math.min(...cpiValues, 1)
  const maxCpi = Math.max(...cpiValues, 1)
  const padding = (maxCpi - minCpi) * 0.1 || 0.2
  const yMin = Math.max(0, Math.floor((minCpi - padding) * 10) / 10)
  const yMax = Math.ceil((maxCpi + padding) * 10) / 10

  return (
    <div className="cost-chart-container">
      <h3>Cost Performance Index (CPI)</h3>
      <p style={{ color: '#6b7280', fontSize: '0.875rem', marginTop: '-8px', marginBottom: '16px' }}>
        CPI = Earned Value / Actual Cost (CPI &gt; 1 = under budget, CPI &lt; 1 = over budget)
      </p>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={data} margin={{ top: 20, right: 30, left: 20, bottom: 20 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#e0e0e0" />
          <XAxis
            dataKey="month"
            tickFormatter={formatMonth}
            tick={{ fontSize: 12 }}
            stroke="#666"
          />
          <YAxis
            domain={[yMin, yMax]}
            tick={{ fontSize: 12 }}
            stroke="#666"
            tickFormatter={(value) => value.toFixed(2)}
          />
          <Tooltip content={<CustomTooltip />} />
          <Legend />
          <ReferenceLine
            y={1}
            stroke="#9ca3af"
            strokeDasharray="5 5"
            label={{ value: 'Target (1.0)', position: 'right', fontSize: 11, fill: '#9ca3af' }}
          />
          <Line
            type="monotone"
            dataKey="cpi"
            stroke="#8b5cf6"
            strokeWidth={2}
            dot={{ fill: '#8b5cf6', strokeWidth: 2, r: 4 }}
            activeDot={{ r: 6 }}
            name="CPI"
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  )
}
