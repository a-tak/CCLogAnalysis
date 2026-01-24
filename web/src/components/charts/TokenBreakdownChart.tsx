import { PieChart, Pie, Cell, ResponsiveContainer, Legend, Tooltip } from 'recharts'
import type { TokenSummary } from '@/lib/api/types'

interface TokenBreakdownChartProps {
  tokens: TokenSummary
}

const COLORS = {
  input: 'hsl(221.2 83.2% 53.3%)', // Blue
  output: 'hsl(262 83% 58%)', // Purple
  cacheCreate: 'hsl(25 95% 53%)', // Orange
  cacheRead: 'hsl(215 16.3% 46.9%)', // Gray
}

export function TokenBreakdownChart({ tokens }: TokenBreakdownChartProps) {
  const data = [
    { name: 'Input Tokens', value: tokens.inputTokens, color: COLORS.input },
    { name: 'Output Tokens', value: tokens.outputTokens, color: COLORS.output },
    { name: 'Cache Creation', value: tokens.cacheCreationInputTokens, color: COLORS.cacheCreate },
    { name: 'Cache Read', value: tokens.cacheReadInputTokens, color: COLORS.cacheRead },
  ].filter(item => item.value > 0) // Only show non-zero values

  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center h-[300px] text-muted-foreground">
        No token data available
      </div>
    )
  }

  return (
    <ResponsiveContainer width="100%" height={300}>
      <PieChart>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          labelLine={false}
          label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(1)}%`}
          outerRadius={80}
          fill="#8884d8"
          dataKey="value"
        >
          {data.map((entry, index) => (
            <Cell key={`cell-${index}`} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip formatter={(value) => (value as number).toLocaleString()} />
        <Legend />
      </PieChart>
    </ResponsiveContainer>
  )
}
