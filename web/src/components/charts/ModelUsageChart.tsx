import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import type { ModelUsage } from '@/lib/api/types'

interface ModelUsageChartProps {
  modelUsage: ModelUsage[]
}

const COLORS = {
  input: 'hsl(221.2 83.2% 53.3%)', // Blue
  output: 'hsl(262 83% 58%)', // Purple
  cacheCreate: 'hsl(25 95% 53%)', // Orange
  cacheRead: 'hsl(215 16.3% 46.9%)', // Gray
}

export function ModelUsageChart({ modelUsage }: ModelUsageChartProps) {
  const data = modelUsage.map((usage) => ({
    model: usage.model.replace('claude-', '').replace('-20250514', ''), // Shorten model name
    input: usage.tokens.inputTokens,
    output: usage.tokens.outputTokens,
    cacheCreate: usage.tokens.cacheCreationInputTokens,
    cacheRead: usage.tokens.cacheReadInputTokens,
  }))

  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[300px] text-muted-foreground">
        No model usage data available
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <BarChart data={data}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="model" />
        <YAxis />
        <Tooltip formatter={(value) => (value as number).toLocaleString()} />
        <Legend />
        <Bar dataKey="input" stackId="a" fill={COLORS.input} name="Input" />
        <Bar dataKey="output" stackId="a" fill={COLORS.output} name="Output" />
        <Bar dataKey="cacheCreate" stackId="a" fill={COLORS.cacheCreate} name="Cache Create" />
        <Bar dataKey="cacheRead" stackId="a" fill={COLORS.cacheRead} name="Cache Read" />
      </BarChart>
    </ResponsiveContainer>
  )
}
